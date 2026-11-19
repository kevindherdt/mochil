package web

import (
	"net/http"
	"net/url"
	"strings"
)

// serveKoReaderFinished lets a paired device mark a catalog book finished/unfinished before ever
// downloading or opening it: PUT /api/koreader/v1/finished/{library}/{id}. Unlike push's progress route,
// this is keyed by library/book id directly (not a file digest), since the device may not have the file
// yet -- it reuses the same reading.Store.SetFinished the browser-facing dashboard already calls.
func (m *multiServer) serveKoReaderFinished(w http.ResponseWriter, r *http.Request, account string) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", "PUT")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/koreader/v1/finished/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, err1 := url.PathUnescape(parts[0])
	bookID, err2 := url.PathUnescape(parts[1])
	library, found := m.library(libraryID)
	if err1 != nil || err2 != nil || !found || library.Store == nil {
		http.NotFound(w, r)
		return
	}
	if _, found := library.Store.Book(bookID); !found {
		http.NotFound(w, r)
		return
	}
	var request struct {
		Finished bool `json:"finished"`
	}
	if decodeJSONRequest(w, r, 1024, &request, false) != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	position, err := m.reading.SetFinished(account, libraryID, bookID, request.Finished)
	if err != nil {
		http.Error(w, "Reading state could not be saved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, position)
}
