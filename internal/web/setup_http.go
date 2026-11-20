package web

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"text/template"

	"mochil/internal/accounts"
)

// firstRunMu guards the one-time handover from "no account yet" to a working account store.
var firstRunMu sync.Mutex

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// createFirstRunAccount is the moment a fresh install stops being open: it opens the store that has
// never existed, creates the administrator and hands it to the server, so the next request is an
// ordinary sign-in.
func (m *multiServer) createFirstRunAccount(username, password string) error {
	firstRunMu.Lock()
	defer firstRunMu.Unlock()
	if m.accounts != nil {
		return accounts.ErrConflict
	}
	store, err := accounts.OpenFirstRun(m.firstRunAccountsPath)
	if err != nil {
		return err
	}
	if _, err := store.Create(username, password, accounts.RoleAdministrator); err != nil {
		return err
	}
	m.accounts = store
	// The replacement handlers of a live reload continue this store rather than
	// re-opening the file the first-run page just created.
	m.stores.adoptAccounts(store)
	return nil
}

// needsFirstAccount reports whether this server is in its first run.
func (m *multiServer) needsFirstAccount() bool {
	firstRunMu.Lock()
	defer firstRunMu.Unlock()
	return m.accounts == nil && m.firstRunAccountsPath != ""
}

// The first-run page is the door of a fresh install. Without it a server with no credentials
// served the whole library to anyone who found the port, which is fine on a laptop and wrong the
// moment it is reachable from anywhere else.
const setupPath = "/setup"

var setupPageTemplate = template.Must(template.New("setup").Parse(`<!doctype html>
<html lang="en" data-theme="dark"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover"><meta name="color-scheme" content="dark light"><title>Create your account · Mochil</title><link rel="stylesheet" href="/assets/design-tokens.css"><link rel="stylesheet" href="/assets/login.css?v=4"></head>
<body><main><section aria-labelledby="setup-title"><a class="wordmark" href="/setup" aria-label="Mochil"><span><img src="/assets/icons/icon-128.png" alt=""></span> Mochil</a><div><h1 id="setup-title">Create your account</h1><p class="intro">This server has no account yet. Choose the name and password you will sign in with; you can add more accounts later in Settings.</p></div><form method="post" action="/setup"><input type="hidden" name="csrf" value="{{.CSRF}}"><label>Username<input name="username" required maxlength="128" autocapitalize="none" autocomplete="username" spellcheck="false"></label><label>Password<input name="password" type="password" required minlength="10" maxlength="72" autocomplete="new-password"></label><label>Repeat the password<input name="confirm" type="password" required minlength="10" maxlength="72" autocomplete="new-password"></label>{{if .Message}}<p class="error" role="alert">{{.Message}}</p>{{end}}<button type="submit">Create account</button></form><small>A password of at least 10 characters. This account is the administrator: it manages libraries and additional accounts.</small></section></main></body></html>`))

// setupRequired reports whether the server has no account yet, so only the first-run page may be
// served. A configured install (credentials in the environment) is never in this state.
func (a *sessionAuth) setupRequired() bool {
	a.authenticateMu.RLock()
	check := a.setup
	a.authenticateMu.RUnlock()
	return check != nil && check()
}

func (a *sessionAuth) setSetup(required func() bool, create func(username, password string) error) {
	a.authenticateMu.Lock()
	a.setup, a.createFirstAccount = required, create
	a.authenticateMu.Unlock()
}

func (a *sessionAuth) serveSetup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		a.setupPage(w, r, http.StatusOK, "")
	case http.MethodPost:
		a.createAccount(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *sessionAuth) setupPage(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data:; form-action 'self'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'")
	csrf, ok := randomToken(24)
	if !ok {
		http.Error(w, "Could not prepare the first-run page", http.StatusServiceUnavailable)
		return
	}
	a.rememberLoginToken(csrf)
	w.WriteHeader(status)
	_ = setupPageTemplate.Execute(w, map[string]string{"Message": message, "CSRF": csrf})
}

func (a *sessionAuth) createAccount(w http.ResponseWriter, r *http.Request) {
	limitRequestBody(w, r, 8<<10, smallBodyBudget)
	if err := r.ParseForm(); err != nil {
		a.setupPage(w, r, http.StatusBadRequest, "Could not read that request.")
		return
	}
	if !a.consumeLoginToken(r.Form.Get("csrf")) {
		a.setupPage(w, r, http.StatusBadRequest, "That page expired. Please try again.")
		return
	}
	// The first-run page creates an administrator, so it deserves the same lock-out as the sign-in
	// form: an unauthenticated visitor must not get unlimited guesses at a page that mints accounts.
	remote := requestAddress(r)
	if a.rateLimited(remote) {
		a.setupPage(w, r, http.StatusTooManyRequests, "Too many attempts. Wait a few minutes and try again.")
		return
	}
	username, password, confirm := r.Form.Get("username"), r.Form.Get("password"), r.Form.Get("confirm")
	if password != confirm {
		a.recordFailure(remote)
		a.setupPage(w, r, http.StatusBadRequest, "The two passwords do not match.")
		return
	}
	a.authenticateMu.RLock()
	create := a.createFirstAccount
	a.authenticateMu.RUnlock()
	if create == nil {
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
		return
	}
	if err := create(username, password); err != nil {
		a.recordFailure(remote)
		a.setupPage(w, r, http.StatusBadRequest, setupFailureMessage(err))
		return
	}
	a.clearFailure(remote)
	http.Redirect(w, r, loginPath, http.StatusSeeOther)
}

// setupFailureMessage says what to change, because the alternative is a field that silently
// refuses and leaves the reader guessing at the password policy.
func setupFailureMessage(err error) string {
	switch {
	case errors.Is(err, accounts.ErrConflict):
		return "That username is already taken."
	case errors.Is(err, accounts.ErrInvalid):
		return fmt.Sprintf("Choose a username and a password of at least 10 characters (at most %d).", 72)
	default:
		return "The account could not be saved: " + err.Error()
	}
}
