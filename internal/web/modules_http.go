package web

import (
	"net/http"
	"strings"

	"mochil/internal/appconfig"
)

// moduleOn reports whether an optional module is running. A nil module map
// means everything is on.
func (m *multiServer) moduleOn(id string) bool { return m.cfg.moduleOn(id) }

func (cfg MultiConfig) moduleOn(id string) bool {
	if cfg.Modules == nil {
		return true
	}
	return cfg.Modules[id]
}

// androidRoute reports whether a path belongs to the Android module itself:
// pairing, sign-in, the APK and the app API.
func androidRoute(path string) bool {
	return strings.HasPrefix(path, androidPrefix) || path == "/pair" || path == "/api/android/app.apk"
}

// androidPrefix is where the Android app reaches the same features the website
// reaches under /api/, so both share one module gate.
const androidPrefix = "/api/android/v1/"

func sharedPath(path string) string {
	if rest, ok := strings.CutPrefix(path, androidPrefix); ok {
		return "/api/" + rest
	}
	return path
}

// moduleForPath names the optional module that owns a route, or "" for the core.
func moduleForPath(path string) string {
	path = sharedPath(path)
	switch {
	case strings.HasPrefix(path, "/videos/"), strings.HasPrefix(path, "/api/video"),
		path == "/api/metadata/video-quality", path == "/api/tmdb-portrait":
		return "video"
	case strings.HasPrefix(path, "/audio/"), strings.HasPrefix(path, "/api/audio"),
		strings.HasPrefix(path, "/podcast-media/"), strings.HasPrefix(path, "/api/podcast"),
		path == "/api/metadata/music-quality":
		return "audio"
	case strings.HasPrefix(path, "/api/avatar"):
		return "avatar"
	case strings.HasPrefix(path, "/api/hermes"):
		return "hermes"
	case strings.HasPrefix(path, "/api/narration"):
		return "narration"
	case strings.HasPrefix(path, "/documents/"), strings.HasPrefix(path, "/api/documents"),
		strings.HasPrefix(path, "/api/vaults/"), strings.HasPrefix(path, "/api/wiki"):
		return "documents"
	}
	return ""
}

// moduleOffResponse answers a request for a switched-off module. The Home
// shelves poll the two progress lists, so those return an empty list instead of
// an error; everything else is simply not there.
func (m *multiServer) moduleOffResponse(w http.ResponseWriter, r *http.Request) {
	if shared := sharedPath(r.URL.Path); r.Method == http.MethodGet && (shared == "/api/video-progress" || shared == "/api/audio-progress") {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	http.NotFound(w, r)
}

type moduleView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Kinds       []string `json:"kinds"`
	Enabled     bool     `json:"enabled"`
	DefaultOn   bool     `json:"default_on"`
	// Group and Option let the settings card show one switch per option under
	// the module that owns the group. Both are absent for a standalone card.
	Group  string `json:"group,omitempty"`
	Option bool   `json:"option,omitempty"`
}

func (m *multiServer) apiModules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	out := make([]moduleView, 0, len(appconfig.Modules))
	for _, module := range appconfig.Modules {
		out = append(out, moduleView{ID: module.ID, Name: module.Name, Description: module.Description, Kinds: module.Kinds, Enabled: m.moduleOn(module.ID), DefaultOn: module.DefaultOn, Group: module.Group, Option: module.Option})
	}
	writeJSON(w, http.StatusOK, out)
}
