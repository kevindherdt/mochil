package web

import (
	"net/http"
	"strings"
)

const maximumLibraryScanRequestBytes = 4 << 10

func (m *multiServer) apiLibraryScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.scanLibrary == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Manual library scanning is unavailable"})
		return
	}
	var input struct {
		LibraryID string `json:"library_id"`
		All       bool   `json:"all"`
	}
	if err := decodeJSONRequest(w, r, maximumLibraryScanRequestBytes, &input, true); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid scan request"})
		return
	}
	input.LibraryID = strings.TrimSpace(input.LibraryID)
	if input.All {
		keys := make([]string, 0, len(m.libraries))
		for _, library := range m.libraries {
			key, err := m.scanLibrary(library.ID)
			if err != nil {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			keys = append(keys, key)
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"job_keys": keys, "status": "queued"})
		return
	}
	if input.LibraryID == "" || len(input.LibraryID) > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Choose a valid library"})
		return
	}
	key, err := m.scanLibrary(input.LibraryID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"job_key": key, "status": "queued"})
}
