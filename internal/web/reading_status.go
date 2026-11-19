package web

import (
	"errors"
	"net/http"

	"mochil/internal/catalog"
	"mochil/internal/reading"
)

// apiReadingStatus sets a reader's own label for one book: unread, reading, read, on_hold or dropped.
func (m *multiServer) apiReadingStatus(w http.ResponseWriter, r *http.Request, lib Library, book catalog.Book, account string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if decodeJSONRequest(w, r, 1024, &request, false) != nil {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}
	position, err := m.reading.SetStatus(account, lib.ID, book.ID, request.Status)
	if errors.Is(err, reading.ErrUnknownStatus) {
		http.Error(w, "Unknown status", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Reading state could not be saved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, position)
}

// heldReadingViews lists the books this reader put on hold or dropped, for the Home shelf.
func (m *multiServer) heldReadingViews(account string, limit int) []readingView {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	positions := m.reading.ListHeld(account, limit)
	out := make([]readingView, 0, len(positions))
	for _, position := range positions {
		if view, ok := m.readingView(position); ok {
			out = append(out, view)
		}
	}
	return out
}
