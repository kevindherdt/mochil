package web

import (
	"net/http"
	"testing"
)

// The book dialog asks for a cover with a device token, so the route has to live in the
// device namespace: the browser's /covers/ is gated by a session cookie or Basic auth and a
// Bearer call never reaches it. Without a token the request must be refused by the *device*
// auth check (401), which is what proves the path is routed there at all — an unregistered
// path is rejected by the earlier whitelist with a plain 404.
func TestKoReaderCoverRequiresADeviceToken(t *testing.T) {
	server, _ := browseTestServer(t)
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/cover/books/a1", "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("cover without a token = %d, want 401; body %s", response.Code, response.Body.String())
	}
}

func TestKoReaderCoverIsScopedToKnownBooks(t *testing.T) {
	server, token := browseTestServer(t)
	for _, path := range []string{
		"/api/koreader/v1/cover/books/a1", // real book, no artwork in the fake store
		"/api/koreader/v1/cover/books/does-not-exist",
		"/api/koreader/v1/cover/removed/a1",
	} {
		response := pluginUpdateRequest(server, http.MethodGet, path, token)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s = %d, want 404; body %s", path, response.Code, response.Body.String())
		}
	}
}

// A cover request must never be answered with JSON: a plugin that renders the body as an
// image would show a broken glyph instead of failing visibly.
func TestKoReaderCoverDoesNotAnswerWithJSON(t *testing.T) {
	server, token := browseTestServer(t)
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/cover/books/a1", token)
	if got := response.Header().Get("Content-Type"); got == "application/json" {
		t.Fatalf("cover answered with JSON (%s): the plugin would render it as an image", got)
	}
}
