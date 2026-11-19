package podcast

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenRejectsMalformedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feeds.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"feeds":[]} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, nil); err == nil {
		t.Fatal("podcast state with trailing JSON was accepted")
	}
}

func TestSubscribeRefreshKeepsStableEpisodeAndUsesConditionalRequest(t *testing.T) {
	etagSeen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"one"` {
			etagSeen = true
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"one"`)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss><channel><title>Quiet Feed</title><item><guid>episode-one</guid><title>First</title><pubDate>Wed, 20 Aug 2025 10:00:00 GMT</pubDate><enclosure url="https://media.example/one.mp3" type="audio/mpeg" length="42"/></item></channel></rss>`))
	}))
	defer server.Close()
	store, err := Open(filepath.Join(t.TempDir(), "feeds.json"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	feed, err := store.Subscribe(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Episodes) != 1 || feed.Episodes[0].Title != "First" {
		t.Fatalf("feed=%+v", feed)
	}
	id := feed.Episodes[0].ID
	feed, err = store.Refresh(context.Background(), feed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !etagSeen || feed.Episodes[0].ID != id {
		t.Fatalf("conditional=%v feed=%+v", etagSeen, feed)
	}
}

func TestResubscribeKeepsDownloadIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss><channel><title>Quiet Feed</title><item><guid>episode-one</guid><title>First</title><enclosure url="https://media.example/one.mp3" type="audio/mpeg" length="42"/></item></channel></rss>`))
	}))
	defer server.Close()
	store, err := Open(filepath.Join(t.TempDir(), "feeds.json"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Subscribe(context.Background(), server.URL)
	if err != nil || len(first.Episodes) != 1 {
		t.Fatalf("first subscription: feed=%+v err=%v", first, err)
	}
	if err := store.Remove(first.ID); err != nil {
		t.Fatal(err)
	}
	second, err := store.Subscribe(context.Background(), server.URL)
	if err != nil || len(second.Episodes) != 1 {
		t.Fatalf("second subscription: feed=%+v err=%v", second, err)
	}
	if second.ID != first.ID || second.Episodes[0].ID != first.Episodes[0].ID {
		t.Fatalf("resubscribe changed download identity: first=%s/%s second=%s/%s", first.ID, first.Episodes[0].ID, second.ID, second.Episodes[0].ID)
	}
}

func TestParseRSSRejectsItemsWithoutEnclosures(t *testing.T) {
	feed, err := parseRSS([]byte(`<rss><channel><title>Feed</title><item><title>Post</title></item></channel></rss>`), "feed")
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Episodes) != 0 {
		t.Fatalf("episodes=%+v", feed.Episodes)
	}
}

func TestFeedSnapshotsDoNotExposeMutableEpisodeStorage(t *testing.T) {
	store := &Store{feeds: []Feed{{ID: "feed", Episodes: []Episode{{ID: "episode", Title: "Original"}}}}}

	feeds := store.Feeds()
	feed, ok := store.Feed("feed")
	if !ok {
		t.Fatal("feed not found")
	}
	feeds[0].Episodes[0].Title = "Mutated through Feeds"
	feed.Episodes[0].Title = "Mutated through Feed"

	got, ok := store.Feed("feed")
	if !ok || got.Episodes[0].Title != "Original" {
		t.Fatalf("stored episode was mutated through a snapshot: %+v", got)
	}
}

func TestParseRSSAcceptsCommonPodcastPublicationDates(t *testing.T) {
	feed, err := parseRSS([]byte(`<rss><channel><title>Feed</title><item><guid>one</guid><title>One</title><pubDate>Mon, 4 May 2009 00:00:00 -0700</pubDate><enclosure url="https://media.example/one.mp3"/></item></channel></rss>`), "feed")
	if err != nil {
		t.Fatal(err)
	}
	if feed.Episodes[0].Published.IsZero() || feed.Episodes[0].Published.Year() != 2009 {
		t.Fatalf("published = %v", feed.Episodes[0].Published)
	}
}

func TestParseRSSRejectsPlaceholderPublicationDate(t *testing.T) {
	feed, err := parseRSS([]byte(`<rss><channel><title>Feed</title><item><guid>one</guid><title>One</title><pubDate>0001-01-01T00:00:00Z</pubDate><enclosure url="https://media.example/one.mp3"/></item></channel></rss>`), "feed")
	if err != nil {
		t.Fatal(err)
	}
	if !feed.Episodes[0].Published.IsZero() {
		t.Fatalf("published = %v, want unknown", feed.Episodes[0].Published)
	}
}

