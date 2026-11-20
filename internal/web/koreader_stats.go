package web

import (
	"math"
	"net/http"
	"time"

	"mochil/internal/reading"
)

// koReaderStatsRow is one book in the device's statistics panel.
type koReaderStatsRow struct {
	LibraryID  string   `json:"library"`
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Authors    []string `json:"authors,omitempty"`
	Fraction   float64  `json:"fraction"`
	FinishedAt int64    `json:"finished_at,omitempty"`
}

// koReaderStatsReply is what the reader's statistics panel is built from.
type koReaderStatsReply struct {
	InProgress        int                `json:"in_progress"`
	Finished          int                `json:"finished"`
	FinishedThisMonth int                `json:"finished_this_month"`
	PagesRead         int                `json:"pages_read"`
	RecentFinished    []koReaderStatsRow `json:"recent_finished,omitempty"`
}

// serveKoReaderStats answers what a device cannot work out by itself. KOReader's own statistics database only
// knows what was read on that one reader, and this server caps a book's progress history, so books finished on the
// phone and reading spread over several devices leave no local trace. Everything here is the server's reading
// state, so it is the same on every device.
func (m *multiServer) serveKoReaderStats(w http.ResponseWriter, r *http.Request, account string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.reading == nil {
		http.Error(w, "Reading statistics are not available", http.StatusServiceUnavailable)
		return
	}
	rows := func(positions []reading.Position, finished bool) ([]koReaderStatsRow, int) {
		out := make([]koReaderStatsRow, 0, len(positions))
		pages := 0
		for _, position := range positions {
			view, ok := m.readingView(position)
			if !ok {
				continue
			}
			if counted := m.koReaderBookPages(view); counted > 0 {
				if finished {
					pages += counted
				} else if view.Fraction > 0 && view.Fraction < 1 {
					pages += int(math.Round(view.Fraction * float64(counted)))
				}
			}
			out = append(out, koReaderStatsRow{
				LibraryID: view.LibraryID, ID: view.Book.ID, Title: view.Title, Authors: view.Authors,
				Fraction: view.Fraction, FinishedAt: view.FinishedAt,
			})
		}
		return out, pages
	}
	// The count uses the same rule as the device's continue shelf and the web one, so this panel
	// cannot call a book "in progress" that neither shelf lists: a book only opened on page one
	// was never read, and one parked as on hold or dropped is not being read either. Pages read
	// still counts every unfinished book, parked or not, because those pages were read.
	_, readPages := rows(m.reading.ListLibraryLimit(account, false, "", 0), false)
	inProgress := m.reading.ListContinue(account, "", 0)
	finished, finishedPages := rows(m.reading.ListLibraryLimit(account, true, "", 0), true)
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	thisMonth := 0
	for _, row := range finished {
		if row.FinishedAt >= monthStart {
			thisMonth++
		}
	}
	const mostRecent = 5
	recent := finished
	if len(recent) > mostRecent {
		recent = recent[:mostRecent]
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, koReaderStatsReply{
		InProgress:        len(inProgress),
		Finished:          len(finished),
		FinishedThisMonth: thisMonth,
		PagesRead:         readPages + finishedPages,
		RecentFinished:    recent,
	})
}

// koReaderBookPages is how many pages a book has, for turning a fraction into pages read. A comic knows its own
// page count; a text book needs the count the editor's word/page job already made, and answers 0 until it exists.
func (m *multiServer) koReaderBookPages(view readingView) int {
	if view.Book.PageCount > 0 {
		return view.Book.PageCount
	}
	if m.bookStats != nil {
		if stats, ok := m.bookStats.Get(view.LibraryID, view.Book.ID, view.Book.Size, view.Book.Modified); ok && stats.Pages > 0 {
			return stats.Pages
		}
	}
	return 0
}
