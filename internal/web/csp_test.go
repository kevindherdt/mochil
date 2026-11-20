package web

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The page's theme pre-paint script is inline, so a policy of script-src 'self' blocks it: a light, paper
// or e-ink reader then sees the dark palette on every load, which is exactly what the script exists to
// prevent. The hash has to come from the bytes that are served, not from a constant somebody remembers to
// update.
func TestTheAppShellAllowsItsOwnInlineScript(t *testing.T) {
	server := &multiServer{}
	response := httptest.NewRecorder()
	server.serveStatic(response, httptest.NewRequest(http.MethodGet, "/", nil), "static/index.html")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	policy := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "'sha256-") {
		t.Fatalf("the inline script is not allowed by %q", policy)
	}
	body := response.Body.String()
	open := strings.Index(body, "<script>")
	if open < 0 {
		t.Fatal("the shell no longer carries the theme pre-paint script")
	}
	closeAt := strings.Index(body[open:], "</script>")
	if closeAt < 0 {
		t.Fatal("unterminated inline script")
	}
	inline := body[open+len("<script>") : open+closeAt]
	sum := sha256.Sum256([]byte(inline))
	want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	if !strings.Contains(policy, want) {
		t.Fatalf("policy %q does not carry the hash of the inline script (%s)", policy, want)
	}
}

func TestTheShellStillAllowsGeneratedCovers(t *testing.T) {
	server := &multiServer{}
	response := httptest.NewRecorder()
	server.serveStatic(response, httptest.NewRequest(http.MethodGet, "/", nil), "static/index.html")
	policy := response.Header().Get("Content-Security-Policy")
	// Cover placeholders and scene backgrounds are drawn as data: and blob: images.
	for _, directive := range []string{"img-src 'self' data: blob:", "media-src 'self' data: blob:"} {
		if !strings.Contains(policy, directive) {
			t.Fatalf("policy %q dropped %q", policy, directive)
		}
	}
}

// The sign-in page draws its field icons from data: images and was refusing them, so the lock and the user
// icon were broken on the screen that asks for a password.
func TestTheSignInPageAllowsItsOwnIcons(t *testing.T) {
	auth := newSessionAuth("someone", "a-password")
	response := httptest.NewRecorder()
	auth.loginPage(response, httptest.NewRequest(http.MethodGet, "/login", nil), http.StatusOK, "")
	policy := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "img-src 'self' data:") {
		t.Fatalf("policy %q blocks the page's own icons", policy)
	}
}
