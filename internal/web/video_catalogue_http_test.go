package web

import (
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

// Home's "jump back in" rail picks its icon from an item's library_kind (kindIconSVG falls back
// to the book glyph for anything it doesn't recognize). Without this field, every movie and TV
// show that showed up on Home carried a book badge, because audioView already sent library_kind
// and videoView never did.
func TestVideoProgressCarriesLibraryKindForTheHomeRail(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Film.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, VideoState: filepath.Join(t.TempDir(), "progress.json")},
		[]Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	itemID := store.Items()[0].ID
	put := httptest.NewRecorder()
	handler.ServeHTTP(put, httptest.NewRequest(http.MethodPut, "/api/video-progress/movies/"+itemID,
		strings.NewReader(`{"position_seconds":10,"duration_seconds":100,"device_id":"tv"}`)))
	if put.Code != http.StatusOK {
		t.Fatalf("put progress: %d %s", put.Code, put.Body.String())
	}
	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/video-progress", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list progress: %d %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), `"library_kind":"movies"`) {
		t.Fatalf("video-progress response has no library_kind, the Home rail would fall back to the book icon: %s", list.Body.String())
	}
}
