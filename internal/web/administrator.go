package web

import (
	"net/http"
	"strings"
)

// administratorOnly identifies global control-plane operations. Personal
// reading/playback/wiki state deliberately remains outside this policy.
func administratorOnly(path, method string) bool {
	if path == "/api/settings" || path == "/api/folders" || path == "/api/library-inspect" || path == "/api/library-scan" || path == "/api/metadata/provider-connection" || path == "/api/video-preferences" || path == "/api/source-connection" {
		return true
	}
	// The website's /api/diagnostics is the administrator's read-only overview, while
	// the Android route of the same shared name is a device posting its own telemetry.
	// Only the read side is administrative.
	if path == "/api/diagnostics" {
		return method != http.MethodPost
	}
	if path == "/api/narration/default-voice" || path == "/api/narration/transcribe" {
		return true
	}
	if path == "/api/narration/voices" {
		return method == http.MethodPost // making a voice from a recording adds to the shared voice list
	}
	if strings.HasPrefix(path, "/api/vaults/") {
		return method != http.MethodGet && method != http.MethodHead // the vault folder is shared: everyone reads, administrators write
	}
	if strings.HasPrefix(path, "/api/narration/voices/") {
		return method == http.MethodPatch || method == http.MethodDelete
	}
	// Hermes can act on this server, so only administrators may chat with it.
	if path == "/api/avatar" || strings.HasPrefix(path, "/api/avatar/") {
		return true
	}
	if path == "/hermes" || path == "/api/hermes" || strings.HasPrefix(path, "/api/hermes/") {
		return true
	}
	if path == "/api/accounts" || strings.HasPrefix(path, "/api/accounts/") {
		return true
	}
	if path == "/api/restart" {
		return method == http.MethodPost
	}
	if path == "/api/audio-analysis" {
		return method != http.MethodGet && method != http.MethodHead
	}
	if path == "/api/jobs" {
		return method != http.MethodGet && method != http.MethodHead
	}
	if strings.HasPrefix(path, "/api/metadata/") || strings.HasPrefix(path, "/api/publication-metadata/") || strings.HasPrefix(path, "/api/video-metadata/") || strings.HasPrefix(path, "/api/video-metadata-match/") || strings.HasPrefix(path, "/api/video-show-metadata/") {
		return true
	}
	if path == "/api/podcast-feeds" || strings.HasPrefix(path, "/api/podcast-feeds/") || path == "/api/podcast-downloads" || strings.HasPrefix(path, "/api/podcast-downloads/") {
		return method != http.MethodGet && method != http.MethodHead
	}
	return false
}

func (m *multiServer) authorizeAdministrator(w http.ResponseWriter, r *http.Request) bool {
	return m.authorizeAdministratorFor(w, r, r.URL.Path)
}

// authorizeAdministratorFor applies the policy to the website path a request stands for. The Android routes reach the same
// handlers under another prefix, so they ask with the shared path.
func (m *multiServer) authorizeAdministratorFor(w http.ResponseWriter, r *http.Request, path string) bool {
	if !administratorOnly(path, r.Method) || m.accounts == nil {
		return true
	}
	identity, ok := identityFromRequest(r)
	if ok && identity.Administrator() {
		return true
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "Administrator access is required"})
	return false
}

// mayEditVaults reports whether this request may change the shared vault folders; it follows the administrator policy.
func (m *multiServer) mayEditVaults(r *http.Request) bool { return m.isAdministrator(r) }

// isAdministrator is true for an administrator, and always when the server has no accounts (a single owner).
func (m *multiServer) isAdministrator(r *http.Request) bool {
	if m.accounts == nil {
		return true
	}
	identity, ok := identityFromRequest(r)
	return ok && identity.Administrator()
}
