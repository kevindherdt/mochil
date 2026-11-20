package enrichment

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestOpenLibrarySearchIsBoundedAndCached(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.Host != "openlibrary.org" || request.URL.Query().Get("limit") != "5" || request.URL.Query().Get("title") != "A Book" {
			t.Fatalf("unexpected request: %s", request.URL)
		}
		body := `{"docs":[{"key":"/works/OL1W","title":"A Book","author_name":["Writer"],"cover_i":42}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	provider := NewOpenLibrary(client)
	for range 2 {
		results, err := provider.Search(context.Background(), Query{Title: "A Book"})
		if err != nil || len(results) != 1 || results[0].ProviderID != "OL1W" || results[0].Authors[0] != "Writer" {
			t.Fatalf("results=%#v err=%v", results, err)
		}
	}
	if calls != 1 {
		t.Fatalf("requests=%d, want cached single request", calls)
	}
}

func TestOpenLibraryRejectsUnboundedQueryAndResponse(t *testing.T) {
	provider := NewOpenLibrary(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 2<<20))), Header: make(http.Header)}, nil
	})})
	if _, err := provider.Search(context.Background(), Query{Title: strings.Repeat("x", 501)}); err == nil {
		t.Fatal("oversized title was accepted")
	}
	if _, err := provider.Search(context.Background(), Query{Title: "book"}); err == nil {
		t.Fatal("oversized response was accepted")
	}
}
