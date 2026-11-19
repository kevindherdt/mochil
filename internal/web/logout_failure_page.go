package web

import "net/http"

func (a *sessionAuth) logout(w http.ResponseWriter, r *http.Request) {
	a.transitionMu.Lock()
	defer a.transitionMu.Unlock()
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		key := sessionKey(cookie.Value)
		a.mu.Lock()
		_, known := a.sessions[key]
		_, persisted := a.persistedSessions[key]
		_, pending := a.pendingRevocations[key]
		delete(a.sessions, key)
		if known || persisted {
			if a.pendingRevocations == nil {
				a.pendingRevocations = make(map[[32]byte]struct{})
			}
			a.pendingRevocations[key] = struct{}{}
		}
		a.mu.Unlock()
		if known || persisted || pending {
			if err := a.persist(); err != nil {
				logoutFailurePage(w)
				return
			}
			a.mu.Lock()
			delete(a.pendingRevocations, key)
			a.mu.Unlock()
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Path: "/", HttpOnly: true, Secure: requestIsHTTPS(r), SameSite: http.SameSiteStrictMode, MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func logoutFailurePage(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Sign out incomplete</title></head><body><h1>Sign out incomplete</h1><p>Could not complete sign out because the session store is unavailable. This session is blocked until the server restarts, but may become valid again afterward. Contact your administrator, then retry after the storage problem is fixed.</p><form method="post" action="/logout"><button type="submit">Retry sign out</button></form></body></html>`))
}
