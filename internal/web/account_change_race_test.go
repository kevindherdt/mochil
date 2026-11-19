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

	"mochil/internal/accounts"
)

// createFirstRunInstall makes a server that has just been through /setup, i.e.
// a fresh install where the administrator was created from the setup page and no
// credentials were configured in the environment.
func createFirstRunInstall(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	handler, err := NewMulti(MultiConfig{ReadingState: filepath.Join(dir, "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/setup", nil))
	form := url.Values{"csrf": {csrfToken(t, page.Body.String())}, "username": {"owner"}, "password": {"a long enough password"}, "confirm": {"a long enough password"}}
	created := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(created, request)
	if created.Code != http.StatusSeeOther {
		t.Fatalf("setup = %d", created.Code)
	}
	return handler
}

// The environment recovery path compares the presented credentials with the
// configured ones. With nothing configured it compared two empty strings and
// matched, so a blank sign-in on a first-run install became the administrator.
func TestBlankCredentialsCannotSignIn(t *testing.T) {
	handler := createFirstRunInstall(t)

	login := httptest.NewRecorder()
	handler.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/login", nil))
	empty := url.Values{"csrf": {csrfToken(t, login.Body.String())}, "username": {""}, "password": {""}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(empty.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code == http.StatusSeeOther {
		t.Fatalf("a blank sign-in was accepted: %d to %q", response.Code, response.Header().Get("Location"))
	}
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("a blank sign-in set a session cookie: %v", cookies)
	}

	basic := httptest.NewRequest(http.MethodGet, "/opds", nil)
	basic.SetBasicAuth("", "")
	basicResponse := httptest.NewRecorder()
	handler.ServeHTTP(basicResponse, basic)
	if basicResponse.Code != http.StatusUnauthorized {
		t.Fatalf("blank Basic credentials = %d, want 401", basicResponse.Code)
	}

	// The account that was actually created still works, so the guard rejects
	// only the empty pair rather than locking everyone out.
	again := httptest.NewRecorder()
	handler.ServeHTTP(again, httptest.NewRequest(http.MethodGet, "/login", nil))
	valid := url.Values{"csrf": {csrfToken(t, again.Body.String())}, "username": {"owner"}, "password": {"a long enough password"}}
	request = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(valid.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	signedIn := httptest.NewRecorder()
	handler.ServeHTTP(signedIn, request)
	if signedIn.Code != http.StatusSeeOther {
		t.Fatalf("the real administrator could not sign in: %d", signedIn.Code)
	}
}

// A Basic check runs bcrypt for tens of milliseconds. If an account is reset or
// disabled while that comparison is running, the result belongs to credentials
// that no longer exist and must not be memoized.
func TestABasicCheckThatLosesItsAccountIsNotRemembered(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	auth := newSessionAuthWith("", "", func(string, string) (accounts.Identity, bool) {
		close(started)
		<-release
		return accounts.Identity{ID: "reader", Username: "reader"}, true
	}, nil)

	result := make(chan bool, 1)
	go func() {
		_, valid := auth.basicIdentity("reader", "old password")
		result <- valid
	}()
	<-started
	auth.forgetCredentials()
	close(release)
	if <-result {
		t.Fatal("a check that finished after the account changed was accepted")
	}
	auth.mu.Lock()
	defer auth.mu.Unlock()
	if len(auth.credentials) != 0 {
		t.Fatalf("the stale check repopulated the memo: %d entries", len(auth.credentials))
	}
}

// A memo entry is answered in microseconds, so it must still check that the
// account it names exists right now: a disabled account stops downloading
// immediately rather than when its memo entry expires.
func TestACachedBasicIdentityStopsWorkingWhenTheAccountIsDisabled(t *testing.T) {
	store, err := accounts.Open(filepath.Join(t.TempDir(), "accounts.json"), "owner", "a long enough password")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := store.Create("reader", "another long password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	auth := newSessionAuthWith("", "", func(username, password string) (accounts.Identity, bool) {
		found, err := store.Authenticate(username, password)
		return found, err == nil
	}, func(candidate accounts.Identity) (accounts.Identity, bool) {
		return store.IdentityByID(candidate.ID)
	})
	if _, valid := auth.basicIdentity("reader", "another long password"); !valid {
		t.Fatal("the account could not sign in")
	}
	if err := store.SetDisabled(identity.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, valid := auth.basicIdentity("reader", "another long password"); valid {
		t.Fatal("a disabled account kept working from the memo")
	}
}

// A sign-in whose password was checked before a reset landed must not end with a
// session for the old password: the reset is the thing the user was told worked.
func TestASignInThatLosesAResetGetsNoSession(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	auth := newSessionAuthWith("", "", func(string, string) (accounts.Identity, bool) {
		close(started)
		<-release
		return accounts.Identity{ID: "reader", Username: "reader"}, true
	}, nil)
	handler := auth.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), func(*http.Request) bool { return false })

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/login", nil))
	form := url.Values{"csrf": {csrfToken(t, page.Body.String())}, "username": {"reader"}, "password": {"old password"}}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(response, request)
		close(done)
	}()
	// The reset happens while the password comparison is in flight.
	<-started
	auth.forgetCredentials()
	close(release)
	<-done
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("sign-in after a reset = %d, want 401", response.Code)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("sign-in after a reset issued a session: %v", cookies)
	}
}
