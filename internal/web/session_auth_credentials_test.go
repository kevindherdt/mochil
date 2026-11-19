package web

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"mochil/internal/accounts"
)

func basicClientAuth(t *testing.T, authenticate func(string, string) (accounts.Identity, bool)) (*sessionAuth, http.Handler) {
	t.Helper()
	auth := newSessionAuthWith("reader", "secret", authenticate, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return auth, auth.wrap(next, func(*http.Request) bool { return false })
}

func basicRequest(handler http.Handler, username, password string) int {
	request := httptest.NewRequest(http.MethodGet, "/opds", nil)
	request.SetBasicAuth(username, password)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func TestBasicAuthChecksACorrectPasswordOncePerLifetime(t *testing.T) {
	var checks atomic.Int32
	auth, handler := basicClientAuth(t, func(username, password string) (accounts.Identity, bool) {
		checks.Add(1)
		return accounts.Identity{ID: "a1", Username: "reader"}, username == "reader" && password == "secret"
	})
	now := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	auth.now = func() time.Time { return now }

	for range 3 {
		if code := basicRequest(handler, "reader", "secret"); code != http.StatusNoContent {
			t.Fatalf("status = %d", code)
		}
	}
	if got := checks.Load(); got != 1 {
		t.Fatalf("password checked %d times for one repeated credential, want 1", got)
	}

	// The memo is short-lived: past its lifetime the password is checked again.
	now = now.Add(basicCredentialLifetime + time.Second)
	if code := basicRequest(handler, "reader", "secret"); code != http.StatusNoContent {
		t.Fatalf("status after expiry = %d", code)
	}
	if got := checks.Load(); got != 2 {
		t.Fatalf("password checked %d times after expiry, want 2", got)
	}
}

func TestBasicAuthNeverCachesAWrongPassword(t *testing.T) {
	var checks atomic.Int32
	_, handler := basicClientAuth(t, func(username, password string) (accounts.Identity, bool) {
		checks.Add(1)
		return accounts.Identity{ID: "a1", Username: "reader"}, username == "reader" && password == "secret"
	})
	for range 2 {
		if code := basicRequest(handler, "reader", "wrong"); code != http.StatusUnauthorized {
			t.Fatalf("wrong password status = %d", code)
		}
	}
	if got := checks.Load(); got != 2 {
		t.Fatalf("wrong password was checked %d times, want 2: a rejected guess must not be memoized", got)
	}
	if code := basicRequest(handler, "reader", "secret"); code != http.StatusNoContent {
		t.Fatalf("correct password status = %d", code)
	}
	if code := basicRequest(handler, "reader", "wrong"); code != http.StatusUnauthorized {
		t.Fatalf("wrong password status after a success = %d", code)
	}
	if got := checks.Load(); got != 4 {
		t.Fatalf("password checks = %d, want 4", got)
	}
}

func TestAccountChangesDiscardCachedBasicCredentials(t *testing.T) {
	directory := t.TempDir()
	handler, err := NewMulti(MultiConfig{
		Username:     "owner",
		Password:     "owner password",
		ReadingState: filepath.Join(directory, "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()

	login := httptest.NewRecorder()
	handler.ServeHTTP(login, loginPOST(t, handler, url.Values{"username": {"owner"}, "password": {"owner password"}, "next": {"/"}}))
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("sign-in cookies = %#v", cookies)
	}
	cookie := cookies[0]

	create := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewBufferString(`{"username":"reader","password":"reader password","role":"user"}`))
	create.Header.Set("Content-Type", "application/json")
	create.AddCookie(cookie)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	var identity struct{ ID string }
	if err := json.Unmarshal(created.Body.Bytes(), &identity); err != nil {
		t.Fatal(err)
	}

	// A successful OPDS request fills the Basic-auth memo.
	if code := basicRequest(handler, "reader", "reader password"); code != http.StatusOK {
		t.Fatalf("reader basic auth = %d", code)
	}

	action := func(body string) int {
		patch := httptest.NewRequest(http.MethodPatch, "/api/accounts", bytes.NewBufferString(body))
		patch.Header.Set("Content-Type", "application/json")
		patch.AddCookie(cookie)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, patch)
		return response.Code
	}

	if code := action(`{"id":"` + identity.ID + `","action":"reset_password","password":"changed password"}`); code != http.StatusNoContent {
		t.Fatalf("reset password = %d", code)
	}
	if code := basicRequest(handler, "reader", "reader password"); code != http.StatusUnauthorized {
		t.Fatalf("the old password still worked after a reset (status %d): a cached check outlived the change", code)
	}
	if code := basicRequest(handler, "reader", "changed password"); code != http.StatusOK {
		t.Fatalf("the new password did not work (status %d)", code)
	}

	if code := action(`{"id":"` + identity.ID + `","action":"disable"}`); code != http.StatusNoContent {
		t.Fatalf("disable = %d", code)
	}
	if code := basicRequest(handler, "reader", "changed password"); code != http.StatusUnauthorized {
		t.Fatalf("a disabled account still authenticated from the cache (status %d)", code)
	}
}
