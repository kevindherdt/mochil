package web

import (
	"net/http"

	"mochil/internal/reading"
)

// serveKoReaderContinue pages the account's current unfinished reading list,
// not the revision history of individual books. Resolve entries before slicing
// so removed books cannot create holes or inflate totals.
func (m *multiServer) serveKoReaderContinue(w http.ResponseWriter, r *http.Request, account string) {
	offset, limit := koReaderPage(r)
	library := r.URL.Query().Get("library")
	// ?held=true is the shelf the web and the Android app already have: the books the reader
	// parked as on hold or dropped, which the continue list deliberately leaves out.
	var positions []reading.Position
	if r.URL.Query().Get("held") == "true" {
		for _, position := range m.reading.ListHeld(account, 0) {
			if library == "" || position.LibraryID == library {
				positions = append(positions, position)
			}
		}
	} else {
		positions = m.reading.ListContinue(account, library, 0)
	}
	entries := make([]koReaderHomeBook, 0, min(limit, len(positions)))
	total := 0
	for _, position := range positions {
		view, ok := m.readingView(position)
		if !ok {
			continue
		}
		if total >= offset && len(entries) < limit {
			entry := koReaderHomeBook{Title: view.Title, Authors: view.Authors, Fraction: view.Fraction}
			switch view.LibraryKind {
			case "books", "comics", "manga", "mixed":
				book := m.koReaderBookOf(account, view.LibraryID, view.Book)
				entry.Book = &book
			}
			entries = append(entries, entry)
		}
		total++
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, struct {
		Total    int                `json:"total"`
		Continue []koReaderHomeBook `json:"continue"`
	}{Total: total, Continue: entries})
}
