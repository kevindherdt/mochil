package web

import (
	"context"
	"strings"

	"encoding/json"
	"mochil/internal/catalog"
	"net/http"
	"testing"
	"time"
)

func waitForCheck(t *testing.T, server *multiServer) map[string]any {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		var status map[string]any
		_ = json.Unmarshal(editorCall(server, http.MethodGet, "/api/metadata/editor/check?library=books", "").Body.Bytes(), &status)
		if status["status"] == "ready" || status["status"] == "failed" {
			return status
		}
	}
	t.Fatal("the check did not finish")
	return nil
}

func TestTheQualityCheckFindsMissingCoversAndUnreadableFilesWithoutChangingAnything(t *testing.T) {
	server := countServer(t)
	before := listEditor(t, server, "")
	if before.Counts["no_cover"] != 0 || before.Counts["unreadable"] != 0 {
		t.Fatalf("nothing is known before the check: %v", before.Counts)
	}
	if before.Counts["no_description"] == 0 {
		t.Fatalf("a missing description needs no file reading: %v", before.Counts)
	}
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/check", `{"library":"books"}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	if status := waitForCheck(t, server); status["status"] != "ready" {
		t.Fatalf("status = %v", status)
	}
	after := listEditor(t, server, "")
	if after.Counts["no_cover"] != 1 {
		t.Fatalf("book a1 is an EPUB with no cover: %v", after.Counts)
	}
	if after.Counts["unreadable"] != len(server.libraries[0].Store.Books())-1 {
		t.Fatalf("every other book has no file: %v", after.Counts)
	}
	if got := listEditor(t, server, "&filter=no_cover"); got.Total != 1 {
		t.Fatalf("no_cover filter = %d", got.Total)
	}
	if edited := listEditor(t, server, "&filter=edited"); edited.Total != 0 {
		t.Fatal("checking must not edit anything")
	}
}

func TestQualityFindingsDoNotInflateNeedsReview(t *testing.T) {
	server := editorTestServer(t)
	plain := listEditor(t, server, "&filter=issues")
	if plain.Total == len(server.libraries[0].Store.Books()) {
		t.Fatalf("every book lacks a description or ISBN in this fixture, yet Needs review must not list them all: %d", plain.Total)
	}
}

// stuckBackend never finishes opening a book until the job is stopped.
type stuckBackend struct {
	staticBackend
	opening chan struct{}
}

func (b stuckBackend) Acquire(ctx context.Context, _ catalog.Book) (catalog.Acquisition, error) {
	select {
	case b.opening <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return catalog.Acquisition{}, ctx.Err()
}

func TestAnActionThatTakesTooLongCanBeStoppedAndStartedAgain(t *testing.T) {
	server := countServer(t)
	opening := make(chan struct{}, 1)
	books := server.libraries[0].Store.Books()
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", stuckBackend{staticBackend: staticBackend{books: books}, opening: opening})
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/check", `{"library":"books"}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d", response.Code)
	}
	select {
	case <-opening:
	case <-time.After(5 * time.Second):
		t.Fatal("the check never started on a book")
	}
	stop := editorCall(server, http.MethodDelete, "/api/metadata/editor/check?library=books", "")
	if stop.Code != http.StatusOK || !strings.Contains(stop.Body.String(), `"stopped":true`) {
		t.Fatalf("stop: %d %s", stop.Code, stop.Body)
	}
	var status map[string]any
	_ = json.Unmarshal(editorCall(server, http.MethodGet, "/api/metadata/editor/check?library=books", "").Body.Bytes(), &status)
	if status["status"] != "canceled" {
		t.Fatalf("status after stopping = %v", status)
	}
	if again := editorCall(server, http.MethodDelete, "/api/metadata/editor/check?library=books", ""); !strings.Contains(again.Body.String(), `"stopped":false`) {
		t.Fatalf("nothing is running any more: %s", again.Body)
	}
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if response := editorCall(server, http.MethodPost, "/api/metadata/editor/check", `{"library":"books"}`); response.Code == http.StatusAccepted {
			editorCall(server, http.MethodDelete, "/api/metadata/editor/check?library=books", "")
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("after stopping, the action could not be started again")
		}
	}
}

func TestCountingCanBeStoppedToo(t *testing.T) {
	server := countServer(t)
	opening := make(chan struct{}, 1)
	books := server.libraries[0].Store.Books()
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", stuckBackend{staticBackend: staticBackend{books: books}, opening: opening})
	editorCall(server, http.MethodPost, "/api/metadata/editor/count", `{"library":"books"}`)
	<-opening
	stop := editorCall(server, http.MethodDelete, "/api/metadata/editor/count?library=books", "")
	if !strings.Contains(stop.Body.String(), `"stopped":true`) {
		t.Fatalf("stop: %s", stop.Body)
	}
}

func TestSortingByISBNBringsEitherGroupToTheTop(t *testing.T) {
	rows := []editorRow{
		{ID: "none", Title: "A", Identifiers: "hardcover:dune"},
		{ID: "late", Title: "B", Identifiers: "goodreads:1, isbn:9780743273565"},
		{ID: "early", Title: "C", Identifiers: "isbn:9780306406157"},
		{ID: "empty", Title: "D"},
	}
	order := func(descending bool) string {
		copied := append([]editorRow{}, rows...)
		sortEditorRows(copied, "isbn", descending)
		ids := ""
		for _, row := range copied {
			ids += row.ID + " "
		}
		return ids
	}
	if got := order(false); got != "early late none empty " {
		t.Fatalf("ascending = %q: books with an ISBN first, by ISBN", got)
	}
	if got := order(true); got != "empty none late early " {
		t.Fatalf("descending = %q: books without an ISBN first", got)
	}
}
