package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mochil/internal/accounts"
)

const (
	loginPath          = "/login"
	sessionCookieName  = "bobobooks_session"
	sessionLifetime    = 12 * time.Hour
	sessionIdleLimit   = 2 * time.Hour
	rememberLifetime   = 30 * 24 * time.Hour // "Remember me": one month, as long as it is used at least that often
	loginTokenLifetime = 10 * time.Minute
	maxSessions        = 128
	maxLoginTokens     = 512
)

type webSession struct {
	created  time.Time
	seen     time.Time
	identity accounts.Identity
	remember bool
}

// expired applies the short limits to an ordinary sign-in and the long one to "Remember me".
func (s webSession) expired(now time.Time) bool {
	if s.remember {
		return now.Sub(s.created) > rememberLifetime || now.Sub(s.seen) > rememberLifetime
	}
	return now.Sub(s.created) > sessionLifetime || now.Sub(s.seen) > sessionIdleLimit
}

type sessionAuth struct {
	username        string
	password        string
	authenticate    func(string, string) (accounts.Identity, bool)
	resolveIdentity func(accounts.Identity) (accounts.Identity, bool)
	authenticateMu  sync.RWMutex
	now             func() time.Time

	persistMu          sync.Mutex // snapshot and write happen as one step, so an older snapshot cannot land after a newer one
	transitionMu       sync.Mutex // serializes login cookie issuance with logout revocation
	mu                 sync.Mutex
	pendingRevocations map[[32]byte]struct{}
	persistedSessions  map[[32]byte]webSession // last successfully loaded or saved disk snapshot
	sessions           map[[32]byte]webSession
	store              *sessionStore // nil: sessions live only in memory
	lastSaved          time.Time
	loginTokens        map[[32]byte]time.Time
	failures           map[string]loginAttempts
	// credentials memoizes successful Basic-auth checks by credential digest,
	// bounded and short-lived, so a reader streaming a book does not run bcrypt
	// per request. Every account mutation clears it.
	credentials map[[32]byte]basicCredential
	// credentialGeneration counts account mutations that clear the memo. A check
	// captures it before the slow password comparison and refuses its own result
	// if the generation moved, so a reset that lands mid-bcrypt cannot have its
	// old credentials re-inserted into the map it just cleared.
	credentialGeneration atomic.Uint64
	// setup answers "does this server still need its first account", and createFirstAccount is how
	// the first-run page makes it. Both nil on an install that was given credentials.
	setup              func() bool
	createFirstAccount func(username, password string) error
}

// SessionContinuity owns browser sessions across live handler replacements.
// Credentials changing deliberately creates a fresh boundary; ordinary settings
// changes keep the existing authenticated browser sessions.
type SessionContinuity struct {
	mu                 sync.Mutex
	setup              func() bool
	createFirstAccount func(username, password string) error
	username           string
	password           string
	auth               *sessionAuth
	storePath          string
}

func NewSessionContinuity() *SessionContinuity { return &SessionContinuity{} }

// SetStorePath makes "Remember me" sessions survive restarts, kept in a private file at path.
func (s *SessionContinuity) SetStorePath(path string) {
	s.mu.Lock()
	s.storePath = path
	s.mu.Unlock()
}

func (s *SessionContinuity) revokeAccount(account string) error {
	s.mu.Lock()
	auth := s.auth
	s.mu.Unlock()
	if auth == nil {
		return nil
	}
	return auth.revokeAccount(account)
}

// forgetCredentials drops memoized Basic-auth results after an account change,
// so the next request checks the password again instead of trusting the memo.
func (s *SessionContinuity) forgetCredentials() {
	s.mu.Lock()
	auth := s.auth
	s.mu.Unlock()
	if auth != nil {
		auth.forgetCredentials()
	}
}

// setSetup hands the first-run hooks to the live session layer, so a handler rebuilt by a settings
// change keeps asking for the first account until one exists.
func (s *SessionContinuity) setSetup(required func() bool, create func(username, password string) error) {
	s.mu.Lock()
	s.setup, s.createFirstAccount = required, create
	s.mu.Unlock()
}

