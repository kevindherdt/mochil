package query

import (
	"testing"

	"mochil/internal/catalog"
)

// nameFixture wraps results the way the shelf does, so the sort is exercised on
// references to published books rather than on a private copy of each one.
func nameFixture(items []BookResult) []matchRef {
	matched := make([]matchRef, len(items))
	for index, item := range items {
		matched[index] = matchRef{
			library: &libraryIndex{library: Library{ID: item.LibraryID, Name: item.LibraryName}},
			indexed: &indexedBook{book: item.Book},
		}
	}
	return matched
}

func fixtureIDs(matched []matchRef) string {
	out := ""
	for _, item := range matched {
		out += item.indexed.book.ID
	}
	return out
}

func TestSortBooksByAuthorAndSeries(t *testing.T) {
	one, two := 1.0, 2.0
	books := func() []BookResult {
		return []BookResult{
			{LibraryID: "l", Book: catalog.Book{ID: "a", Title: "Zed", Authors: []string{"Zoe Zane"}}},
			{LibraryID: "l", Book: catalog.Book{ID: "b", Title: "Two", Authors: []string{"Ann Able"}, Series: "Saga", SeriesIndex: &two}},
			{LibraryID: "l", Book: catalog.Book{ID: "c", Title: "One", Authors: []string{"Ann Able"}, Series: "Saga", SeriesIndex: &one}},
			{LibraryID: "l", Book: catalog.Book{ID: "d", Title: "Nobody"}},
			{LibraryID: "l", Book: catalog.Book{ID: "e", Title: "Alone", Authors: []string{"Bob Baker"}}},
		}
	}
	byAuthor := nameFixture(books())
	sortBooksByName(byAuthor, "author")
	if got := fixtureIDs(byAuthor); got != "cbead" {
		t.Fatalf("author order = %s, want cbead (Able's series in order, Baker, Zane, no author last)", got)
	}
	bySeries := nameFixture(books())
	sortBooksByName(bySeries, "series")
	if got := fixtureIDs(bySeries); got != "cbeda" {
		t.Fatalf("series order = %s, want cbeda (series in volume order, then the rest by title)", got)
	}
}
