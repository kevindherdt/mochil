package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The 404/not-found page must carry the app's branding (logo/wordmark), a clear
// message and a way back home, instead of the bare Go default "404 page not found".
func TestUnknownPathServesBrandedNotFoundPage(t *testing.T) {
	m := &multiServer{}
	recorder := httptest.NewRecorder()
	m.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/this-page-does-not-exist", nil))

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
