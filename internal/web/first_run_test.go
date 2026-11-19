package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// With no library configured the browser should land in Settings, where a folder can be added.
// That has to happen client-side: a 303 to "/#/settings" is a redirect to "/", because a browser
// never sends the fragment back, so the server answered 303 forever and Chrome gave up with
// ERR_TOO_MANY_REDIRECTS. Only browsers looped — curl sends Accept: */* and got the shell.
func TestEmptyInstallServesTheShellWithoutARedirect(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")},
		nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if location := response.Header().Get("Location"); location != "" {
		t.Fatalf("empty install root redirects to %q; the fragment never reaches the server, so this loops", location)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("empty install root = %d, want 200 with the app shell", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("empty install root content type = %q, want text/html", contentType)
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
