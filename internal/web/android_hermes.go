package web

import "net/http"

// The Android app reaches the Hermes chat and the avatar beside it under /api/android/v1/, which are the same handlers
// as /api/hermes/ and /api/avatar/. The administrator rule is checked on the original path before the device is known,
// so a forwarded request has to be checked here: Hermes can act on this server, so a paired device of an ordinary
// account is refused.
func (m *multiServer) androidAdministratorForward(w http.ResponseWriter, r *http.Request, from, to string, handler http.HandlerFunc) {
	if m.accountStore() != nil {
		identity, ok := identityFromRequest(r)
		if !ok || !identity.Administrator() {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Administrator access is required"})
			return
		}
	}
	forwardAndroidPath(r, from, to, handler, w)
}

func (m *multiServer) androidHermes(w http.ResponseWriter, r *http.Request) {
	m.androidAdministratorForward(w, r, "/api/android/v1/hermes", "/api/hermes", m.apiHermes)
}

func (m *multiServer) androidAvatar(w http.ResponseWriter, r *http.Request) {
	m.androidAdministratorForward(w, r, "/api/android/v1/avatar", "/api/avatar", m.apiAvatar)
}
