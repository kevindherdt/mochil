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
)

func TestMusicQualityIsScopedAndBounded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Unknown Track.mp3"), []byte("track"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/music-quality?library=music", nil))
	var report audio.QualityReport
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &report) != nil {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
	if report.Tracks != 1 || report.Counts["unknown_artist"] != 1 || len(report.Issues) != 1 {
		t.Fatalf("report=%+v", report)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache-control=%q", response.Header().Get("Cache-Control"))
	}
}

func TestMusicQualityRejectsOtherLibraryKinds(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/music-quality?library=books", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}
