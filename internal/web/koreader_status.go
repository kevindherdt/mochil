package web

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"mochil/internal/reading"
)

// serveKoReaderStatus lets a device park a book the same way the web UI does. Without it the
// status a reader sets on the web (on hold, dropped, read) is invisible from KOReader, and the
// continue shelf's "not parked" rule can only be driven from a browser.
func (m *multiServer) serveKoReaderStatus(w http.ResponseWriter, r *http.Request, account string) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", "PUT")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/koreader/v1/status/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, err1 := url.PathUnescape(parts[0])
	bookID, err2 := url.PathUnescape(parts[1])
	if err1 != nil || err2 != nil || libraryID == "" || bookID == "" {
		http.NotFound(w, r)
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if decodeJSONRequest(w, r, 1024, &request, false) != nil {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}
	position, err := m.reading.SetStatus(account, libraryID, bookID, request.Status)
	if errors.Is(err, reading.ErrUnknownStatus) {
		http.Error(w, "Unknown status", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Reading state could not be saved", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   position.Status,
		"finished": position.Finished,
	})
}
