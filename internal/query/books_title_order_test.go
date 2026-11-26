package query

import (
	"testing"

	"mochil/internal/catalog"
)

// A shelf that spans libraries must sort titles the same way one library does:
// "Saga Book 2" before "Saga Book 9" before "Saga Book 10". The per-library
// index already sorts naturally, so a byte comparison in the cross-library path
// put volume 10 first and made the same three books change order when the
// reader switched from one library to all of them.
func TestBooksAcrossLibrariesOrderTitlesNaturally(t *testing.T) {
	one := &testBackend{}
	one.replace([]catalog.Book{{ID: "b2", Title: "Saga Book 2"}})
	two := &testBackend{}
	two.replace([]catalog.Book{{ID: "b9", Title: "Saga Book 9"}, {ID: "b10", Title: "Saga Book 10"}})
	service := New([]Library{
		{ID: "one", Name: "One", Store: one},
		{ID: "two", Name: "Two", Store: two},
	})

	var got []string
	for _, item := range service.Books(BooksRequest{Limit: 10}).Items {
		got = append(got, item.LibraryID+"/"+item.Book.ID)
	}
	want := []string{"one/b2", "two/b9", "two/b10"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("all libraries: got %v, want %v", got, want)
		}
	}

	// The single library that holds volumes 9 and 10 already answers naturally;
	// the cross-library list above has to agree with it.
	var single []string
	for _, item := range service.Books(BooksRequest{LibraryID: "two", Limit: 10}).Items {
		single = append(single, item.Book.ID)
	}
	if len(single) != 2 || single[0] != "b9" || single[1] != "b10" {
		t.Fatalf("one library: got %v, want [b9 b10]", single)
	}
}

// A series whose volumes carry no numeric index falls back to their titles, and
// that fallback is read by the e-reader's series view: it has to be the same
// natural order the rest of the catalogue uses, not a byte comparison.
func TestSeriesVolumesFallBackToNaturalTitleOrder(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{
		{ID: "b10", Title: "Saga Book 10", Series: "Saga"},
		{ID: "b2", Title: "Saga Book 2", Series: "Saga"},
		{ID: "b9", Title: "Saga Book 9", Series: "Saga"},
	})
	service := New([]Library{{ID: "one", Name: "One", Store: backend}})
	series := "Saga"
	var got []string
	for _, item := range service.Books(BooksRequest{LibraryID: "one", Series: &series, Sort: "series_volume", Limit: 10}).Items {
		got = append(got, item.Book.ID)
	}
	if len(got) != 3 || got[0] != "b2" || got[1] != "b9" || got[2] != "b10" {
		t.Fatalf("series volumes: got %v, want [b2 b9 b10]", got)
	}
}

// The author shelf is sorted from references held next to each library's index,
// so the page has to be built from the book each reference points at, in the
// library it came from.
func TestBooksAuthorSortAcrossLibraries(t *testing.T) {
	one, two := &testBackend{}, &testBackend{}
	one.replace([]catalog.Book{
		{ID: "ten", Title: "Ten", Authors: []string{"Ada Author 10"}},
		{ID: "beta", Title: "Beta", Authors: []string{"Bea Writer"}},
	})
	two.replace([]catalog.Book{{ID: "two", Title: "Two", Authors: []string{"Ada Author 2"}}})
	service := New([]Library{
		{ID: "one", Name: "One", Store: one},
		{ID: "two", Name: "Two", Store: two},
	})

	var got []string
	for _, item := range service.Books(BooksRequest{Sort: "author", Limit: 10}).Items {
		got = append(got, item.LibraryID+"/"+item.LibraryName+"/"+item.Book.ID)
	}
	want := []string{"two/Two/two", "one/One/ten", "one/One/beta"}
	if len(got) != len(want) {
		t.Fatalf("author shelf: got %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("author shelf: got %v, want %v", got, want)
		}
	}
}

// The same rule for the grouped sorts: an author list that counts up as numbers,
// not bytes, so "Author 2" does not sit after "Author 10".
func TestSortBooksByNameCountsNumbersAsNumbers(t *testing.T) {
	items := []BookResult{
		{LibraryID: "l", Book: catalog.Book{ID: "ten", Title: "Ten", Authors: []string{"Author 10"}}},
		{LibraryID: "l", Book: catalog.Book{ID: "two", Title: "Two", Authors: []string{"Author 2"}}},
	}
	matched := nameFixture(items)
	sortBooksByName(matched, "author")
	if got := fixtureIDs(matched); got != "twoten" {
		t.Fatalf("author order = %s, want two then ten", got)
	}
}

// The series list files books with no series under a group name of its own. Selecting that group has to mean "books
// with no series" — as a series name it matched nothing, and the group opened on an empty shelf.
func TestStandaloneSelectsBooksWithNoSeries(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{
		{ID: "a", Title: "Loose Book"},
		{ID: "b", Title: "Another Loose Book"},
		{ID: "c", Title: "First", Series: "The Expanse"},
	})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	page := service.Books(BooksRequest{LibraryID: "books", Standalone: true, Limit: 60})
	if page.Total != 2 || len(page.Items) != 2 || page.Items[0].Book.ID != "b" || page.Items[1].Book.ID != "a" {
		t.Fatalf("standalone gave total %d, items %+v", page.Total, page.Items)
	}
	name := "The Expanse"
	if page := service.Books(BooksRequest{LibraryID: "books", Series: &name, Limit: 60}); page.Total != 1 {
		t.Fatalf("a real series gave %d books", page.Total)
	}
}
