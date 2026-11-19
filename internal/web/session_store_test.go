package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"mochil/internal/accounts"
)

func continuity(path string) (*SessionContinuity, http.Handler) {
	sessions := NewSessionContinuity()
	sessions.SetStorePath(path)
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return sessions, sessions.wrap("reader", "secret", nil, nil, ok, func(*http.Request) bool { return false })
}

func signIn(t *testing.T, handler http.Handler, remember bool) *http.Cookie {
	t.Helper()
	form := url.Values{"username": {"reader"}, "password": {"secret"}, "next": {"/"}}
	if remember {
		form.Set("remember", "1")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, loginPOST(t, handler, form))
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("sign-in cookies = %#v", cookies)
	}
	return cookies[0]
}

func openWith(handler http.Handler, cookie *http.Cookie) int {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func TestARememberedSignInSurvivesARestartButAnOrdinaryOneDoesNot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	_, before := continuity(path)
	remembered, ordinary := signIn(t, before, true), signIn(t, before, false)
	if openWith(before, remembered) != http.StatusNoContent || openWith(before, ordinary) != http.StatusNoContent {
		t.Fatal("both sign-ins should work before the restart")
	}
	_, after := continuity(path) // a fresh process reading the same file
	if got := openWith(after, remembered); got != http.StatusNoContent {
		t.Fatalf("the remembered sign-in should survive a restart, got %d", got)
	}
	if got := openWith(after, ordinary); got == http.StatusNoContent {
		t.Fatal("an ordinary sign-in must not survive a restart")
	}
}

func TestTheSessionFileHoldsOnlyHashesIsPrivateAndDiesWithAPasswordChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	_, before := continuity(path)
	cookie := signIn(t, before, true)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), cookie.Value) || strings.Contains(string(data), "secret") {
		t.Fatalf("the file must hold neither the token nor the password: %s", data)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("the file must be private, got %v", info.Mode().Perm())
	}
	changed := NewSessionContinuity()
	changed.SetStorePath(path)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := changed.wrap("reader", "another-password", nil, nil, next, func(*http.Request) bool { return false })
	if got := openWith(handler, cookie); got == http.StatusNoContent {
		t.Fatal("a password change must end remembered sign-ins")
	}
}

func TestSigningOutRemovesTheRememberedSessionFromDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	_, before := continuity(path)
	cookie := signIn(t, before, true)
	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(cookie)
	before.ServeHTTP(httptest.NewRecorder(), request)
	_, after := continuity(path)
	if got := openWith(after, cookie); got == http.StatusNoContent {
		t.Fatal("a signed-out session must not come back after a restart")
	}
}

func TestLogoutReportsFailureAndCanBeRetriedAfterStoreRecovers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	_, handler := continuity(path)
	cookie := signIn(t, handler, true)
	if got := openWith(handler, cookie); got != http.StatusNoContent {
		t.Fatalf("remembered login unavailable before logout: %d", got)
	}

	// A directory at the target path makes the atomic rename fail even as root.
	backup := filepath.Join(dir, "sessions.backup")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	logout := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/logout", nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	failed := logout()
	if failed.Code != http.StatusServiceUnavailable || !strings.Contains(failed.Body.String(), "may become valid again") || !strings.Contains(failed.Body.String(), "retry") {
		t.Fatalf("failed logout must warn about restart and allow retry: status=%d body=%q", failed.Code, failed.Body.String())
	}
	if !strings.Contains(failed.Header().Get("Content-Type"), "text/html") || !strings.Contains(failed.Body.String(), `<form method="post" action="/logout">`) {
		t.Fatalf("failed logout must provide a same-origin POST retry form: type=%q body=%q", failed.Header().Get("Content-Type"), failed.Body.String())
	}
	if failed.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("failed logout warning may be cached: %q", failed.Header().Get("Cache-Control"))
	}
	if len(failed.Result().Cookies()) != 0 {
		t.Fatal("failed logout discarded the cookie needed to retry")
	}
	if got := openWith(handler, cookie); got != http.StatusSeeOther {
		t.Fatalf("failed logout must revoke the live session, got %d", got)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	retried := logout()
	if retried.Code != http.StatusSeeOther || retried.Header().Get("Location") != "/login" {
		t.Fatalf("retry logout: status=%d location=%q", retried.Code, retried.Header().Get("Location"))
	}
	if cookies := retried.Result().Cookies(); len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("successful retry did not clear cookie: %#v", cookies)
	}
	_, restarted := continuity(path)
	if got := openWith(restarted, cookie); got != http.StatusSeeOther {
		t.Fatalf("logged-out session revived after restart: %d", got)
	}
}