func (s *SessionContinuity) wrap(username, password string, authenticate func(string, string) (accounts.Identity, bool), resolveIdentity func(accounts.Identity) (accounts.Identity, bool), next http.Handler, bypass func(*http.Request) bool) http.Handler {
	s.mu.Lock()
	if s.auth == nil || s.username != username || s.password != password {
		s.username, s.password = username, password
		if authenticate == nil {
			s.auth = newSessionAuth(username, password)
		} else {
			s.auth = newSessionAuthWith(username, password, authenticate, resolveIdentity)
		}
	}
	if s.auth.store == nil && s.storePath != "" {
		s.auth.store = newSessionStore(s.storePath, s.username, s.password)
		s.auth.sessions = s.auth.store.load(time.Now())
		s.auth.persistedSessions = make(map[[32]byte]webSession, len(s.auth.sessions))
		for key, session := range s.auth.sessions {
			s.auth.persistedSessions[key] = session
		}
	}
	auth := s.auth
	setup, createFirstAccount := s.setup, s.createFirstAccount
	s.mu.Unlock()
	auth.setSetup(setup, createFirstAccount)
	if authenticate != nil {
		auth.setAuthenticator(authenticate, resolveIdentity)
	}
	return auth.wrap(next, bypass)
}

func (a *sessionAuth) setAuthenticator(authenticate func(string, string) (accounts.Identity, bool), resolveIdentity func(accounts.Identity) (accounts.Identity, bool)) {
	a.authenticateMu.Lock()
	a.authenticate = authenticate
	a.resolveIdentity = resolveIdentity
	a.authenticateMu.Unlock()
}

func (a *sessionAuth) resolve(identity accounts.Identity) (accounts.Identity, bool) {
	a.authenticateMu.RLock()
	resolveIdentity := a.resolveIdentity
	a.authenticateMu.RUnlock()
	if resolveIdentity == nil {
		return identity, identity.ID != ""
	}
	return resolveIdentity(identity)
}

func (a *sessionAuth) authenticateCredentials(username, password string) (accounts.Identity, bool) {
	a.authenticateMu.RLock()
	authenticate := a.authenticate
	a.authenticateMu.RUnlock()
	return authenticate(username, password)
}

func newSessionAuth(username, password string) *sessionAuth {
	return newSessionAuthWith(username, password, func(candidateUsername, candidatePassword string) (accounts.Identity, bool) {
		goodUser := subtle.ConstantTimeCompare([]byte(candidateUsername), []byte(username)) == 1
		goodPass := subtle.ConstantTimeCompare([]byte(candidatePassword), []byte(password)) == 1
		return accounts.Identity{ID: username, Username: username, Role: accounts.RoleAdministrator}, goodUser && goodPass
	}, nil)
}

func newSessionAuthWith(username, password string, authenticate func(string, string) (accounts.Identity, bool), resolveIdentity func(accounts.Identity) (accounts.Identity, bool)) *sessionAuth {
	return &sessionAuth{username: username, password: password, authenticate: authenticate, resolveIdentity: resolveIdentity, now: time.Now, sessions: make(map[[32]byte]webSession), loginTokens: make(map[[32]byte]time.Time), failures: make(map[string]loginAttempts), credentials: make(map[[32]byte]basicCredential)}
}

func (a *sessionAuth) wrap(next http.Handler, bypass func(*http.Request) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setAuthSecurityHeaders(w)
		if bypass(r) {
			next.ServeHTTP(w, r)
			return
		}
		identity, sessionOK := a.validSession(r)
		switch {
		// A server with no account yet serves the first-run page and nothing else: an open library
		// reachable from the network is not a safe default.
		case a.setupRequired():
			if r.URL.Path == setupPath {
				a.serveSetup(w, r)
				return
			}
			http.Redirect(w, r, setupPath, http.StatusSeeOther)
			return
		case r.URL.Path == setupPath:
			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		case r.URL.Path == "/login" && r.Method == http.MethodGet:
			a.loginPage(w, r, http.StatusOK, "")
		case r.URL.Path == "/login" && r.Method == http.MethodPost:
			a.login(w, r)
		case r.URL.Path == "/logout" && r.Method == http.MethodPost:
			a.logout(w, r)
		case sessionOK:
			next.ServeHTTP(w, r.WithContext(contextWithIdentity(r.Context(), identity)))
		case isBasicClientPath(r.URL.Path):
			a.requireBasic(w, r, next)
		default:
			http.Redirect(w, r, "/login?next="+url.QueryEscape(safeNext(r.URL.RequestURI())), http.StatusSeeOther)
		}
	})
}

