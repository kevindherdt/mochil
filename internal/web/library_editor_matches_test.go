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
			response = `{"data":{"books":[{"id":6,"title":"Dune Messiah","description":"More spice.","release_year":1969,"slug":"dm","contributions":[{"author":{"name":"Frank Herbert"}}],"book_series":[{"position":2,"featured":true,"series":{"name":"Dune"}}]}]}}`
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
	// The row the review panel reads has to carry the series, or "Replace from the match" can fill a series but never
	// rename one — which is the case it exists for.
	if row := byID["a2"]; row.Match == nil || row.Match.Series != "Dune" || row.Match.SeriesIndex == nil || *row.Match.SeriesIndex != 2 {
		t.Fatalf("a2's match must carry its series and position: %+v", row.Match)
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
	// A finding in the shape this version writes: complete, so there is nothing left to ask Hardcover about.
	sentinel := bulkMatch{Version: matchFindingVersion, How: "title", Clear: true, Candidates: []matchCandidate{{Suggestion: enrichment.Suggestion{Title: "Looked up earlier", ProviderID: "77"}, Score: 1}}}
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

// A finding stored before the editor asked Hardcover for a series cannot answer "which series is this?" — and it looks
// exactly like a finding where Hardcover genuinely has no series. It is looked up again rather than shown as complete,
// which is what makes the series of an already-matched library fixable at all.
func TestMatchingRefreshesAFindingFromBeforeSeriesWereAsked(t *testing.T) {
	server, requests := matchServer(t)
	var n1 catalog.Book
	for _, book := range server.libraries[0].Store.Books() {
		if book.ID == "n1" {
			n1 = book
		}
	}
	incomplete := bulkMatch{How: "title", Clear: true, Candidates: []matchCandidate{{Suggestion: enrichment.Suggestion{Title: "Looked up earlier", ProviderID: "77"}, Score: 1}}}
	server.matchFound.Put("books", "n1", n1.Size, n1.Modified, incomplete) // no Version: written by the build before series
	editorCall(server, http.MethodPost, "/api/metadata/editor/matches", `{"library":"books","ids":["n1"]}`)
	waitForMatches(t, server)
	if requests.Load() == 0 {
		t.Fatal("an incomplete finding was trusted and never looked up again")
	}
	for _, row := range listEditor(t, server, "").Items {
		if row.ID == "n1" && (row.Match == nil || row.Match.ProviderID == "77") {
			t.Fatalf("n1 still carries the old finding: %+v", row.Match)
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

// Applying a match is what links the book to Hardcover, and the link used to be the cue to stop showing the finding —
// so the series disappeared at the moment it had been used, and "Replace from the match" would only ever work on the
// first pass. The finding stays readable; only the badge that means "needs a look" stops.
func TestAMatchedBookKeepsItsFindingOnceItIsLinked(t *testing.T) {
	server, _ := matchServer(t)
	books := server.libraries[0].Store.Books()
	var a2 catalog.Book
	for i := range books {
		if books[i].ID == "a2" {
			a2 = books[i]
			books[i].Identifiers = map[string]string{"isbn": "9780593098271", "hardcover": "6"}
		}
	}
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", staticBackend{books: books})
	position := 2.0
	server.matchFound.Put("books", "a2", a2.Size, a2.Modified, bulkMatch{
		Version: matchFindingVersion, How: "isbn", ISBN: "9780593098271", Clear: true,
		Candidates: []matchCandidate{{Suggestion: enrichment.Suggestion{Title: "Dune Messiah", ProviderID: "6", Series: "Dune", SeriesIndex: &position}, Score: 1}},
	})
	seen := false
	for _, row := range listEditor(t, server, "").Items {
		if row.ID != "a2" {
			continue
		}
		seen = true
		if row.Match == nil || row.Match.Series != "Dune" || row.Match.SeriesIndex == nil || *row.Match.SeriesIndex != 2 {
			t.Fatalf("a linked book lost its finding, so its series can never be replaced: %+v", row.Match)
		}
		for _, issue := range row.Issues {
			if issue == "match_found" {
				t.Fatalf("a book that is linked is not work to do: %v", row.Issues)
			}
		}
	}
	if !seen {
		t.Fatal("a2 was not in the list")
	}
}

// Search ranks by how alike the words are, and every volume of a series has the same words: asking about volume 6 came
// back with volume 1 first. The number is the question, so it decides the order.
func TestTheRightVolumeComesBeforeADifferentOne(t *testing.T) {
	one, six := 1.0, 6.0
	candidates := []matchCandidate{
		{Suggestion: enrichment.Suggestion{Title: "7th Time Loop: The Villainess Enjoys a Carefree Life (Light Novel), Vol. 1", SeriesIndex: &one}, Score: 48},
		{Suggestion: enrichment.Suggestion{Title: "7th Time Loop - Tome 01"}, Score: 47},
		{Suggestion: enrichment.Suggestion{Title: "7th Time Loop - Volume 06", SeriesIndex: &six}, Score: 41},
	}
	rankForBook(candidates, "7th Time Loop - Volume 06", 6)
	if got := candidates[0].Suggestion.Title; got != "7th Time Loop - Volume 06" {
		t.Fatalf("first candidate = %q, want volume 6 despite the lower score", got)
	}
	// And a book with no volume of its own keeps the plain score order.
	plain := []matchCandidate{
		{Suggestion: enrichment.Suggestion{Title: "Dune"}, Score: 1},
		{Suggestion: enrichment.Suggestion{Title: "Dune Messiah"}, Score: 2},
	}
	rankForBook(plain, "Dune", 0)
	if plain[0].Suggestion.Title != "Dune Messiah" {
		t.Fatalf("first = %q, want the higher score", plain[0].Suggestion.Title)
	}
}

func TestTheLastVolumeMarkerInATitleIsTheVolume(t *testing.T) {
	for title, want := range map[string]float64{
		"7th Time Loop - Volume 06":                        6,
		"7th Time Loop - Side Stories - Volume 01.2":       1.2,
		"Sword Art Online Vol. 16 - Alicization Exploding": 16,
		"Book 3 of Something":                              3,
		"Dune":                                             0,
	} {
		if got := editionVolume(title); got != want {
			t.Errorf("editionVolume(%q) = %v, want %v", title, got, want)
		}
	}
}
