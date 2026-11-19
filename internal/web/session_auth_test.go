package web

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"mochil/internal/accounts"
	"mochil/internal/catalog"
)

func TestWebUsesSessionLoginWithoutBasicChallenge(t *testing.T) {
	handler := authenticatedMulti(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("reader", "secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login?next=%2F" {
		t.Fatalf("unexpected web response: %d %q", response.Code, response.Header().Get("Location"))
	}
	if challenge := response.Header().Get("WWW-Authenticate"); challenge != "" {
		t.Fatalf("web route exposed a Basic Auth challenge: %q", challenge)
	}
}

func TestProcessHealthDoesNotRequireCredentials(t *testing.T) {
	handler := authenticatedMulti(t)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != "ok" {
		t.Fatalf("health response: %d %q", response.Code, response.Body.String())
	}
	if challenge := response.Header().Get("WWW-Authenticate"); challenge != "" {
		t.Fatalf("health route exposed a Basic Auth challenge: %q", challenge)
	}
}

func TestSessionAndBasicAuthAttachStableRequestIdentity(t *testing.T) {
	authenticate := func(username, password string) (accounts.Identity, bool) {
		if username == "reader" && password == "secret" {
			return accounts.Identity{ID: "acct_stable", Username: "reader", Role: accounts.RoleUser}, true
		}
		return accounts.Identity{}, false
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := identityFromRequest(r)
		if !ok {
			t.Error("request identity missing")
			return
		}
		_, _ = fmt.Fprintf(w, "%s:%s", identity.ID, identity.Role)
	})
	auth := newSessionAuthWith("bootstrap", "recovery", authenticate, nil).wrap(next, func(*http.Request) bool { return false })

	login := loginPOST(t, auth, url.Values{"username": {"reader"}, "password": {"secret"}})
	loginResponse := httptest.NewRecorder()
	auth.ServeHTTP(loginResponse, login)
	cookie := loginResponse.Result().Cookies()[0]
	browser := httptest.NewRequest(http.MethodGet, "/api/libraries", nil)
	browser.AddCookie(cookie)
	browserResponse := httptest.NewRecorder()
	auth.ServeHTTP(browserResponse, browser)
	if browserResponse.Body.String() != "acct_stable:user" {
		t.Fatalf("browser identity=%q", browserResponse.Body.String())
	}

	basic := httptest.NewRequest(http.MethodGet, "/opds", nil)
	basic.SetBasicAuth("reader", "secret")
	basicResponse := httptest.NewRecorder()
	auth.ServeHTTP(basicResponse, basic)
	if basicResponse.Body.String() != "acct_stable:user" {
		t.Fatalf("basic identity=%q", basicResponse.Body.String())
	}
}

func TestDisabledIdentityRevokesExistingBrowserSession(t *testing.T) {
	active := true
	identity := accounts.Identity{ID: "acct_reader", Username: "reader", Role: accounts.RoleUser}
	auth := newSessionAuthWith("bootstrap", "recovery", func(username, password string) (accounts.Identity, bool) {
		return identity, username == "reader" && password == "secret"
	}, func(candidate accounts.Identity) (accounts.Identity, bool) { return candidate, active }).wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), func(*http.Request) bool { return false })
	login := loginPOST(t, auth, url.Values{"username": {"reader"}, "password": {"secret"}})
	response := httptest.NewRecorder()
	auth.ServeHTTP(response, login)
	cookie := response.Result().Cookies()[0]
	active = false
	request := httptest.NewRequest(http.MethodGet, "/settings", nil)
	request.AddCookie(cookie)
	denied := httptest.NewRecorder()
	auth.ServeHTTP(denied, request)
	if denied.Code != http.StatusSeeOther || !strings.HasPrefix(denied.Header().Get("Location"), "/login") {
		t.Fatalf("disabled session=%d %q", denied.Code, denied.Header().Get("Location"))
	}
}

