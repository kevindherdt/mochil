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

func TestVideoQualityReportIsLibrarySpecificAndReadOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Show", "Season 01"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Show", "Season 01", "Show.S01E01.mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("shows", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, VideoMetadataState: filepath.Join(t.TempDir(), "video-metadata.json")}, []Library{{ID: "shows", Name: "Shows", Kind: "tv", SourceType: "filesystem", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/video-quality?library=shows", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"unmatched_show":1`) || !strings.Contains(response.Body.String(), `"groups":1`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	wrongMethod := httptest.NewRecorder()
	handler.ServeHTTP(wrongMethod, httptest.NewRequest(http.MethodPost, "/api/metadata/video-quality?library=shows", nil))
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", wrongMethod.Code)
	}
}
