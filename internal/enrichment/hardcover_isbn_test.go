package enrichment

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestISBNsAreLookedUpTogetherAndOnlyKnownOnesComeBack(t *testing.T) {
	var bodies []string
	hardcover := NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		bodies = append(bodies, string(raw))
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":{"editions":[
			{"isbn_13":"9780306406157","book":{"id":5,"title":"Dune","description":"Spice.","release_year":1965,"slug":"dune","contributions":[{"author":{"name":"Frank Herbert"}}]}},
			{"isbn_13":"9780306406157","book":{"id":6,"title":"Dune (duplicate edition)","slug":"x","contributions":[]}},
			{"isbn_13":"9780743273565","book":{"id":0,"title":"Broken","slug":"y","contributions":[]}}]}}`))}, nil
	})})
	found, err := hardcover.LookupISBNs(context.Background(), []string{"9780306406157", "9780743273565", "9780000000002"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 1 || !strings.Contains(bodies[0], "9780000000002") {
		t.Fatalf("one request for all three: %v", bodies)
	}
	dune, ok := found["9780306406157"]
	if !ok || dune.ProviderID != "5" || dune.Title != "Dune" || dune.Year != 1965 || dune.Overview != "Spice." || dune.Authors[0] != "Frank Herbert" {
		t.Fatalf("dune = %+v %v", dune, ok)
	}
	if len(found) != 1 {
		t.Fatalf("unknown and broken entries are left out: %+v", found)
	}
}

func TestISBNLookupRefusesMissingTokenAndTooManyISBNs(t *testing.T) {
	if _, err := NewHardcover("", nil).LookupISBNs(context.Background(), []string{"9780306406157"}); err != ErrProviderNotConnected {
		t.Fatalf("err = %v", err)
	}
	many := make([]string, MaximumISBNLookup+1)
	if _, err := NewHardcover("t", nil).LookupISBNs(context.Background(), many); err == nil {
		t.Fatal("an unbounded list must be refused")
	}
	if found, err := NewHardcover("t", nil).LookupISBNs(context.Background(), nil); err != nil || len(found) != 0 {
		t.Fatalf("nothing to look up is not an error: %v %v", found, err)
	}
}
