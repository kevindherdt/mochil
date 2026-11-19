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

// A matching run resumed from an offset past the end of the target list must
// report it finished rather than overflow offset+limit into a negative slice
// bound and panic.
func TestVideoMetadataMatchClampsHugeOffset(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Arrival (2016).mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, filepath.Join(t.TempDir(), "video.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		TMDBToken:          "private-token",
		VideoMetadataState: filepath.Join(t.TempDir(), "metadata.json"),
	}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", MetadataProvider: "tmdb", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/video-metadata-match/movies?offset=9223372036854775807", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("match response=%d %s", response.Code, response.Body.String())
	}
	if body := response.Body.String(); !strings.Contains(body, `"unmatched_total":1`) || strings.Contains(body, "next_offset") {
		t.Fatalf("match body=%s, want the run reported as finished", body)
	}
}
