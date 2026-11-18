package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func newLiveHandler(t *testing.T, cfg MultiConfig) http.Handler {
	t.Helper()
	handler, err := NewMulti(cfg,
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func signInCode(t *testing.T, handler http.Handler, username, password string) int {
	t.Helper()
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/login", nil))
	form := url.Values{"csrf": {csrfToken(t, page.Body.String())}, "username": {username}, "password": {password}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

// A live reload builds a second handler. If it opens the account and device
// stores again, it keeps its own snapshot: disabling an account (or resetting a
// password) through a request the old handler is still finishing would land in
// the generation that is about to be discarded, so the published server kept
// accepting the credentials it had already loaded.
func TestALiveReloadSharesOneAccountStore(t *testing.T) {
	directory := t.TempDir()
	cfg := MultiConfig{ReadingState: filepath.Join(directory, "reading.json")}
	stores := NewStoreSet()
	cfg.Stores = stores

	first := newLiveHandler(t, cfg)
	page := httptest.NewRecorder()
	first.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/setup", nil))
	form := url.Values{"csrf": {csrfToken(t, page.Body.String())}, "username": {"owner"}, "password": {"a long enough password"}, "confirm": {"a long enough password"}}
	request := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	created := httptest.NewRecorder()
	first.ServeHTTP(created, request)
	if created.Code != http.StatusSeeOther {
		t.Fatalf("setup = %d", created.Code)
	}

	reader, err := stores.accounts.Create("reader", "another long password", "user")
	if err != nil {
		t.Fatal(err)
	}
	if code := signInCode(t, first, "reader", "another long password"); code != http.StatusSeeOther {
		t.Fatalf("the reader could not sign in: %d", code)
	}

	// The reload: a second handler over the same state file.
	second := newLiveHandler(t, cfg)
	carried, ok := second.(interface{ Stores() *StoreSet })
	if !ok {
		t.Fatal("the rebuilt handler does not hand its stores back")
	}
	if carried.Stores() != stores {
		t.Fatal("the rebuilt handler opened its own account state")
	}

	// A change made through the first generation must already be in force for
	// the second, without any coordination at reload time.
	if err := stores.accounts.SetDisabled(reader.ID, true); err != nil {
		t.Fatal(err)
	}
	if code := signInCode(t, second, "reader", "another long password"); code != http.StatusUnauthorized {
		t.Fatalf("the rebuilt handler still signed in a disabled account: %d", code)
	}
}
