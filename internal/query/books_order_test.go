package query

import (
	"testing"

	"mochil/internal/catalog"
)

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
	order := func(items []BookResult) string {
		out := ""
		for _, item := range items {
			out += item.Book.ID
		}
		return out
	}
	byAuthor := books()
	sortBooksByName(byAuthor, "author")
	if got := order(byAuthor); got != "cbead" {
		t.Fatalf("author order = %s, want cbead (Able's series in order, Baker, Zane, no author last)", got)
	}
	bySeries := books()
	sortBooksByName(bySeries, "series")
	if got := order(bySeries); got != "cbeda" {
		t.Fatalf("series order = %s, want cbeda (series in volume order, then the rest by title)", got)
	}
}
