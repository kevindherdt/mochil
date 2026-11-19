package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A server with no credentials configured must not serve an open library: it asks for the first
// account instead, and only that page is reachable until the account exists.
func TestFreshInstallAsksForTheFirstAccount(t *testing.T) {
	dir := t.TempDir()
	handler, err := NewMulti(MultiConfig{ReadingState: filepath.Join(dir, "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	root := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, root)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/setup" {
		t.Fatalf("root = %d %q, want 303 to /setup", response.Code, response.Header().Get("Location"))
	}

	// Anything else is held behind the setup page rather than exposing the library.
	books := httptest.NewRecorder()
	handler.ServeHTTP(books, httptest.NewRequest(http.MethodGet, "/api/libraries", nil))
	if books.Code != http.StatusSeeOther || books.Header().Get("Location") != "/setup" {
		t.Fatalf("library list = %d %q, want 303 to /setup", books.Code, books.Header().Get("Location"))
	}

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/setup", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("setup page = %d", page.Code)
	}
	if !strings.Contains(page.Body.String(), `name="password"`) {
		t.Fatalf("setup page has no password field: %s", page.Body.String())
	}

	form := url.Values{"csrf": {csrfToken(t, page.Body.String())}, "username": {"owner"}, "password": {"a long enough password"}, "confirm": {"a long enough password"}}
	created := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(created, request)
	if created.Code != http.StatusSeeOther || created.Header().Get("Location") != "/login" {
		t.Fatalf("creating the account = %d %q, want 303 to /login", created.Code, created.Header().Get("Location"))
	}

	// The setup door closes once an administrator exists, and the library now asks for a sign-in.
	again := httptest.NewRecorder()
	handler.ServeHTTP(again, httptest.NewRequest(http.MethodGet, "/setup", nil))
	if again.Code != http.StatusSeeOther || again.Header().Get("Location") != "/login" {
		t.Fatalf("setup after creation = %d %q, want 303 to /login", again.Code, again.Header().Get("Location"))
	}

	login := httptest.NewRecorder()
	handler.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/login", nil))
	if login.Code != http.StatusOK {
		t.Fatalf("login page = %d", login.Code)
	}
	signedIn := url.Values{"csrf": {csrfToken(t, login.Body.String())}, "username": {"owner"}, "password": {"a long enough password"}}
	session := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(signedIn.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(session, request)
	if session.Code != http.StatusSeeOther {
		t.Fatalf("sign-in = %d %s", session.Code, session.Body.String())
	}
	cookies := session.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("sign-in set no session cookie")
	}
	after := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/libraries", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	handler.ServeHTTP(after, request)
	if after.Code != http.StatusOK {
		t.Fatalf("signed-in library list = %d, want 200", after.Code)
	}
}

// A mismatched confirmation must not create an account, and must not half-create one either.
func TestSetupRejectsAMismatchedConfirmation(t *testing.T) {
	dir := t.TempDir()
	handler, err := NewMulti(MultiConfig{ReadingState: filepath.Join(dir, "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/setup", nil))
	form := url.Values{"csrf": {csrfToken(t, page.Body.String())}, "username": {"owner"}, "password": {"a long enough password"}, "confirm": {"something else entirely"}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("mismatched confirmation = %d, want 400", response.Code)
	}
	// Still the setup page, because no account was created.
	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Header().Get("Location") != "/setup" {
		t.Fatalf("root = %q, want /setup", root.Header().Get("Location"))
	}
}

// An install that already has credentials keeps the behaviour it always had.
func TestConfiguredServerStillOpensTheLoginPage(t *testing.T) {
	dir := t.TempDir()
	handler, err := NewMulti(MultiConfig{Username: "owner", Password: "owner password", ReadingState: filepath.Join(dir, "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusSeeOther || !strings.HasPrefix(response.Header().Get("Location"), "/login") {
		t.Fatalf("root = %d %q, want 303 to /login", response.Code, response.Header().Get("Location"))
	}
}

var csrfPattern = regexp.MustCompile(`name="csrf" value="([^"]+)"`)

func csrfToken(t *testing.T, page string) string {
	t.Helper()
	match := csrfPattern.FindStringSubmatch(page)
	if len(match) != 2 {
		t.Fatalf("no csrf token in page: %s", page)
	}
	return match[1]
}

// The first-run page mints an administrator, so repeated failed attempts must lock out the same way
// the sign-in form does — otherwise it is an unauthenticated endpoint with unlimited tries.
func TestSetupLocksOutAfterRepeatedFailures(t *testing.T) {
	dir := t.TempDir()
	handler, err := NewMulti(MultiConfig{ReadingState: filepath.Join(dir, "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/setup", nil))

	attempt := func() *httptest.ResponseRecorder {
		form := url.Values{"csrf": {csrfToken(t, page.Body.String())}, "username": {"owner"},
			"password": {"a long enough password"}, "confirm": {"a different password"}}
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		handler.ServeHTTP(response, request)
		// Each page hands out a fresh token, and the token is consumed by the attempt.
		page = httptest.NewRecorder()
		handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/setup", nil))
		return response
	}

	for i := 0; i < 10; i++ {
		if code := attempt().Code; code != http.StatusBadRequest {
			t.Fatalf("attempt %d = %d, want 400", i+1, code)
		}
	}
	if code := attempt().Code; code != http.StatusTooManyRequests {
		t.Fatalf("attempt after the lock-out = %d, want 429", code)
	}
}

// A rejected first attempt (a password the policy refuses) must not leave the install unable to try
// again: the store is opened but not written until an account exists.
func TestASecondAttemptWorksAfterTheFirstIsRejected(t *testing.T) {
	dir := t.TempDir()
	handler, err := NewMulti(MultiConfig{ReadingState: filepath.Join(dir, "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	post := func(password string) int {
		page := httptest.NewRecorder()
		handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/setup", nil))
		form := url.Values{"csrf": {csrfToken(t, page.Body.String())}, "username": {"owner"},
			"password": {password}, "confirm": {password}}
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		handler.ServeHTTP(response, request)
		return response.Code
	}
	if code := post("short"); code != http.StatusBadRequest {
		t.Fatalf("a password the policy refuses = %d, want 400", code)
	}
	if code := post("a long enough password"); code != http.StatusSeeOther {
		t.Fatalf("the second attempt = %d, want 303", code)
	}
	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Header().Get("Location") != "/login?next=%2F" && !strings.HasPrefix(root.Header().Get("Location"), "/login") {
		t.Fatalf("after a successful second attempt root = %q", root.Header().Get("Location"))
	}
}
