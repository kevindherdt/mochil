package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"mochil/internal/enrichment"
)

// The match blurb is the largest field on an editor row and is read in one place only — filling a description that the
// book does not have. Sending it for a book that already has one costs about 700 bytes per row for nothing.
func TestMatchOverviewOnlyTravelsWhenItIsTheOnlyDescription(t *testing.T) {
	candidate := matchCandidate{Suggestion: enrichment.Suggestion{Title: "Dune", Overview: "A long blurb about sand."}, Score: 0.9}

	withDescription := candidate
	if got := withDescription.forList(&editorRow{Description: "What the book file says."}).Overview; got != "" {
		t.Errorf("a book with its own description still carried the match blurb: %q", got)
	}

	empty := candidate
	if got := empty.forList(&editorRow{}).Overview; got != "A long blurb about sand." {
		t.Errorf("a book with no description must keep the blurb, it is what staging a match fills in; got %q", got)
	}

	// A candidate that came back from the on-disk cache holds its blurb on this struct's own field (that is where JSON
	// writes it), with the embedded one empty. Both shapes must behave the same: dropped when the book has its own
	// description, kept when it does not — otherwise the books that most need a blurb, the ones with no description,
	// are exactly the ones that lose it.
	cached := matchCandidate{Score: 0.9}
	cached.Overview = "A long blurb about sand."
	if got := cached.forList(&editorRow{Description: "What the book file says."}).Overview; got != "" {
		t.Errorf("a cached candidate leaked its blurb onto a book that has a description: %q", got)
	}
	cachedAgain := matchCandidate{Score: 0.9}
	cachedAgain.Overview = "A long blurb about sand."
	if got := cachedAgain.forList(&editorRow{}).Overview; got != "A long blurb about sand." {
		t.Errorf("a cached candidate lost the blurb for a book with no description: %q", got)
	}
}

// The single-book search is the one place a blurb must always travel: it is opened to decide about that book, and Replace
// is supposed to be able to fix a wrong description. The blurb sits on a field that shadows the suggestion's own, so if it
// is not copied across when the candidate is built it never leaves the server — Replace then staged nothing at all for
// the description, and the dialog showed candidates with no blurb to judge them by.
func TestTheSingleBookSearchCarriesItsBlurb(t *testing.T) {
	server := editorTestServer(t) // "Dune" by Frank Herbert is book a1 in library "books"
	server.cfg.Modules = map[string]bool{"hardcover": true}
	server.hardcover = enrichment.NewHardcover("token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "SearchBooks"):
			response = `{"data":{"search":{"ids":[9]}}}`
		case strings.Contains(body, "BookDetails"):
			response = `{"data":{"books":[{"id":9,"title":"Dune","slug":"dune","description":"Spice.","contributions":[{"author":{"name":"Frank Herbert"}}]}]}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})

	answer := editorCall(server, http.MethodGet, "/api/metadata/editor/match?library=books&book=a1&title=Dune&author=Frank%20Herbert", "")
	if answer.Code != http.StatusOK {
		t.Fatalf("the search answered %d: %s", answer.Code, answer.Body.String())
	}
	var body struct {
		Candidates []map[string]any `json:"candidates"`
	}
	if err := json.Unmarshal(answer.Body.Bytes(), &body); err != nil {
		t.Fatalf("the search answer was not JSON: %v (%s)", err, answer.Body.String())
	}
	if len(body.Candidates) == 0 {
		t.Fatalf("no candidates came back: %s", answer.Body.String())
	}
	if blurb, _ := body.Candidates[0]["overview"].(string); blurb != "Spice." {
		t.Fatalf("the candidate carried no blurb (%q), so Replace can neither show it nor stage it: %s", blurb, answer.Body.String())
	}
}
