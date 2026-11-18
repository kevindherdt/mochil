package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/video"
)

func TestVideoProgressRejectsCheckpointForReplacedMedia(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Film.mp4")
	if err := os.WriteFile(file, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	handler, err := NewMulti(MultiConfig{AllowOpen: true, VideoState: filepath.Join(t.TempDir(), "progress.json")}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/video-progress/movies/" + item.ID
	put := func(mediaRevision string, revision int, position int) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(map[string]any{"expected_media_revision": mediaRevision, "expected_revision": revision, "position_seconds": position, "duration_seconds": 100, "explicit_seek": true})
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, path, strings.NewReader(string(body))))
		return response
	}
	if response := put(item.Revision, 0, 20); response.Code != http.StatusOK {
		t.Fatalf("initial=%d %s", response.Code, response.Body.String())
	}
	if err := os.WriteFile(file, []byte("replacement with different size"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	replacement := store.Items()[0]
	if replacement.ID != item.ID || replacement.Revision == item.Revision {
		t.Fatal("fixture did not replace same-identity media")
	}
	response := put(item.Revision, 1, 5)
	if response.Code != http.StatusPreconditionFailed || !strings.Contains(response.Body.String(), "video_media_changed") {
		t.Fatalf("stale=%d %s", response.Code, response.Body.String())
	}
	current := httptest.NewRecorder()
	handler.ServeHTTP(current, httptest.NewRequest(http.MethodGet, path, nil))
	var progress video.Progress
	if err := json.Unmarshal(current.Body.Bytes(), &progress); err != nil {
		t.Fatal(err)
	}
	if progress.Position != 20 || progress.Revision != 1 {
		t.Fatalf("stale write changed progress: %+v", progress)
	}
	if response := put(replacement.Revision, 1, 30); response.Code != http.StatusOK {
		t.Fatalf("fresh=%d %s", response.Code, response.Body.String())
	}
}
