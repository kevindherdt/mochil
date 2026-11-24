package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJavaScriptAssetsAlwaysRevalidate(t *testing.T) {
	server := &multiServer{}
	request := httptest.NewRequest(http.MethodGet, "/assets/reader.js?v=1", nil)
	response := httptest.NewRecorder()
	server.serveStatic(response, request, "static/reader.js")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	if got := response.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("Cache-Control=%q", got)
	}
	etag := response.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	revalidated := httptest.NewRecorder()
	conditional := httptest.NewRequest(http.MethodGet, "/assets/reader.js?v=1", nil)
	conditional.Header.Set("If-None-Match", etag)
	server.serveStatic(revalidated, conditional, "static/reader.js")
	if revalidated.Code != http.StatusNotModified {
		t.Fatalf("conditional status=%d", revalidated.Code)
	}
}

func TestVersionedStylesRemainImmutable(t *testing.T) {
	server := &multiServer{}
	request := httptest.NewRequest(http.MethodGet, "/assets/app.css?v=3", nil)
	response := httptest.NewRecorder()
	server.serveStatic(response, request, "static/app.css")
	if got := response.Header().Get("Cache-Control"); got != "private, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control=%q", got)
	}
	if got := response.Header().Get("Content-Security-Policy"); !strings.Contains(got, "form-action 'self'") {
		t.Fatalf("Content-Security-Policy=%q", got)
	}
}
