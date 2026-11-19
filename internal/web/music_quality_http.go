package web

import (
	"net/http"
	"strings"

	"mochil/internal/audio"
)

func (m *multiServer) apiMusicQuality(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(strings.TrimSpace(r.URL.Query().Get("library")))
	if !ok || library.Kind != "music" || library.Audio == nil {
		http.Error(w, "Music library not found", http.StatusNotFound)
		return
	}
	report := audio.InspectQuality(library.Audio.Items(), func(item audio.Item) bool {
		file, _, found := library.Audio.OpenSidecar(item.ID)
		if found {
			file.Close()
		}
		return found
	})
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, report)
}
