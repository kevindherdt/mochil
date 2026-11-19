package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCookieLessConcurrentLoginWaitsForFailedLogout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	sessions, handler := continuity(path)
	cookie := signIn(t, handler, true)
	backup := filepath.Join(dir, "backup")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	login := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}})
	logout := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logout.AddCookie(cookie)
	sessions.auth.persistMu.Lock()
	logoutResult := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, logout)
		logoutResult <- response
	}()
	deadline := time.After(5 * time.Second)
	for {
		sessions.auth.mu.Lock()
		_, pending := sessions.auth.pendingRevocations[sessionKey(cookie.Value)]
		sessions.auth.mu.Unlock()
		if pending {
			break
		}
		select {
		case <-deadline:
			sessions.auth.persistMu.Unlock()
			t.Fatal("logout did not mark pending")
		default:
			runtime.Gosched()
		}
	}
	loginResult := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, login)
		loginResult <- response
	}()
	sessions.auth.persistMu.Unlock()
	failed, blocked := <-logoutResult, <-loginResult
	if failed.Code != http.StatusServiceUnavailable || blocked.Code != http.StatusServiceUnavailable || len(blocked.Result().Cookies()) != 0 {
		t.Fatalf("logout=%d cookie-less login=%d cookies=%d", failed.Code, blocked.Code, len(blocked.Result().Cookies()))
	}
}

func TestSnapshotDoesNotClearLaterRevocation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	sessions, handler := continuity(path)
	cookie := signIn(t, handler, true)
	sessions.auth.mu.Lock()
	previousSave := sessions.auth.lastSaved
	sessions.auth.mu.Unlock()
	sessions.auth.store.mu.Lock()
	persistDone := make(chan error, 1)
	go func() { persistDone <- sessions.auth.persist() }()
	deadline := time.After(5 * time.Second)
	for {
		sessions.auth.mu.Lock()
		snapshotted := sessions.auth.lastSaved.After(previousSave)
		sessions.auth.mu.Unlock()
		if snapshotted {
			break
		}
		select {
		case <-deadline:
			sessions.auth.store.mu.Unlock()
			t.Fatal("persist did not snapshot")
		default:
			runtime.Gosched()
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(cookie)
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() { response := httptest.NewRecorder(); handler.ServeHTTP(response, request); result <- response }()
	for {
		sessions.auth.mu.Lock()
		_, pending := sessions.auth.pendingRevocations[sessionKey(cookie.Value)]
		sessions.auth.mu.Unlock()
		if pending {
			break
		}
		select {
		case <-deadline:
			sessions.auth.store.mu.Unlock()
			t.Fatal("logout did not mark pending")
		default:
			runtime.Gosched()
		}
	}
	// The later logout must write after the blocked snapshot, not before it.
	sessions.auth.store.mu.Unlock()
	if err := <-persistDone; err != nil {
		t.Fatal(err)
	}
	response := <-result
	if response.Code != http.StatusSeeOther {
		t.Fatalf("logout status=%d", response.Code)
	}
	_, restarted := continuity(path)
	if got := openWith(restarted, cookie); got != http.StatusSeeOther {
		t.Fatalf("concurrent snapshot revived session: %d", got)
	}
}

func TestPersistedButEvictedSessionCanBeRevokedDuringStoreFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	sessions, handler := continuity(path)
	cookie := signIn(t, handler, true)
	sessions.auth.mu.Lock()
	delete(sessions.auth.sessions, sessionKey(cookie.Value))
	sessions.auth.mu.Unlock()
	backup := filepath.Join(dir, "backup")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || len(response.Result().Cookies()) != 0 {
		t.Fatalf("logout status=%d cookies=%d", response.Code, len(response.Result().Cookies()))
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("retry status=%d", response.Code)
	}
	_, restarted := continuity(path)
	if got := openWith(restarted, cookie); got != http.StatusSeeOther {
		t.Fatalf("revoked session revived: %d", got)
	}
}

func TestSuccessfulPersistenceClearsOnlyRevocationsInItsSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	sessions, handler := continuity(path)
	first := signIn(t, handler, true)
	second := signIn(t, handler, true)
	backup := filepath.Join(dir, "backup")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range []*http.Cookie{first, second} {
		request := httptest.NewRequest(http.MethodPost, "/logout", nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("logout status=%d", response.Code)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	if err := sessions.auth.persist(); err != nil {
		t.Fatal(err)
	}
	sessions.auth.mu.Lock()
	count := len(sessions.auth.pendingRevocations)
	sessions.auth.mu.Unlock()
	if count != 0 {
		t.Fatalf("persisted revocations still pending=%d", count)
	}
	_, restarted := continuity(path)
	for _, cookie := range []*http.Cookie{first, second} {
		if got := openWith(restarted, cookie); got != http.StatusSeeOther {
			t.Fatalf("revoked session revived: %d", got)
		}
	}
}

func TestCookieLessLoginFailsClosedWhileRevocationCannotPersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	sessions, handler := continuity(path)
	cookie := signIn(t, handler, true)
	backup := filepath.Join(dir, "backup")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	logout := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logout.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, logout)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("logout status=%d", response.Code)
	}
	login := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, login)
	if response.Code != http.StatusServiceUnavailable || len(response.Result().Cookies()) != 0 {
		t.Fatalf("cookie-less login status=%d cookies=%d", response.Code, len(response.Result().Cookies()))
	}
	sessions.auth.mu.Lock()
	_, retained := sessions.auth.pendingRevocations[sessionKey(cookie.Value)]
	sessions.auth.mu.Unlock()
	if !retained {
		t.Fatal("failed login lost pending retry")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	login = loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, login)
	if response.Code != http.StatusSeeOther || len(response.Result().Cookies()) != 1 {
		t.Fatalf("recovered login status=%d cookies=%d", response.Code, len(response.Result().Cookies()))
	}
	_, restarted := continuity(path)
	if got := openWith(restarted, cookie); got != http.StatusSeeOther {
		t.Fatalf("revoked cookie revived after restart: %d", got)
	}
}

func TestUnknownLogoutCookiesDoNotAccumulatePendingRevocations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	continuity, handler := continuity(path)
	cookie := signIn(t, handler, true)
	backup := filepath.Join(dir, "backup")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(cookie)
	failed := httptest.NewRecorder()
	handler.ServeHTTP(failed, request)
	if failed.Code != http.StatusServiceUnavailable {
		t.Fatalf("known logout status=%d", failed.Code)
	}
	for i := 0; i < maxSessions*2; i++ {
		request := httptest.NewRequest(http.MethodPost, "/logout", nil)
		request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: fmt.Sprintf("unknown-%d", i)})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("unknown logout status=%d", response.Code)
		}
	}
	continuity.auth.mu.Lock()
	count := len(continuity.auth.pendingRevocations)
	_, retained := continuity.auth.pendingRevocations[sessionKey(cookie.Value)]
	continuity.auth.mu.Unlock()
	if count != 1 || !retained {
		t.Fatalf("pending revocations=%d, known retry retained=%v; want 1, true", count, retained)
	}
}
