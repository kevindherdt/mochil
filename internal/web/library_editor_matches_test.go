package web

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/enrichment"
	"mochil/internal/filefacts"
	"mochil/internal/jobs"
)

// matchHardcover plays Hardcover: ISBN 9780306406157 is Dune (book 5); a title search finds Dune Messiah (book 6)
// and, for "Neuromancer", two equally good books. requests counts every request made.
func matchHardcover(requests *atomic.Int32) *enrichment.Hardcover {
	return enrichment.NewHardcover("token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		requests.Add(1)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "EditionsByISBN"):
			response = `{"data":{"editions":[{"isbn_13":"9780306406157","book":{"id":5,"title":"Dune","description":"Spice.","release_year":1965,"slug":"dune","contributions":[{"author":{"name":"Frank Herbert"}}]}}]}}`
		case strings.Contains(body, "SearchBooks") && strings.Contains(body, "Neuromancer"):
			response = `{"data":{"search":{"ids":[8,9]}}}`
		case strings.Contains(body, "SearchBooks"):
			response = `{"data":{"search":{"ids":[6]}}}`
		case strings.Contains(body, "BookDetails") && strings.Contains(body, "8"):
			response = `{"data":{"books":[{"id":8,"title":"Neuromancer","slug":"n","contributions":[{"author":{"name":"William Gibson"}}]},{"id":9,"title":"Neuromancer: The Graphic Novel","slug":"n2","contributions":[{"author":{"name":"William Gibson"}}]}]}}`
		case strings.Contains(body, "BookDetails"):
			response = `{"data":{"books":[{"id":6,"title":"Dune Messiah","description":"More spice.","release_year":1969,"slug":"dm","contributions":[{"author":{"name":"Frank Herbert"}}]}]}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
}

func matchServer(t *testing.T) (*multiServer, *atomic.Int32) {
	t.Helper()
	quickHardcoverTimings(t)
	server := editorTestServer(t) // Dune a1, Dune Messiah a2, Neuromancer n1, ...
	requests := &atomic.Int32{}
	server.hardcover = matchHardcover(requests)
	server.cfg.Modules = map[string]bool{"hardcover": true}
	store, err := filefacts.Open[bulkMatch](filepath.Join(t.TempDir(), "hardcover-found.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.matchFound = store
	server.backgroundJobs = jobs.New(2, 64)
	t.Cleanup(server.backgroundJobs.Close)
	return server, requests
}

func waitForMatches(t *testing.T, server *multiServer) map[string]any {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		var status map[string]any
		_ = json.Unmarshal(editorCall(server, http.MethodGet, "/api/metadata/editor/matches?library=books", "").Body.Bytes(), &status)
		if status["status"] == "ready" || status["status"] == "failed" {
			return status
		}
	}
	t.Fatal("matching did not finish")
	return nil
}

func TestMatchingFindsBooksByISBNExactlyAndByTitleOtherwiseAndAppliesNothing(t *testing.T) {
	server, _ := matchServer(t)
	// a1 carries its ISBN; a2 and n1 have none and are found by title.
	books := server.libraries[0].Store.Books()
	for i := range books {
		if books[i].ID == "a1" {
			books[i].Identifiers = map[string]string{"isbn": "978-0-306-40615-7"}
		}
	}
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", staticBackend{books: books})
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/matches", `{"library":"books","ids":["a1","a2","n1"]}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	status := waitForMatches(t, server)
	if status["status"] != "ready" || !strings.Contains(status["message"].(string), "1 by ISBN") {
		t.Fatalf("status = %v", status)
	}
	byID := map[string]editorRow{}
	for _, row := range listEditor(t, server, "").Items {
		byID[row.ID] = row
	}
	if row := byID["a1"]; row.Match == nil || row.Match.ProviderID != "5" || !row.MatchClear || !strings.Contains(row.MatchNote, "exact") {
		t.Fatalf("a1 is Dune, exact by ISBN: %+v", row)
	}
	if row := byID["a2"]; row.Match == nil || row.Match.ProviderID != "6" || !row.MatchClear {
		t.Fatalf("a2 is Dune Messiah by title: %+v", row)
	}
	if row := byID["n1"]; row.Match == nil || row.MatchClear || !strings.Contains(row.MatchNote, "Not certain") {
		t.Fatalf("n1 has two close candidates and must not be called clear: %+v", row)
	}
	if got := listEditor(t, server, "&filter=match_unsure"); got.Total != 1 || got.Items[0].ID != "n1" {
		t.Fatalf("unsure filter = %+v", got)
	}
	if edited := listEditor(t, server, "&filter=edited"); edited.Total != 0 {
		t.Fatal("matching must apply nothing")
	}
}

func TestMatchingLeavesBooksAlreadyLinkedAndAlreadyLookedUpAlone(t *testing.T) {
	server, requests := matchServer(t)
	books := server.libraries[0].Store.Books()
	for i := range books {
		if books[i].ID == "a1" {
			books[i].Identifiers = map[string]string{"hardcover": "5"} // linked already
		}
	}
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", staticBackend{books: books})
	var n1 catalog.Book
	for _, book := range books {
		if book.ID == "n1" {
			n1 = book
		}
	}
	sentinel := bulkMatch{How: "title", Clear: true, Candidates: []matchCandidate{{Suggestion: enrichment.Suggestion{Title: "Looked up earlier", ProviderID: "77"}, Score: 1}}}
	server.matchFound.Put("books", "n1", n1.Size, n1.Modified, sentinel) // looked up already
	editorCall(server, http.MethodPost, "/api/metadata/editor/matches", `{"library":"books","ids":["a1","a2","n1"]}`)
	waitForMatches(t, server)
	if got := requests.Load(); got != 2 {
		t.Fatalf("only a2 needs Hardcover (a search and its details): %d requests", got)
	}
	if got := listEditor(t, server, "&filter=match_found"); got.Total != 2 {
		t.Fatalf("a2 is new and n1 kept what it had: %+v", got)
	}
	for _, row := range listEditor(t, server, "").Items {
		switch row.ID {
		case "a1":
			if row.Match != nil {
				t.Fatal("a1 is linked already and must not be offered a match")
			}
		case "n1":
			if row.Match == nil || row.Match.ProviderID != "77" {
				t.Fatalf("a lookup made earlier must not be repeated: %+v", row.Match)
			}
		}
	}
}

func TestMatchingNeedsTheHardcoverModuleAndCanBeStopped(t *testing.T) {
	server, _ := matchServer(t)
	server.cfg.Modules = map[string]bool{"hardcover": false}
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/matches", `{"library":"books"}`); response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "Turn on the Hardcover module") {
		t.Fatalf("%d %s", response.Code, response.Body)
	}
	server.cfg.Modules = map[string]bool{"hardcover": true}
	hardcoverTimingMu.Lock()
	hardcoverTimingIs.requestGap = time.Minute // each pause lasts until the job is stopped
	hardcoverTimingMu.Unlock()
	editorCall(server, http.MethodPost, "/api/metadata/editor/matches", `{"library":"books"}`)
	time.Sleep(100 * time.Millisecond)
	if stop := editorCall(server, http.MethodDelete, "/api/metadata/editor/matches?library=books", ""); !strings.Contains(stop.Body.String(), `"stopped":true`) {
		t.Fatalf("stop: %s", stop.Body)
	}
}
