package web

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	"mochil/internal/externallibrary"
)

const maximumExternalLibraryRequestBytes = 32 << 10

func (m *multiServer) apiExternalLibrary(w http.ResponseWriter, r *http.Request) {
	if m.externalLibrary == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "External library synchronization is unavailable"})
		return
	}
	prefix := "/api/external-library"
	if strings.HasPrefix(r.URL.Path, "/api/android/v1/external-library") {
		prefix = "/api/android/v1/external-library"
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	account := m.readingAccount(r)
	if rest == "" {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"items": savedOPDSTitles(m.externalLibrary.List(account))})
		case http.MethodPost:
			var input externallibrary.Title
			if err := decodeJSONRequest(w, r, maximumExternalLibraryRequestBytes, &input, true); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid external title"})
				return
			}
			if input.ProviderID != "opds" || input.ExecutionLocation != externallibrary.ExecutionDevice {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Only device-read OPDS publications can be saved"})
				return
			}
			title, err := m.externalLibrary.Put(account, input)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, title)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	parts := strings.Split(rest, "/")
	titleID, err := url.PathUnescape(parts[0])
	if err != nil || titleID == "" {
		http.NotFound(w, r)
		return
	}
	title, exists := m.externalLibrary.Get(account, titleID)
	if !exists || title.ProviderID != "opds" || title.ExecutionLocation != externallibrary.ExecutionDevice {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodDelete {
		err := m.externalLibrary.Remove(account, titleID)
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not remove external title"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if len(parts) == 2 && parts[1] == "progress" {
		m.externalLibraryProgress(w, r, account, titleID)
		return
	}
	http.NotFound(w, r)
}

func savedOPDSTitles(titles []externallibrary.Title) []externallibrary.Title {
	result := make([]externallibrary.Title, 0, len(titles))
	for _, title := range titles {
		if title.ProviderID == "opds" && title.ExecutionLocation == externallibrary.ExecutionDevice {
			result = append(result, title)
		}
	}
	return result
}

func (m *multiServer) externalLibraryProgress(w http.ResponseWriter, r *http.Request, account, titleID string) {
	switch r.Method {
	case http.MethodGet:
		chapterID := strings.TrimSpace(r.URL.Query().Get("chapter"))
		if chapterID == "" {
			writeJSON(w, http.StatusOK, map[string]any{"items": m.externalLibrary.ListProgress(account, titleID)})
			return
		}
		value, ok := m.externalLibrary.Progress(account, titleID, chapterID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, value)
	case http.MethodPut:
		var input struct {
			Progress     externallibrary.Progress `json:"progress"`
			BaseRevision int64                    `json:"base_revision"`
		}
		if err := decodeJSONRequest(w, r, maximumExternalLibraryRequestBytes, &input, true); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid external progress"})
			return
		}
		value, err := m.externalLibrary.PutProgress(account, titleID, input.Progress, input.BaseRevision)
		if errors.Is(err, externallibrary.ErrConflict) {
			writeJSON(w, http.StatusConflict, value)
			return
		}
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, value)
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