func TestParseRSSNormalizesEpisodeDescription(t *testing.T) {
	feed, err := parseRSS([]byte(`<rss xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel><title>Feed</title><item><guid>one</guid><title>Post</title><content:encoded><![CDATA[<p>Saul &amp; Angel<br>find <strong>supplies</strong>.</p>]]></content:encoded><enclosure url="https://media.example/one.mp3" length="54777528"/></item></channel></rss>`), "feed")
	if err != nil {
		t.Fatal(err)
	}
	if got := feed.Episodes[0].Description; got != "Saul & Angel find supplies." {
		t.Fatalf("description = %q", got)
	}
	if got := feed.Episodes[0].Size; got != 54777528 {
		t.Fatalf("size = %d", got)
	}
}

func TestFeedPolicyPersistsAcrossRefreshAndRestart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<rss><channel><title>Policy Feed</title><item><guid>one</guid><title>One</title><enclosure url="https://media.example/one.mp3" type="audio/mpeg"/></item></channel></rss>`))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "feeds.json")
	store, err := Open(path, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	feed, err := store.Subscribe(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	policy := FeedPolicy{AutoDownload: true, RetainDownloads: 3, DeletePlayedAfterDays: 14}
	if _, err = store.SetPolicy(feed.ID, policy); err != nil {
		t.Fatal(err)
	}
	refreshed, err := store.Refresh(context.Background(), feed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Policy != policy {
		t.Fatalf("policy after refresh = %+v", refreshed.Policy)
	}
	reopened, err := Open(path, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Feeds()[0].Policy; got != policy {
		t.Fatalf("policy after restart = %+v", got)
	}
	if got := reopened.Feeds()[0].Episodes[0].MediaURL; got != "https://media.example/one.mp3" {
		t.Fatalf("media URL after restart = %q", got)
	}
}

func TestRefreshStaleRecoversLegacyEpisodesWithoutMediaURLs(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = w.Write([]byte(`<rss><channel><title>Legacy Feed</title><item><guid>one</guid><title>One</title><enclosure url="https://media.example/one.mp3" type="audio/mpeg"/></item></channel></rss>`))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "feeds.json")
	freshTimestamp := time.Now().UTC().Format(time.RFC3339Nano)
	legacy := `{"version":1,"feeds":[{"id":"legacy","url":"` + server.URL + `","title":"Legacy Feed","updated":"` + freshTimestamp + `","episodes":[{"id":"one","feed_id":"legacy","feed_title":"Legacy Feed","title":"One"}]}]}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	store.RefreshStale(context.Background(), 24*time.Hour)
	feed := store.Feeds()[0]
	if requests != 1 || len(feed.Episodes) != 1 || feed.Episodes[0].MediaURL != "https://media.example/one.mp3" {
		t.Fatalf("requests=%d feed=%+v", requests, feed)
	}
}

func TestFeedPolicyValidation(t *testing.T) {
	store, err := Open("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.SetPolicy("missing", FeedPolicy{AutoDownload: true}); err == nil {
		t.Fatal("automatic download without retention should fail")
	}
	if _, err = store.SetPolicy("missing", FeedPolicy{DeletePlayedAfterDays: 3651}); err == nil {
		t.Fatal("excessive played retention should fail")
	}
}

func TestRefreshOfARemovedFeedDoesNotTouchOthersAndKeepsNewPolicy(t *testing.T) {
	var hold atomic.Bool
	started, release := make(chan struct{}, 4), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hold.Load() && r.URL.Path == "/slow" {
			started <- struct{}{}
			<-release
		}
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss><channel><title>Feed` + r.URL.Path + `</title><item><guid>g</guid><enclosure url="https://media.example/a.mp3" type="audio/mpeg" length="1"/></item></channel></rss>`))
	}))
	defer server.Close()
	store, err := Open(filepath.Join(t.TempDir(), "feeds.json"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Subscribe(context.Background(), server.URL+"/first")
	if err != nil {
		t.Fatal(err)
	}
	slow, err := store.Subscribe(context.Background(), server.URL+"/slow")
	if err != nil {
		t.Fatal(err)
	}
	hold.Store(true)
	done := make(chan error, 1)
	go func() { _, err := store.Refresh(context.Background(), slow.ID); done <- err }()
	<-started
	if _, err := store.SetPolicy(slow.ID, FeedPolicy{AutoDownload: true, RetainDownloads: 3}); err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(first.ID); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	feeds := store.Feeds()
	if len(feeds) != 1 || feeds[0].ID != slow.ID || feeds[0].Policy.RetainDownloads != 3 {
		t.Fatalf("feeds after refresh: %+v", feeds)
	}
	if _, err := store.Refresh(context.Background(), first.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a removed feed must stay removed: %v", err)
	}
}
