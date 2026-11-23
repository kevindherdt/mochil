package web

import "net/http"

// apiMe answers who the caller is — their own identity, and nothing else.
//
// This exists because the only endpoint that carried an identity was GET /api/accounts, which is
// administrator-only (administrator.go). An ordinary user asking "who am I" therefore got a 403, and any screen
// built on that answer showed an error instead of the person's own name. That was found by signing in as a
// non-administrator and loading Settings → Accounts: the role check never got to run, because the session read
// itself had failed.
//
// Deliberately narrow: the caller's own id, username and role. It cannot list, read or modify another account, so
// it is safe to leave outside the administrator gate, and it says nothing about the server's configuration.
func (m *multiServer) apiMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	identity, _ := identityFromRequest(r)
	writeJSON(w, http.StatusOK, map[string]string{
		"id":       identity.ID,
		"username": identity.Username,
		"role":     identity.Role,
	})
}
