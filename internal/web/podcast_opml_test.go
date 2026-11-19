package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mochil/internal/podcast"
)

func TestApiPodcastOPMLExportsEverySubscription(t *testing.T) {
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<rss><channel><title>Show</title></channel></rss>`))
	}))
	defer feedServer.Close()
	store, err := podcast.Open(t.TempDir()+"/podcasts.json", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Subscribe(t.Context(), feedServer.URL); err != nil {
		t.Fatal(err)
	}
	m := &multiServer{podcasts: store}
	w := httptest.NewRecorder()
	m.apiPodcastOPML(w, httptest.NewRequest(http.MethodGet, "/api/podcast-opml", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "opml") {
		t.Errorf("content-type=%q, want opml", ct)
	}
	if !strings.Contains(w.Body.String(), feedServer.URL) {
		t.Errorf("export did not include the subscribed feed URL: %s", w.Body.String())
	}
}

func TestApiPodcastOPMLImportsFeedsAndReportsTheResult(t *testing.T) {
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<rss><channel><title>Show</title></channel></rss>`))
	}))
	defer feedServer.Close()
	store, err := podcast.Open(t.TempDir()+"/podcasts.json", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{podcasts: store}
	body := `<?xml version="1.0"?><opml version="2.0"><head><title>x</title></head><body>
  <outline text="Show" type="rss" xmlUrl="` + feedServer.URL + `"/>
</body></opml>`
	w := httptest.NewRecorder()
	m.apiPodcastOPML(w, httptest.NewRequest(http.MethodPost, "/api/podcast-opml", strings.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var result podcast.ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Subscribed != 1 {
		t.Errorf("subscribed=%d, want 1", result.Subscribed)
	}
	if len(store.Feeds()) != 1 {
		t.Errorf("feeds=%d, want 1", len(store.Feeds()))
	}
}

func TestApiPodcastOPMLRejectsGarbageWithoutCrashing(t *testing.T) {
	store, err := podcast.Open(t.TempDir()+"/podcasts.json", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{podcasts: store}
	w := httptest.NewRecorder()
	m.apiPodcastOPML(w, httptest.NewRequest(http.MethodPost, "/api/podcast-opml", strings.NewReader("not opml")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestApiPodcastOPML404sWithoutAPodcastStore(t *testing.T) {
	m := &multiServer{}
	w := httptest.NewRecorder()
	m.apiPodcastOPML(w, httptest.NewRequest(http.MethodGet, "/api/podcast-opml", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404", w.Code)
	}
}
