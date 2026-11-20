package web

import (
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"mochil/internal/catalog"
)

// The device could see a series name and a volume number but had no way to ask what came next, so
// continuing a series meant hunting the title again in the library list. The catalogue already
// knows the order, so the dialog can offer the next volume in one tap.
type koReaderSeriesPage struct {
	Series   string        `json:"series"`
	Index    *float64      `json:"index,omitempty"`
	Previous *koReaderBook `json:"previous,omitempty"`
	Next     *koReaderBook `json:"next,omitempty"`
}

// serveKoReaderSeries answers /api/koreader/v1/series/{library}/{id} with the neighbouring volumes
// of that book's series. A book outside a series is a valid request with an empty "series" field,
// not an error: the plugin asks only for books that carry a series, and a stale catalogue entry
// should not produce a scary failure in the dialog.
func (m *multiServer) serveKoReaderSeries(w http.ResponseWriter, r *http.Request, account string) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/koreader/v1/series/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, err := url.PathUnescape(parts[0])
	bookID, err2 := url.PathUnescape(parts[1])
	library, found := m.library(libraryID)
	if err != nil || err2 != nil || !found || library.Store == nil {
		http.NotFound(w, r)
		return
	}
	book, found := library.Store.Book(bookID)
	if !found {
		http.NotFound(w, r)
		return
	}
	page := koReaderSeriesPage{Series: book.Series, Index: book.SeriesIndex}
	previous, next, ok := seriesNeighbours(library.Store.Books(), book)
	if !ok {
		writeJSON(w, http.StatusOK, page)
		return
	}
	// A nil pointer is how "no such volume" travels; an empty book object would render as a row.
	if previous.ID != "" {
		neighbour := m.koReaderBookOf(account, libraryID, previous)
		page.Previous = &neighbour
	}
	if next.ID != "" {
		neighbour := m.koReaderBookOf(account, libraryID, next)
		page.Next = &neighbour
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, page)
}

// seriesNeighbours finds the volume before and after this one. It returns false when the two are
// not orderable - no series, no index, or nothing else in the series - because a neighbour guessed
// from an unordered set is worse than none.
func seriesNeighbours(books []catalog.Book, book catalog.Book) (previous, next catalog.Book, ok bool) {
	if book.Series == "" || book.SeriesIndex == nil {
		return catalog.Book{}, catalog.Book{}, false
	}
	// A NaN index compares false against everything, so ordering would silently stop being total and
	// this book would have no neighbours at all. Treat it as unordered instead of guessing.
	if math.IsNaN(*book.SeriesIndex) {
		return catalog.Book{}, catalog.Book{}, false
	}
	var siblings []catalog.Book
	for _, other := range books {
		if other.ID != book.ID && other.Series == book.Series && other.SeriesIndex != nil {
			siblings = append(siblings, other)
		}
	}
	// Unknown volumes sort last: a book without an index must never be promoted above a numbered one.
	sort.SliceStable(siblings, func(i, j int) bool {
		if *siblings[i].SeriesIndex != *siblings[j].SeriesIndex {
			return *siblings[i].SeriesIndex < *siblings[j].SeriesIndex
		}
		return siblings[i].Title < siblings[j].Title
	})
	mine := *book.SeriesIndex
	for _, sibling := range siblings {
		if *sibling.SeriesIndex < mine {
			previous = sibling
		} else if *sibling.SeriesIndex > mine {
			next = sibling
			break
		}
	}
	return previous, next, previous.ID != "" || next.ID != ""
}