func TestLoginCreatesSecureSessionAndLogoutInvalidatesIt(t *testing.T) {
	handler := authenticatedMulti(t)
	form := url.Values{"username": {"reader"}, "password": {"secret"}, "next": {"/api/libraries"}}
	request := loginPOST(t, handler, form)
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/api/libraries" {
		t.Fatalf("unexpected login response: %d %q", response.Code, response.Header().Get("Location"))
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("unsafe session cookie: %#v", cookies)
	}

	authorized := httptest.NewRequest(http.MethodGet, "/api/libraries", nil)
	authorized.AddCookie(cookies[0])
	authorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(authorizedResponse, authorized)
	if authorizedResponse.Code != http.StatusOK {
		t.Fatalf("session was not accepted: %d", authorizedResponse.Code)
	}

	logout := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logout.AddCookie(cookies[0])
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logout)
	if logoutResponse.Code != http.StatusSeeOther {
		t.Fatalf("logout status: %d", logoutResponse.Code)
	}

	reused := httptest.NewRequest(http.MethodGet, "/api/libraries", nil)
	reused.AddCookie(cookies[0])
	reusedResponse := httptest.NewRecorder()
	handler.ServeHTTP(reusedResponse, reused)
	if reusedResponse.Code != http.StatusSeeOther {
		t.Fatalf("logged-out session remained valid: %d", reusedResponse.Code)
	}
}

func TestSessionContinuityPreservesLoginAcrossHandlerReplacement(t *testing.T) {
	continuity := NewSessionContinuity()
	first := continuity.wrap("reader", "secret", nil, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), func(*http.Request) bool { return false })
	login := loginPOST(t, first, url.Values{"username": {"reader"}, "password": {"secret"}})
	response := httptest.NewRecorder()
	first.ServeHTTP(response, login)
	cookies := response.Result().Cookies()
	if response.Code != http.StatusSeeOther || len(cookies) != 1 {
		t.Fatalf("login=%d cookies=%d", response.Code, len(cookies))
	}

	second := continuity.wrap("reader", "secret", nil, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) }), func(*http.Request) bool { return false })
	request := httptest.NewRequest(http.MethodGet, "/settings", nil)
	request.AddCookie(cookies[0])
	replaced := httptest.NewRecorder()
	second.ServeHTTP(replaced, request)
	if replaced.Code != http.StatusAccepted {
		t.Fatalf("session did not survive handler replacement: %d", replaced.Code)
	}
}

func TestOPDSRetainsBasicAuthForReaderClients(t *testing.T) {
	handler := authenticatedMulti(t)
	request := httptest.NewRequest(http.MethodGet, "/opds", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Header().Get("WWW-Authenticate"), "Mochil OPDS") {
		t.Fatalf("missing OPDS challenge: %d %q", response.Code, response.Header().Get("WWW-Authenticate"))
	}

	authorized := httptest.NewRequest(http.MethodGet, "/opds", nil)
	authorized.SetBasicAuth("reader", "secret")
	authorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(authorizedResponse, authorized)
	if authorizedResponse.Code != http.StatusOK {
		t.Fatalf("OPDS Basic Auth was rejected: %d", authorizedResponse.Code)
	}
}

func TestBasicAuthRateLimitCannotBeBypassedWithForwardedAddress(t *testing.T) {
	handler := newSessionAuth("reader", "secret").wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), func(*http.Request) bool { return false })
	for attempt := 0; attempt < 10; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "/opds", nil)
		request.RemoteAddr = "192.0.2.10:" + strconv.Itoa(4000+attempt)
		request.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", attempt+1))
		request.SetBasicAuth("reader", "wrong")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d returned %d", attempt+1, response.Code)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/opds", nil)
	request.RemoteAddr = "192.0.2.10:9000"
	request.Header.Set("X-Forwarded-For", "203.0.113.77")
	request.SetBasicAuth("reader", "secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") == "" {
		t.Fatalf("rate-limited response = %d, Retry-After=%q", response.Code, response.Header().Get("Retry-After"))
	}
}

func TestLoginPageDoesNotRenderConfiguredCredentials(t *testing.T) {
	handler := authenticatedMulti(t)
	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	if strings.Contains(body, "reader") || strings.Contains(body, "secret") || !strings.Contains(body, `autocomplete="current-password"`) || !strings.Contains(body, `autocomplete="username"`) {
		t.Fatalf("login page rendered credentials or omitted password guidance: %s", body)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("login page may be cached: %q", response.Header().Get("Cache-Control"))
	}
	if !strings.Contains(response.Header().Get("Permissions-Policy"), "camera=()") || response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("login page omitted browser capability or framing restrictions: %v", response.Header())
	}
}

