package web

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/bookcheck"
	"mochil/internal/bookstats"
	"mochil/internal/catalog"
	"mochil/internal/jobs"
)

func epubBytes(t *testing.T, words string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	files := map[string]string{
		"META-INF/container.xml": `<container><rootfiles><rootfile full-path="c.opf"/></rootfiles></container>`,
		"c.opf":                  `<package><manifest><item id="a" href="a.xhtml"/></manifest><spine><itemref idref="a"/></spine></package>`,
		"a.xhtml":                `<html><body><p>` + words + `</p></body></html>`,
	}
	for name, body := range files {
		writer, _ := archive.Create(name)
		_, _ = writer.Write([]byte(body))
	}
	_ = archive.Close()
	return buffer.Bytes()
}

type readableFile struct{ *bytes.Reader }

func (readableFile) Close() error { return nil }

// epubBackend serves a real EPUB for book a1 only; every other book has no file to read.
type epubBackend struct {
	staticBackend
	file []byte
}

func (b epubBackend) Acquire(_ context.Context, book catalog.Book) (catalog.Acquisition, error) {
	if book.ID != "a1" {
		return catalog.Acquisition{}, http.ErrMissingFile
	}
	return catalog.Acquisition{Reader: readableFile{bytes.NewReader(b.file)}}, nil
}

func countServer(t *testing.T) *multiServer {
	t.Helper()
	server := editorTestServer(t)
	books := server.libraries[0].Store.Books()
	for i := range books {
		books[i].Format = "epub"
		books[i].Size = 1
	}
	backend := epubBackend{staticBackend: staticBackend{books: books}, file: epubBytes(t, strings.Repeat("word ", 600))}
	overrides := server.metadataOverrides
	server.libraries[0].Store = overrides.Wrap("books", backend)
	store, err := bookstats.Open(filepath.Join(t.TempDir(), "book-stats.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.bookStats = store
	if server.bookQuality, err = bookcheck.Open(filepath.Join(t.TempDir(), "book-quality.json")); err != nil {
		t.Fatal(err)
	}
	server.backgroundJobs = jobs.New(2, 64)
	t.Cleanup(server.backgroundJobs.Close)
	return server
}

func waitForCounting(t *testing.T, server *multiServer) map[string]any {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		var status map[string]any
		_ = json.Unmarshal(editorCall(server, http.MethodGet, "/api/metadata/editor/count?library=books", "").Body.Bytes(), &status)
		if status["status"] == "ready" || status["status"] == "failed" {
			return status
		}
	}
	t.Fatal("counting did not finish")
	return nil
}

func TestCountingFillsTheWordsAndPagesColumnsAndSurvivesBooksThatCannotBeRead(t *testing.T) {
	server := countServer(t)
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/count", `{"library":"books"}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	status := waitForCounting(t, server)
	if status["status"] != "ready" || !strings.Contains(status["message"].(string), "Counted 1 books") || !strings.Contains(status["message"].(string), "4 could not be read") {
		t.Fatalf("status = %v", status)
	}
	rows := listEditor(t, server, "&sort=words&dir=desc").Items
	if rows[0].ID != "a1" || rows[0].Words != 600 || rows[0].Pages != 3 {
		t.Fatalf("first row = %+v, want Dune with 600 words and 3 pages on top", rows[0])
	}
	if rows[1].Words != 0 {
		t.Fatalf("an uncounted book has no numbers: %+v", rows[1])
	}
}

func TestCountingOnlyTheChosenBooksAndNotAgainUnlessAsked(t *testing.T) {
	server := countServer(t)
	editorCall(server, http.MethodPost, "/api/metadata/editor/count", `{"library":"books","ids":["a2"]}`)
	status := waitForCounting(t, server)
	if !strings.Contains(status["message"].(string), "Counted 0 books") {
		t.Fatalf("a2 has no readable file, so nothing is counted: %v", status)
	}
	editorCall(server, http.MethodPost, "/api/metadata/editor/count", `{"library":"books","ids":["a1"]}`)
	waitForCounting(t, server)
	before := listEditor(t, server, "&q=dune").Items[0].Words
	// The file is now unreadable; a plain run keeps the stored count, a recount would have to skip it.
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", epubBackend{staticBackend: staticBackend{books: server.libraries[0].Store.Books()}})
	editorCall(server, http.MethodPost, "/api/metadata/editor/count", `{"library":"books","ids":["a1"]}`)
	waitForCounting(t, server)
	if after := listEditor(t, server, "&q=dune").Items[0].Words; after != before || before != 600 {
		t.Fatalf("words before=%d after=%d, want the stored 600 kept", before, after)
	}
}
