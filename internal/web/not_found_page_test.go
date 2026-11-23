package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Since cutover the app owns routing: an extension-less path is handed to the app router, which shows its own
// Not-found screen. What still lands on the server's branded 404 is a path that names a file — a typo'd .js, .css or
// .ico — because answering those with an HTML shell is how a missing asset turns into a page that "looks broken".
//
// The page must carry the app's branding (logo/wordmark), a clear message and a way back home, instead of the bare Go
// default "404 page not found".
func TestUnknownFilePathServesBrandedNotFoundPage(t *testing.T) {
	m := &multiServer{}
	recorder := httptest.NewRecorder()
	m.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/this-page-does-not-exist.js", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	body := recorder.Body.String()
	if strings.Contains(body, "404 page not found") {
		t.Fatalf("not-found page must not be the bare Go default text, got: %q", body)
	}
	if !strings.Contains(body, "wordmark") {
		t.Fatalf("not-found page must carry the app's branding (wordmark), got: %q", body)
	}
	if !strings.Contains(body, `href="/"`) {
		t.Fatalf("not-found page must link back home, got: %q", body)
	}
	contentType := recorder.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("not-found page must be served as HTML, got Content-Type %q", contentType)
	}
}

// An extension-less dead end belongs to the app, which renders its own Not-found screen inside the shell — so the
// server must hand back the app rather than a bare 404.
func TestUnknownAppPathServesTheApp(t *testing.T) {
	if _, err := os.Stat(filepath.Join("static", "next", "index.html")); err != nil {
		t.Skip("internal/web/static/next/ not built — run `npm run build` in web/ first")
	}
	m := &multiServer{}
	recorder := httptest.NewRecorder()
	m.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/this-page-does-not-exist", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (the app router decides)", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `id="root"`) {
		t.Fatalf("expected the app shell, got: %.120q", body)
	}
}
