package web

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/podcast"
)

func TestPodcastMediaRepairsLegacySourceBeforeStreaming(t *testing.T) {
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed" {
			_, _ = io.WriteString(w, `<rss><channel><title>Legacy</title><item><guid>legacy-one</guid><title>One</title><enclosure url="`+upstream.URL+`/episode.mp3" type="audio/mpeg"/></item></channel></rss>`)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = io.WriteString(w, "recovered")
	}))
	defer upstream.Close()
	sum := sha256.Sum256([]byte("first\x00legacy-one"))
	episodeID := hex.EncodeToString(sum[:16])
	state := filepath.Join(t.TempDir(), "podcasts.json")
	legacy := `{"version":1,"feeds":[{"id":"first","url":"` + upstream.URL + `/feed","title":"Legacy","updated":"2099-01-01T00:00:00Z","episodes":[{"id":"` + episodeID + `","feed_id":"first","feed_title":"Legacy","title":"One","media_type":"audio/mpeg"}]}]}`
	if err := os.WriteFile(state, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := podcast.Open(state, upstream.Client())
	if err != nil {
		t.Fatal(err)
	}
	downloader := podcast.NewDownloader(t.TempDir(), upstream.Client(), 1<<20)
	defer downloader.Close()
	previousClient := podcastPublicClient
	podcastPublicClient = upstream.Client()
	defer func() { podcastPublicClient = previousClient }()
	response := httptest.NewRecorder()

	(&multiServer{podcasts: store, podcastDownloads: downloader}).podcastMedia(response, httptest.NewRequest(http.MethodGet, "/podcast-media/"+episodeID, nil))

	if response.Code != http.StatusOK || response.Body.String() != "recovered" {
		t.Fatalf("response=%d body=%q", response.Code, response.Body.String())
	}
	if episode, ok := store.Episode(episodeID); !ok || episode.MediaURL != upstream.URL+"/episode.mp3" {
		t.Fatalf("episode=%+v exists=%v", episode, ok)
	}
}

func TestPodcastMediaPreservesResumeHeadersAcrossRedirect(t *testing.T) {
	var receivedRange, receivedIfRange, receivedEncoding string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/episode.mp3", http.StatusFound)
			return
		}
		receivedRange = r.Header.Get("Range")
		receivedIfRange = r.Header.Get("If-Range")
		receivedEncoding = r.Header.Get("Accept-Encoding")
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Range", "bytes 200-203/1000")
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, "data")
	}))
	defer upstream.Close()

	state := filepath.Join(t.TempDir(), "podcasts.json")
	data := `{"version":1,"feeds":[{"id":"first","url":"https://example.com/feed","title":"First Show","episodes":[{"id":"episode","feed_id":"first","feed_title":"First Show","title":"One","media_url":"` + upstream.URL + `/start","media_type":"audio/mpeg"}]}]}`
	if err := os.WriteFile(state, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := podcast.Open(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	downloader := podcast.NewDownloader(t.TempDir(), upstream.Client(), 1<<20)
	defer downloader.Close()
	previousClient := podcastPublicClient
	podcastPublicClient = upstream.Client()
	defer func() { podcastPublicClient = previousClient }()
	m := multiServer{podcasts: store, podcastDownloads: downloader}
	if episode, ok := store.Episode("episode"); !ok || episode.MediaURL != upstream.URL+"/start" {
		t.Fatalf("episode media URL=%q exists=%v", episode.MediaURL, ok)
	}
	request := httptest.NewRequest(http.MethodGet, "/podcast-media/episode", nil)
	request.Header.Set("Range", "bytes=200-203")
	request.Header.Set("If-Range", `"episode-etag"`)
	response := httptest.NewRecorder()

	m.podcastMedia(response, request)

	if response.Code != http.StatusPartialContent || response.Body.String() != "data" || response.Header().Get("Content-Range") != "bytes 200-203/1000" {
		t.Fatalf("response=%d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}
	if receivedRange != "bytes=200-203" || receivedIfRange != `"episode-etag"` || receivedEncoding != "identity" {
		t.Fatalf("range=%q if-range=%q encoding=%q", receivedRange, receivedIfRange, receivedEncoding)
	}
}

func TestAndroidPodcastShowsAreSearchableAndBounded(t *testing.T) {
	state := filepath.Join(t.TempDir(), "podcasts.json")
	data := `{"version":1,"feeds":[{"id":"second","url":"https://example.com/second","title":"Second Show","description":"Science","episodes":[]},{"id":"first","url":"https://example.com/first","title":"First Show","description":"Books","episodes":[{"id":"episode","feed_id":"first","feed_title":"First Show","title":"One"}]}]}`
	if err := os.WriteFile(state, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := podcast.Open(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := multiServer{podcasts: store}
	response := httptest.NewRecorder()
	m.androidPodcastShows(response, httptest.NewRequest(http.MethodGet, "/api/android/v1/podcast-shows?q=books&limit=1", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"title":"First Show"`) || !strings.Contains(response.Body.String(), `"episode_count":1`) || strings.Contains(response.Body.String(), "Second Show") {
		t.Fatalf("shows=%d %s", response.Code, response.Body.String())
	}
}

func TestAndroidPodcastSubscriptionRoutesRequireDeviceAndCanRemoveFeed(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, "podcasts.json")
	data := `{"version":1,"feeds":[{"id":"first","url":"https://example.com/first","title":"First Show","episodes":[]}]}`
	if err := os.WriteFile(state, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := podcast.Open(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	devices, err := newAndroidDeviceStore(filepath.Join(root, "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := multiServer{podcasts: store, androidDevices: devices, cfg: MultiConfig{Username: "reader"}}
	token, _, err := devices.issue(m.readingAccount(nil), "Test phone")
	if err != nil {
		t.Fatal(err)
	}

	denied := httptest.NewRecorder()
	m.androidAPI(denied, httptest.NewRequest(http.MethodGet, "/api/android/v1/podcast-feeds", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Fatalf("uncredentialed feeds=%d", denied.Code)
	}
	policy := httptest.NewRequest(http.MethodPut, "/api/android/v1/podcast-feeds/first/policy", strings.NewReader(`{"auto_download":true,"retain_downloads":3,"delete_played_after_days":7}`))
	policy.Header.Set("Authorization", "Bearer "+token)
	policy.Header.Set("Content-Type", "application/json")
	policyResponse := httptest.NewRecorder()
	m.androidAPI(policyResponse, policy)
	updated, exists := store.Feed("first")
	if policyResponse.Code != http.StatusOK || !exists || !updated.Policy.AutoDownload || updated.Policy.RetainDownloads != 3 || updated.Policy.DeletePlayedAfterDays != 7 {
		t.Fatalf("policy=%d feed=%+v body=%s", policyResponse.Code, updated, policyResponse.Body.String())
	}

	remove := httptest.NewRequest(http.MethodDelete, "/api/android/v1/podcast-feeds/first", nil)
	remove.Header.Set("Authorization", "Bearer "+token)
	removed := httptest.NewRecorder()
	m.androidAPI(removed, remove)
	if removed.Code != http.StatusNoContent || len(store.Feeds()) != 0 {
		t.Fatalf("remove=%d feeds=%d body=%s", removed.Code, len(store.Feeds()), removed.Body.String())
	}
}
