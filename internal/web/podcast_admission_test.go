package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/podcast"
)

type admissionTransport func(*http.Request) (*http.Response, error)

func (f admissionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type admissionBody struct {
	entered chan struct{}
	release chan struct{}
}

func (b *admissionBody) Read([]byte) (int, error) { close(b.entered); <-b.release; return 0, io.EOF }
func (b *admissionBody) Close() error             { return nil }

func TestPodcastDownloadDeleteThenImmediateSaveReportsConflict(t *testing.T) {
	body := &admissionBody{entered: make(chan struct{}), release: make(chan struct{})}
	client := &http.Client{Transport: admissionTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: -1, Header: make(http.Header)}, nil
	})}
	statePath := filepath.Join(t.TempDir(), "podcasts.json")
	data := `{"version":1,"feeds":[{"id":"show","url":"https://example.com/feed","title":"Show","episodes":[{"id":"episode","feed_id":"show","title":"One","media_url":"https://example.com/audio"}]}]}`
	if err := os.WriteFile(statePath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := podcast.Open(statePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	downloader := podcast.NewDownloader(t.TempDir(), client, 1024)
	defer downloader.Close()
	defer close(body.release)
	m := &multiServer{podcasts: store, podcastDownloads: downloader}
	request := func(method string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		m.apiPodcastDownload(w, httptest.NewRequest(method, "/api/podcast-downloads/episode", nil))
		return w
	}
	first := request(http.MethodPost)
	if first.Code != http.StatusAccepted {
		t.Fatalf("first post=%d %s", first.Code, first.Body.String())
	}
	select {
	case <-body.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not start")
	}
	deleted := request(http.MethodDelete)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete=%d %s", deleted.Code, deleted.Body.String())
	}
	rejected := request(http.MethodPost)
	if rejected.Code != http.StatusConflict {
		t.Fatalf("immediate post=%d body=%s; want conflict", rejected.Code, rejected.Body.String())
	}
	var result podcast.Download
	if err := json.Unmarshal(rejected.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "failed" || !strings.Contains(strings.ToLower(result.Error), "try again") {
		t.Fatalf("result=%+v; want retryable error", result)
	}
}
