package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/audio"
	"mochil/internal/document"
	"mochil/internal/video"
)

func TestLibrariesReturnsMediaShelfCounts(t *testing.T) {
	root := t.TempDir()
	videoRoot := filepath.Join(root, "video")
	audioRoot := filepath.Join(root, "audio")
	documentRoot := filepath.Join(root, "documents")
	for _, directory := range []string{videoRoot, audioRoot, documentRoot} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	for path, body := range map[string]string{
		filepath.Join(videoRoot, "film.mkv"):      "video",
		filepath.Join(audioRoot, "track.mp3"):     "audio",
		filepath.Join(documentRoot, "one.md"):     "one",
		filepath.Join(documentRoot, "second.txt"): "second",
	} {
		if err := os.WriteFile(path, []byte(body), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	videoStore := video.NewStore("movies", videoRoot, "")
	audioStore := audio.NewStore("music", audioRoot, "")
	documentStore := document.NewStore("notes", documentRoot, "")
	if _, err := videoStore.Scan(); err != nil {
		t.Fatal(err)
	}
	if _, err := audioStore.Scan(); err != nil {
		t.Fatal(err)
	}
	if _, err := documentStore.Scan(); err != nil {
		t.Fatal(err)
	}

	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{
		{ID: "movies", Name: "Movies", Kind: "movies", Video: videoStore},
		{ID: "music", Name: "Music", Kind: "music", Audio: audioStore},
		{ID: "notes", Name: "Notes", Kind: "documents", Documents: documentStore},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/libraries", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("library discovery must not cache renamed labels: %q", response.Header().Get("Cache-Control"))
	}
	var libraries []libraryView
	if err := json.Unmarshal(response.Body.Bytes(), &libraries); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, library := range libraries {
		counts[library.ID] = library.Count
	}
	if counts["movies"] != 1 || counts["music"] != 1 || counts["notes"] != 2 {
		t.Fatalf("counts=%v", counts)
	}
}
