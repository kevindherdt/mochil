package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestATruncatedBookCountsFileDoesNotPreventStartup covers the crash the file
// used to cause: a half-written book-stats.json made bookstats.Open fail, which
// aborted the whole server instead of recovering one disposable cache.
func TestATruncatedBookCountsFileDoesNotPreventStartup(t *testing.T) {
	directory := t.TempDir()
	statsPath := filepath.Join(directory, "book-stats.json")
	if err := os.WriteFile(statsPath, []byte(`{"version":1,"books":{"books":{"one":{"words":`), 0o600); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{
		AllowOpen:           true,
		MetadataReviewState: filepath.Join(directory, "reviews.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("a damaged book-counts file prevented startup: %v", err)
	}
	defer handler.(interface{ Close() }).Close()

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/opds", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("library did not serve after recovery: %d", response.Code)
	}
	aside, err := filepath.Glob(statsPath + ".corrupt-*")
	if err != nil || len(aside) != 1 {
		t.Fatalf("the damaged file was not kept aside: %v %v", aside, err)
	}
}
