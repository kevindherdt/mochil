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

// apiMe exists so an ordinary user can learn their own identity: the only endpoint that used to answer that was
// GET /api/accounts, which is administrator-only. Both halves of that are worth pinning down — the identity comes
// back for a non-administrator, and the same account still cannot use the administrator list.
func TestMeAnswersTheCallersOwnIdentityForAnyAccount(t *testing.T) {
	directory := t.TempDir()
	handler, err := NewMulti(MultiConfig{Username: "owner", Password: "owner password", ReadingState: filepath.Join(directory, "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()

	ownerCookie := loginCookie(t, handler, "owner", "owner password")
	create := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":"reader","password":"reader password","role":"user"}`))
	create.Header.Set("Content-Type", "application/json")
	create.AddCookie(ownerCookie)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", created.Code, created.Body.String())
	}

	readerCookie := loginCookie(t, handler, "reader", "reader password")

	me := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	me.AddCookie(readerCookie)
	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer, me)
	if answer.Code != http.StatusOK {
		t.Fatalf("me=%d %s", answer.Code, answer.Body.String())
	}
	body := answer.Body.String()
	if !strings.Contains(body, `"username":"reader"`) || !strings.Contains(body, `"role":"user"`) {
		t.Fatalf("me body = %s", body)
	}
	// The caller's own identity only: no password material, and nothing about any other account.
	if strings.Contains(body, "password") || strings.Contains(body, "owner") {
		t.Fatalf("me leaked more than the caller's identity: %s", body)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	list.AddCookie(readerCookie)
	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, list)
	if listed.Code == http.StatusOK {
		t.Fatalf("a non-administrator could list accounts: %d %s", listed.Code, listed.Body.String())
	}
}

func loginCookie(t *testing.T, handler http.Handler, username, password string) *http.Cookie {
	t.Helper()
	request := loginPOST(t, handler, url.Values{"username": {username}, "password": {password}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("sign-in as %s set no session cookie (status %d)", username, recorder.Code)
	}
	return cookies[0]
}
