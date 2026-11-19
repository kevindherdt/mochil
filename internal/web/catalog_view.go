package web

import (
	"time"

	"mochil/internal/catalog"
)

// publishedBooks returns a catalogue for a read-only feed. A backend that keeps
// an immutable published snapshot hands it over without the deep copy Books
// makes, which is what every browse feed needs: the feeds only read these
// values, and the store swaps whole slices instead of editing one in place.
func publishedBooks(backend catalog.Backend) []catalog.Book {
	if snapshot, ok := backend.(interface{ Snapshot() []catalog.Book }); ok {
		return snapshot.Snapshot()
	}
	return backend.Books()
}

// catalogueSummary answers "how many, and how new" without materialising the
// catalogue. The OPDS root feed needs nothing else from 50k books.
func catalogueSummary(backend catalog.Backend) (int, time.Time) {
	if summary, ok := backend.(interface {
		Summary() (int, time.Time)
	}); ok {
		return summary.Summary()
	}
	books := backend.Books()
	return len(books), latestBookTime(books)
}

func latestBookTime(books []catalog.Book) time.Time {
	var latest time.Time
	for _, book := range books {
		if book.Modified.After(latest) {
			latest = book.Modified
		}
	}
	return latest
}
