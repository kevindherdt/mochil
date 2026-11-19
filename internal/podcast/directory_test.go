package podcast

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSearchDirectoryNormalizesResults(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("term") != "history" || r.URL.Query().Get("entity") != "podcast" {
			t.Fatalf("unexpected search query: %s", r.URL.RawQuery)
		}
		body := `{"results":[{"collectionName":"Past Present","artistName":"A. Host","feedUrl":"https://example.test/feed.xml","artworkUrl600":"https://example.test/art.jpg","trackCount":42,"genres":["History"]},{"collectionName":"No Feed"}]}`
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	results, err := SearchDirectory(context.Background(), client, " history ", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Past Present" || results[0].EpisodeCount != 42 || results[0].Genre != "History" {
		t.Fatalf("unexpected results: %#v", results)
	}
}
