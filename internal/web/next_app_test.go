package web

// The new app is the default since cutover: it is served from "/", its own client routes fall back to index.html, the
// built assets are cached forever (Vite names them by content hash), and the retired shell answers under /legacy.
//
// These run against whatever is actually built into internal/web/static/next/ at test time, which is gitignored and
// built by `npm run build` in web/ — so on a checkout that has never been built the built-dependent tests skip with a
// reason, and TestNextAppFallsBackToIndexWhenUnbuilt is the one that exercises real behaviour.
import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func nextAppBuilt(t *testing.T) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join("static", "next", "index.html"))
	return err == nil
}

func newTestHandler(t *testing.T, config MultiConfig) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := NewMulti(config, nil, logger)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestAppServesIndexAndFallsBackForClientRoutes(t *testing.T) {
	if !nextAppBuilt(t) {
		t.Skip("internal/web/static/next/ not built — run `npm run build` in web/ first")
	}
	handler := newTestHandler(t, MultiConfig{AllowOpen: true})

	for _, path := range []string{"/", "/library", "/__gallery", "/library/books/discover", "/settings/libraries"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Errorf("%s = %d, want 200", path, recorder.Code)
		}
		if recorder.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s Cache-Control = %q, want no-store (index.html must never be cached, it names the\n"+
				"current content-hashed assets)", path, recorder.Header().Get("Cache-Control"))
		}
	}
}

// The app used to live at /next. Old bookmarks, home-screen icons and links inside a stale APK still ask for it, so
// those requests must arrive at the same path without the prefix instead of 404ing inside the SPA.
func TestOldNextAddressesRedirectToTheRoot(t *testing.T) {
	handler := newTestHandler(t, MultiConfig{AllowOpen: true})

	cases := map[string]string{
		"/next":                    "/",
		"/next/":                   "/",
		"/next/library":            "/library",
		"/next/editor?x=1":         "/editor?x=1",
		"/next/settings/libraries": "/settings/libraries",
	}
	for from, want := range cases {
		request := httptest.NewRequest(http.MethodGet, from, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusPermanentRedirect {
			t.Errorf("%s = %d, want 308", from, recorder.Code)
			continue
		}
		if got := recorder.Header().Get("Location"); got != want {
			t.Errorf("%s redirects to %q, want %q", from, got, want)
		}
	}
}

func TestAppCachesHashedAssetsForever(t *testing.T) {
	if !nextAppBuilt(t) {
		t.Skip("internal/web/static/next/ not built — run `npm run build` in web/ first")
	}
	entries, err := os.ReadDir(filepath.Join("static", "next", "assets"))
	if err != nil || len(entries) == 0 {
		t.Skip("no built assets to check")
	}
	handler := newTestHandler(t, MultiConfig{AllowOpen: true})

	// The build is served from the root, so its assets are requested at /assets/… — the same prefix the legacy pages
	// use, which is why the handler has to look in the new build first.
	request := httptest.NewRequest(http.MethodGet, "/assets/"+entries[0].Name(), nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /assets/%s = %d, want 200", entries[0].Name(), recorder.Code)
	}
	if recorder.Header().Get("Cache-Control") != "private, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want immutable (Vite names this file by content hash, so it is safe to\n"+
			"cache forever — the name changes when the content does)", recorder.Header().Get("Cache-Control"))
	}
}

// The retired shell is gone: /legacy sends stale bookmarks home, and only the pages nobody has ported yet still answer.
//
// It used to serve the shell's own page at /legacy/. The shell is deleted (245 modules, 19 000 lines), so a request for
// it is a bookmark from another year — a redirect is the honest answer, and the three unported pages must keep working
// or the editor's import and the old chat would go with it.
func TestLegacyPathsSendBookmarksHomeAndKeepTheUnportedPages(t *testing.T) {
	handler := newTestHandler(t, MultiConfig{AllowOpen: true})

	for _, path := range []string{"/legacy/", "/legacy", "/legacy/library", "/legacy/settings"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusTemporaryRedirect || recorder.Header().Get("Location") != "/" {
			t.Errorf("%s = %d %q, want 307 to /", path, recorder.Code, recorder.Header().Get("Location"))
		}
	}

	for _, path := range []string{"/legacy/editor", "/legacy/editor/import", "/legacy/hermes"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Errorf("%s = %d, want 200: this page has not been ported", path, recorder.Code)
		}
	}

	// The root must never be the retired shell. Without a built app there is nothing to serve at "/" — a 404 is
	// correct then, and the point being checked still holds.
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if body := recorder.Body.String(); containsAny(body, "/assets/dist/app.", "mochil-shell-v1") {
		t.Fatalf("/ still serves the legacy shell; the new app must own the root")
	}
}

// containsAny reports whether any needle is in the text — the two builds are told apart by what they link.
func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func TestNextAppFallsBackToIndexWhenUnbuilt(t *testing.T) {
	if nextAppBuilt(t) {
		t.Skip("web/ is built in this checkout — this test only means something when it is not")
	}
	handler := newTestHandler(t, MultiConfig{AllowOpen: true})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("/ with no build present = %d, want 404 with a plain explanation, not a panic", recorder.Code)
	}
}

// An unknown API path must answer as an API, not with the app's HTML: a client that parses the body as JSON would read
// a 200 HTML shell as a successful (and empty) response.
func TestUnknownAPIPathAnswersAsJSON(t *testing.T) {
	handler := newTestHandler(t, MultiConfig{AllowOpen: true})

	request := httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if body := recorder.Body.String(); !containsAny(body, `"error"`) {
		t.Fatalf("expected a JSON error body, got: %.120q", body)
	}
}