func TestFailedLogoutBlocksLoginUntilRevocationIsPersisted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	_, handler := continuity(path)
	cookie := signIn(t, handler, true)
	backup := filepath.Join(dir, "sessions.backup")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	logout := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logout.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, logout)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("failed logout status = %d", response.Code)
	}
	login := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}, "next": {"/"}})
	login.AddCookie(cookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, login)
	if response.Code != http.StatusServiceUnavailable || len(response.Result().Cookies()) != 0 {
		t.Fatalf("login overwrote pending logout cookie: status=%d cookie count=%d", response.Code, len(response.Result().Cookies()))
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	login = loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}, "next": {"/"}})
	login.AddCookie(cookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, login)
	if response.Code != http.StatusSeeOther || len(response.Result().Cookies()) != 1 {
		t.Fatalf("login after storage recovery: status=%d cookie count=%d", response.Code, len(response.Result().Cookies()))
	}
	_, restarted := continuity(path)
	if got := openWith(restarted, cookie); got == http.StatusNoContent {
		t.Fatal("old cookie revived after restart")
	}
}

func TestConcurrentLoginAndFailedLogoutDoNotOverwriteRevocationCookie(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	continuity, handler := continuity(path)
	cookie := signIn(t, handler, true)
	backup := filepath.Join(dir, "sessions.backup")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	login := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}, "next": {"/"}})
	login.AddCookie(cookie)
	logout := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logout.AddCookie(cookie)
	// Hold persistence while logout removes the old session, then race login with it.
	continuity.auth.persistMu.Lock()
	logoutResponse := httptest.NewRecorder()
	logoutDone := make(chan struct{})
	go func() { handler.ServeHTTP(logoutResponse, logout); close(logoutDone) }()
	deadline := time.After(5 * time.Second)
	for {
		continuity.auth.mu.Lock()
		_, exists := continuity.auth.sessions[sessionKey(cookie.Value)]
		continuity.auth.mu.Unlock()
		if !exists {
			break
		}
		select {
		case <-deadline:
			continuity.auth.persistMu.Unlock()
			t.Fatal("logout did not begin")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	loginResponse := httptest.NewRecorder()
	loginDone := make(chan struct{})
	go func() { handler.ServeHTTP(loginResponse, login); close(loginDone) }()
	continuity.auth.persistMu.Unlock()
	<-logoutDone
	<-loginDone
	if logoutResponse.Code != http.StatusServiceUnavailable || loginResponse.Code != http.StatusServiceUnavailable || len(loginResponse.Result().Cookies()) != 0 {
		t.Fatalf("concurrent logout/login: logout=%d login=%d cookie count=%d", logoutResponse.Code, loginResponse.Code, len(loginResponse.Result().Cookies()))
	}
}

func TestLogoutStaysLoggedOutAfterConcurrentPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	identity := accounts.Identity{ID: "acct_reader", Username: "reader", Role: accounts.RoleUser}
	auth := newSessionAuth("owner", "pw")
	auth.store = newSessionStore(path, "owner", "pw")
	token, _ := auth.createSession(identity, true)
	var wait sync.WaitGroup
	for range 20 {
		wait.Add(1)
		go func() { defer wait.Done(); auth.persist() }()
	}
	auth.mu.Lock()
	delete(auth.sessions, sessionKey(token))
	auth.mu.Unlock()
	auth.persist()
	wait.Wait()
	if len(newSessionStore(path, "owner", "pw").load(time.Now())) != 0 {
		t.Fatal("a signed-out remembered session came back from disk")
	}
}
