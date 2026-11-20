package query

import (
	"fmt"
	"runtime"
	"testing"

	"mochil/internal/catalog"
)

// A page of "all libraries" is sorted per request. It used to copy every match
// into a BookResult first — 424 bytes each, and append growth doubled that — so
// one click on All libraries over a large shelf allocated tens of megabytes to
// return a few hundred books. The page may cost its own size plus one key per
// book, never a copy of every book.
func TestBooksPageDoesNotCopyTheWholeShelf(t *testing.T) {
	backends := []*testBackend{{}, {}}
	libraries := []Library{
		{ID: "a", Name: "First", Store: backends[0]},
		{ID: "b", Name: "Second", Store: backends[1]},
	}
	for libraryIndex, backend := range backends {
		books := make([]catalog.Book, 10_000)
		for index := range books {
			books[index] = catalog.Book{ID: fmt.Sprintf("%08d", index), Title: fmt.Sprintf("The Book Volume %08d", libraryIndex*10_000+index)}
		}
		backend.replace(books)
	}
	service := New(libraries)
	if got := service.Books(BooksRequest{Limit: 120}); got.Total != 20_000 {
		t.Fatalf("warm-up saw %d books, want 20000", got.Total)
	}

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	result := service.Books(BooksRequest{Limit: 120})
	runtime.ReadMemStats(&after)

	if result.Total != 20_000 || len(result.Items) != 120 {
		t.Fatalf("page = %d of %d books, want 120 of 20000", len(result.Items), result.Total)
	}
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("a page of %d from %d books allocated %.1f MB", len(result.Items), result.Total, float64(allocated)/(1<<20))
	if allocated > 12<<20 {
		t.Fatalf("paging one shelf of 20000 books allocated %.1f MB; a page of 120 must not copy the whole shelf", float64(allocated)/(1<<20))
	}
}
