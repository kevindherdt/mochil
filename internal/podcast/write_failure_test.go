package podcast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// A feed that could not be written to disk used to stay in memory: the retry then
// answered "already subscribed" and returned success without ever saving, so the
// subscription disappeared on the next restart.
func TestAFailedSaveLeavesNothingBehind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss><channel><title>Quiet Feed</title><item><guid>episode-one</guid><title>First</title><pubDate>Wed, 20 Aug 2025 10:00:00 GMT</pubDate><enclosure url="https://media.example/one.mp3" type="audio/mpeg" length="42"/></item></channel></rss>`))
	}))
	defer server.Close()
	directory := t.TempDir()
	store, err := Open(filepath.Join(directory, "feeds.json"), server.Client())
	if err != nil {
		t.Fatal(err)
	}

	// A file where the store wants a directory makes every write fail.
	blocked := filepath.Join(directory, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocked, "feeds.json")

	if _, err := store.Subscribe(context.Background(), server.URL); err == nil {
		t.Fatal("subscribe reported success although the state could not be written")
	}
	if feeds := store.Feeds(); len(feeds) != 0 {
		t.Fatalf("a feed that never reached disk stayed in memory: %#v", feeds)
	}

	// With storage healthy again the retry has to persist for real.
	store.path = filepath.Join(directory, "feeds.json")
	feed, err := store.Subscribe(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("retry after storage recovered: %v", err)
	}
	if _, ok := store.Feed(feed.ID); !ok {
		t.Fatal("the retried subscription was not stored")
	}
	reopened, err := Open(store.path, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Feed(feed.ID); !ok {
		t.Fatal("the retried subscription did not survive a restart")
	}
}

// Removing a feed is the case where a failed write is most visible: the feed is
// gone from the screen and back after a restart.
func TestAFailedRemovalKeepsTheFeed(t *testing.T) {
	directory := t.TempDir()
	store, err := Open(filepath.Join(directory, "feeds.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(directory, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.feeds = append(store.feeds, Feed{ID: "one", URL: "https://example.test/feed.xml"})
	store.path = filepath.Join(blocked, "feeds.json")

	if err := store.Remove("one"); err == nil {
		t.Fatal("removal reported success although the state could not be written")
	}
	if _, ok := store.Feed("one"); !ok {
		t.Fatal("the feed vanished in memory although the removal was not saved")
	}
}
