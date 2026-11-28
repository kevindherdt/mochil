package podcast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// The same feed pasted with and without a trailing slash used to become two subscriptions with disjoint episode ids: with
// auto-download every enclosure was fetched twice, and playback state was split between the two.
func TestFeedAddressVariantsAreOneSubscription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss><channel><title>Quiet Feed</title><item><guid>episode-one</guid><title>First</title><enclosure url="https://media.example/one.mp3" type="audio/mpeg" length="42"/></item></channel></rss>`))
	}))
	defer server.Close()
	store, err := Open(filepath.Join(t.TempDir(), "feeds.json"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Subscribe(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Subscribe(context.Background(), server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("the same feed twice made two subscriptions: %q vs %q", first.ID, second.ID)
	}
	if feeds := store.Feeds(); len(feeds) != 1 {
		t.Fatalf("%d subscriptions, want 1", len(feeds))
	}
	if first.Episodes[0].ID != second.Episodes[0].ID {
		t.Fatalf("episode ids differ: %q vs %q", first.Episodes[0].ID, second.Episodes[0].ID)
	}
}