func TestLoginRejectsMissingCSRFAndExternalNext(t *testing.T) {
	handler := authenticatedMulti(t)
	form := url.Values{"username": {"reader"}, "password": {"secret"}, "next": {"https://evil.example/"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing-CSRF login status: %d", response.Code)
	}

	request = loginPOST(t, handler, form)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Header().Get("Location") != "/" {
		t.Fatalf("external redirect was accepted: %q", response.Header().Get("Location"))
	}
}

func loginPOST(t *testing.T, handler http.Handler, form url.Values) *http.Request {
	t.Helper()
	page := httptest.NewRequest(http.MethodGet, "/login", nil)
	pageResponse := httptest.NewRecorder()
	handler.ServeHTTP(pageResponse, page)
	body := pageResponse.Body.String()
	marker := `name="csrf" value="`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatal("login page did not render CSRF token")
	}
	start += len(marker)
	end := strings.IndexByte(body[start:], '"')
	if end < 0 {
		t.Fatal("login page rendered malformed CSRF token")
	}
	form.Set("csrf", body[start:start+end])
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

func TestLoginTokenWorksWithoutCookiesAndCannotBeReplayed(t *testing.T) {
	handler := authenticatedMulti(t)
	form := url.Values{"username": {"reader"}, "password": {"secret"}}
	request := loginPOST(t, handler, form)
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	request.Body = io.NopCloser(strings.NewReader(string(body)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("cookie-free login status = %d", response.Code)
	}
	replay := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(string(body)))
	replay.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	replayed := httptest.NewRecorder()
	handler.ServeHTTP(replayed, replay)
	if replayed.Code != http.StatusBadRequest || !strings.Contains(replayed.Body.String(), "expired") {
		t.Fatalf("replayed login = %d %s", replayed.Code, replayed.Body.String())
	}
}

func TestLoginTokenIsNotCoupledToProxyOrBrowserOriginHeaders(t *testing.T) {
	handler := authenticatedMulti(t)
	request := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}})
	request.Header.Set("Origin", "https://attacker.example")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("login with rewritten origin headers = %d %s", response.Code, response.Body.String())
	}
}

func TestLoginAcceptsSameOriginBrowserSignalBehindRewrittenProxyHost(t *testing.T) {
	handler := authenticatedMulti(t)
	request := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}})
	request.Host = "internal-service:8080"
	request.Header.Set("Origin", "https://opds.example")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("proxied same-origin login = %d %s", response.Code, response.Body.String())
	}
}

func authenticatedMulti(t *testing.T) http.Handler {
	t.Helper()
	handler, err := NewMulti(MultiConfig{Username: "reader", Password: "secret"}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closer, ok := handler.(interface{ Close() }); ok {
			closer.Close()
		}
	})
	return handler
}

func TestRememberMeKeepsTheSessionForAMonthAndOtherwiseTwelveHours(t *testing.T) {
	for _, tc := range []struct {
		remember string
		want     time.Duration
	}{{"1", rememberLifetime}, {"", sessionLifetime}} {
		handler := authenticatedMulti(t)
		form := url.Values{"username": {"reader"}, "password": {"secret"}, "next": {"/"}}
		if tc.remember != "" {
			form.Set("remember", tc.remember)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, loginPOST(t, handler, form))
		cookies := response.Result().Cookies()
		if len(cookies) != 1 || cookies[0].MaxAge != int(tc.want.Seconds()) {
			t.Fatalf("remember=%q cookie = %#v, want MaxAge %v", tc.remember, cookies, tc.want)
		}
	}
	short, long := webSession{created: time.Now().Add(-3 * time.Hour), seen: time.Now().Add(-3 * time.Hour)}, webSession{created: time.Now().Add(-3 * time.Hour), seen: time.Now().Add(-3 * time.Hour), remember: true}
	if !short.expired(time.Now()) || long.expired(time.Now()) {
		t.Fatalf("an idle ordinary session should expire and a remembered one should not")
	}
}

func TestLoginPageIgnoresTheRetiredLoginBackgroundSetting(t *testing.T) {
	config := []byte(`{"appearance":{"login_background":"spines"},"libraries":[]}`)
	handler, err := NewMulti(MultiConfig{Username: "reader", Password: "secret",
		ConfigJSON: func() ([]byte, error) { return config, nil },
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/login", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "Welcome back") {
		t.Fatalf("login page: %d %s", response.Code, body)
	}
	if strings.Contains(body, "spines") || strings.Contains(body, "data-login") {
		t.Fatalf("a retired setting still changes the sign-in page: %s", body)
	}
}
