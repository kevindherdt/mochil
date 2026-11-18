package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// With no library configured the app has nothing to show: the browser should land in settings,
// where a folder can be added, rather than on an empty dashboard with no obvious next step.
func TestEmptyInstallLandsInSettings(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")},
		nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/#/settings" {
		t.Fatalf("empty install root = %d %q, want 303 to /#/settings", response.Code, response.Header().Get("Location"))
	}
}

// A configured install keeps serving the app shell.
func TestConfiguredInstallServesTheShell(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("configured root = %d, want 200", response.Code)
	}
}