type requestIdentityContextKey struct{}

func contextWithIdentity(ctx context.Context, identity accounts.Identity) context.Context {
	return context.WithValue(ctx, requestIdentityContextKey{}, identity)
}

func identityFromRequest(r *http.Request) (accounts.Identity, bool) {
	if r == nil {
		return accounts.Identity{}, false
	}
	identity, ok := r.Context().Value(requestIdentityContextKey{}).(accounts.Identity)
	return identity, ok && identity.ID != ""
}

func (a *sessionAuth) loginPage(w http.ResponseWriter, r *http.Request, status int, message string) {
	if _, ok := a.validSession(r); ok {
		http.Redirect(w, r, safeNext(r.URL.Query().Get("next")), http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; font-src 'self'; img-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'")
	csrf, ok := randomToken(24)
	if !ok {
		http.Error(w, "Could not prepare a secure sign-in", http.StatusServiceUnavailable)
		return
	}
	a.rememberLoginToken(csrf)
	w.WriteHeader(status)
	_ = loginPageTemplate.Execute(w, map[string]string{"Next": safeNext(r.URL.Query().Get("next")), "Message": message, "CSRF": csrf})
}

func (a *sessionAuth) login(w http.ResponseWriter, r *http.Request) {
	limitRequestBody(w, r, 8<<10, smallBodyBudget)
	if err := r.ParseForm(); err != nil {
		a.loginPage(w, r, http.StatusBadRequest, "Could not read that sign-in request.")
		return
	}
	if !a.consumeLoginToken(r.Form.Get("csrf")) {
		a.loginPage(w, r, http.StatusBadRequest, "That sign-in page expired. Please try again.")
		return
	}
	next := safeNext(r.Form.Get("next"))
	remote := requestAddress(r)
	if a.rateLimited(remote) {
		a.loginPage(w, r, http.StatusTooManyRequests, "Too many attempts. Wait a few minutes and try again.")
		return
	}
	// A password checked before an account mutation landed is not evidence of
	// anything after it: a reset whose bcrypt ran while this sign-in was in
	// flight must not end with a fresh session for the old password.
	generation := a.credentialGeneration.Load()
	identity, valid := a.authenticateCredentials(r.Form.Get("username"), r.Form.Get("password"))
	if !valid {
		a.recordFailure(remote)
		a.loginPage(w, r, http.StatusUnauthorized, "That username or password is not correct.")
		return
	}
	a.transitionMu.Lock()
	defer a.transitionMu.Unlock()
	if a.credentialGeneration.Load() != generation {
		a.loginPage(w, r, http.StatusUnauthorized, "Those credentials changed while you were signing in. Please try again.")
		return
	}
	a.mu.Lock()
	pending := len(a.pendingRevocations) != 0
	a.mu.Unlock()
	if pending {
		if err := a.persist(); err != nil {
			logoutFailurePage(w)
			return
		}
	}
	a.clearFailure(remote)
	remember := r.Form.Get("remember") == "1"
	token, ok := a.createSession(identity, remember)
	if !ok {
		http.Error(w, "Could not start a secure session", http.StatusServiceUnavailable)
		return
	}
	maxAge := sessionLifetime
	if remember {
		maxAge = rememberLifetime
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true, Secure: requestIsHTTPS(r), SameSite: http.SameSiteStrictMode, MaxAge: int(maxAge.Seconds())})
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (a *sessionAuth) validSession(r *http.Request) (accounts.Identity, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || len(cookie.Value) < 40 || len(cookie.Value) > 64 {
		return accounts.Identity{}, false
	}
	key, now := sessionKey(cookie.Value), a.now()
	a.mu.Lock()
	session, ok := a.sessions[key]
	if !ok || session.expired(now) {
		delete(a.sessions, key)
		a.mu.Unlock()
		return accounts.Identity{}, false
	}
	session.seen = now
	a.sessions[key] = session
	stale := session.remember && a.store != nil && now.Sub(a.lastSaved) > time.Hour
	a.mu.Unlock()
	if stale {
		a.persist() // the idle limit counts from last use, so an hourly refresh is enough
	}
	identity, active := a.resolve(session.identity)
	if !active {
		a.mu.Lock()
		delete(a.sessions, key)
		a.mu.Unlock()
		return accounts.Identity{}, false
	}
	return identity, true
}

func (a *sessionAuth) requireBasic(w http.ResponseWriter, r *http.Request, next http.Handler) {
	remote := requestAddress(r)
	if a.rateLimited(remote) {
		w.Header().Set("Retry-After", "900")
		http.Error(w, "Too many attempts", http.StatusTooManyRequests)
		return
	}
	u, p, ok := r.BasicAuth()
	if !ok {
		// A client's first request carries no credentials: it is the challenge, not a wrong guess, so it must not count
		// towards the lock-out (everyone behind the same proxy shares it) or cost a password check.
		w.Header().Set("WWW-Authenticate", `Basic realm="Mochil OPDS", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	identity, valid := a.basicIdentity(u, p)
	if !valid {
		a.recordFailure(remote)
		w.Header().Set("WWW-Authenticate", `Basic realm="Mochil OPDS", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	a.clearFailure(remote)
	next.ServeHTTP(w, r.WithContext(contextWithIdentity(r.Context(), identity)))
}

func (a *sessionAuth) clearFailure(address string) {
	a.mu.Lock()
	delete(a.failures, address)
	a.mu.Unlock()
}

func (a *sessionAuth) rememberLoginToken(token string) {
	now := a.now()
	a.mu.Lock()
	a.pruneLocked(now)
	if len(a.loginTokens) >= maxLoginTokens {
		var oldestKey [32]byte
		var oldest time.Time
		for key, created := range a.loginTokens {
			if oldest.IsZero() || created.Before(oldest) {
				oldestKey, oldest = key, created
			}
		}
		delete(a.loginTokens, oldestKey)
	}
	a.loginTokens[sessionKey(token)] = now
	a.mu.Unlock()
}

func (a *sessionAuth) consumeLoginToken(token string) bool {
	if len(token) < 24 || len(token) > 64 {
		return false
	}
	key, now := sessionKey(token), a.now()
	a.mu.Lock()
	defer a.mu.Unlock()
	created, ok := a.loginTokens[key]
	delete(a.loginTokens, key)
	return ok && now.Sub(created) <= loginTokenLifetime
}

func (a *sessionAuth) pruneLocked(now time.Time) {
	for key, session := range a.sessions {
		if session.expired(now) {
			delete(a.sessions, key)
		}
	}
	for address, attempt := range a.failures {
		if now.Sub(attempt.started) >= 15*time.Minute {
			delete(a.failures, address)
		}
	}
	for key, created := range a.loginTokens {
		if now.Sub(created) > loginTokenLifetime {
			delete(a.loginTokens, key)
		}
	}
	for key, cached := range a.credentials {
		if !now.Before(cached.expires) {
			delete(a.credentials, key)
		}
	}
}

func sessionKey(token string) [32]byte { return sha256.Sum256([]byte(token)) }

func randomToken(bytes int) (string, bool) {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", false
	}
	return base64.RawURLEncoding.EncodeToString(raw), true
}

func isBasicClientPath(path string) bool {
	return path == "/api/android/v1/login" || path == "/opds" || strings.HasPrefix(path, "/opds/") || strings.HasPrefix(path, "/books/") || strings.HasPrefix(path, "/stream/") || strings.HasPrefix(path, "/covers/") || strings.HasPrefix(path, "/documents/")
}

func safeNext(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || u.Path == "/login" || u.Path == "/logout" {
		return "/"
	}
	return u.RequestURI()
}

func requestAddress(r *http.Request) string {
	// Forwarding headers are attacker-controlled unless the direct peer is a
	// configured trusted proxy. Rate-limit the actual TCP peer; when Mochil
	// is behind Caddy this deliberately creates one conservative proxy bucket.
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]), "https")
}

func setAuthSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(self), geolocation=(), payment=(), usb=(), browsing-topics=()")
}
