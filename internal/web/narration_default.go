package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// The default narration voice is a server-wide choice made in Settings, Narration. It is kept in its own small state
// file (not the main config) so changing it never reloads the application. The Hermes chat and the book reader both
// start from it.
type narrationDefault struct {
	Voice string `json:"voice"`
}

func (m *multiServer) narrationDefaultVoice() string {
	if path := m.cfg.NarrationDefaultVoiceState; path != "" {
		if data, err := os.ReadFile(path); err == nil {
			var saved narrationDefault
			if json.Unmarshal(data, &saved) == nil && saved.Voice != "" {
				if _, _, ok := m.narrationVoiceRef(saved.Voice); ok {
					return saved.Voice
				}
			}
		}
	}
	return defaultNarrationVoice
}

// narrationSetDefault saves the voice chosen in Settings. Administrators only (see administratorOnly).
func (m *multiServer) narrationSetDefault(w http.ResponseWriter, r *http.Request) {
	if m.cfg.NarrationDefaultVoiceState == "" {
		http.Error(w, "The default voice cannot be saved on this server.", http.StatusNotImplemented)
		return
	}
	var request narrationDefault
	if err := decodeJSONRequest(w, r, 1<<10, &request, true); err != nil || strings.TrimSpace(request.Voice) == "" {
		http.Error(w, "Choose a voice.", http.StatusBadRequest)
		return
	}
	if _, _, ok := m.narrationVoiceRef(request.Voice); !ok {
		http.Error(w, "Unknown voice.", http.StatusBadRequest)
		return
	}
	data, _ := json.Marshal(narrationDefault{Voice: request.Voice})
	if err := os.MkdirAll(filepath.Dir(m.cfg.NarrationDefaultVoiceState), 0o755); err != nil {
		http.Error(w, "Could not save the voice.", http.StatusInternalServerError)
		return
	}
	temporary := m.cfg.NarrationDefaultVoiceState + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil || os.Rename(temporary, m.cfg.NarrationDefaultVoiceState) != nil {
		http.Error(w, "Could not save the voice.", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, narrationDefault{Voice: request.Voice})
}
