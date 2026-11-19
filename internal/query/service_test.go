package query

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"mochil/internal/catalog"
)

type testBackend struct {
	mu      sync.RWMutex
	books   []catalog.Book
	version uint64
}

func (b *testBackend) Scan() (catalog.Stats, error) { return catalog.Stats{Books: len(b.Books())}, nil }
func (b *testBackend) Books() []catalog.Book {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]catalog.Book(nil), b.books...)
}
func (b *testBackend) Book(id string) (catalog.Book, bool) {
	for _, book := range b.Books() {
		if book.ID == id {
			return book, true
		}
	}
	return catalog.Book{}, false
}
func (b *testBackend) Acquire(context.Context, catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{}, nil
}
func (b *testBackend) CatalogVersion() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.version
}
func (b *testBackend) replace(books []catalog.Book) {
	b.mu.Lock()
	b.books = append([]catalog.Book(nil), books...)
	b.version++
	b.mu.Unlock()
}

func TestBooksAcrossLibrariesNormalizeTitlesAndUseStableTieBreaks(t *testing.T) {
	first, second := &testBackend{}, &testBackend{}
	first.replace([]catalog.Book{{ID: "z", Title: "Alpha"}, {ID: "apple", Title: "An Apple"}})
	second.replace([]catalog.Book{{ID: "a", Title: "The Alpha"}, {ID: "book", Title: "A Book"}})
	service := New([]Library{
		{ID: "a", Name: "First", Store: first},
		{ID: "b", Name: "Second", Store: second},
	})

	result := service.Books(BooksRequest{Limit: 10})
	want := []string{"a/z", "b/a", "a/apple", "b/book"}
	if len(result.Items) != len(want) {
		t.Fatalf("got %d books, want %d: %+v", len(result.Items), len(want), result.Items)
	}
	for index, item := range result.Items {
		if got := item.LibraryID + "/" + item.Book.ID; got != want[index] {
			t.Fatalf("item %d = %q, want %q", index, got, want[index])
		}
	}
}

func BenchmarkBooksMultiLibraryTitleSort50000(b *testing.B) {
	backends := []*testBackend{{}, {}}
	libraries := []Library{{ID: "a", Name: "First", Store: backends[0]}, {ID: "b", Name: "Second", Store: backends[1]}}
	for libraryIndex, backend := range backends {
		books := make([]catalog.Book, 25_000)
		for index := range books {
			books[index] = catalog.Book{ID: fmt.Sprintf("%08d", index), Title: fmt.Sprintf("The Book Volume %08d", libraryIndex*25_000+index)}
		}
		backend.replace(books)
	}
	service := New(libraries)
	if got := service.Books(BooksRequest{Limit: 500}); len(got.Items) != 500 {
		b.Fatalf("warm-up returned %d books", len(got.Items))
	}
	b.ResetTimer()
	for range b.N {
		if got := service.Books(BooksRequest{Limit: 500}); len(got.Items) != 500 {
			b.Fatalf("returned %d books", len(got.Items))
		}
	}
}

func TestBooksComposesFiltersOrderingAndPagination(t *testing.T) {
	one, two := 1.0, 2.0
	books := &testBackend{}
	books.replace([]catalog.Book{
		{ID: "2", Title: "Later", Authors: []string{"Ada"}, Series: "The Alpha", SeriesIndex: &two},
		{ID: "1", Title: "Earlier", Authors: []string{"Ada"}, Series: "The Alpha", SeriesIndex: &one},
		{ID: "3", Title: "Another", Authors: []string{"Bea"}, Series: "Beta"},
	})
	service := New([]Library{{ID: "books", Name: "Books", Kind: "books", Store: books}})
	series := "The Alpha"
	result := service.Books(BooksRequest{LibraryID: "books", Query: "ada", Series: &series, Bucket: "L", Limit: 1})
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Book.ID != "2" {
		t.Fatalf("unexpected composed result: %+v", result)
	}

	result = service.Books(BooksRequest{LibraryID: "books", Series: &series, Limit: 1, Offset: 1})
	if result.Total != 2 || len(result.Items) != 1 || result.Items[0].Book.ID != "2" {
		t.Fatalf("unexpected series page: %+v", result)
	}
}

func TestBooksRecentSortUsesModifiedTimeAndStableCursorAnchor(t *testing.T) {
	backend := &testBackend{}
	base := time.Unix(1_700_000_000, 0)
	backend.replace([]catalog.Book{
		{ID: "old", Title: "Alpha", Modified: base},
		{ID: "new", Title: "Zulu", Modified: base.Add(2 * time.Hour)},
		{ID: "middle", Title: "Beta", Modified: base.Add(time.Hour)},
	})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	first := service.Books(BooksRequest{LibraryID: "books", Sort: "recent", Limit: 2})
	if len(first.Items) != 2 || first.Items[0].Book.ID != "new" || first.Items[1].Book.ID != "middle" || !first.HasMore {
		t.Fatalf("recent first page = %+v", first)
	}
	second := service.Books(BooksRequest{LibraryID: "books", Sort: "recent", Limit: 2, AfterLibraryID: "books", AfterBookID: "middle"})
	if !second.AnchorFound || len(second.Items) != 1 || second.Items[0].Book.ID != "old" {
		t.Fatalf("recent continuation = %+v", second)
	}
}

func TestBooksKeysetContinuationFindsAnchorAndReportsMore(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{{ID: "a", Title: "Alpha"}, {ID: "b", Title: "Beta"}, {ID: "c", Title: "Gamma"}})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	first := service.Books(BooksRequest{LibraryID: "books", Limit: 1})
	if len(first.Items) != 1 || first.Items[0].Book.ID != "a" || !first.HasMore || !first.AnchorFound {
		t.Fatalf("first keyset page = %+v", first)
	}
	second := service.Books(BooksRequest{LibraryID: "books", Limit: 1, AfterLibraryID: "books", AfterBookID: "a"})
	if len(second.Items) != 1 || second.Items[0].Book.ID != "b" || !second.HasMore || !second.AnchorFound {
		t.Fatalf("second keyset page = %+v", second)
	}
	missing := service.Books(BooksRequest{LibraryID: "books", Limit: 1, AfterLibraryID: "books", AfterBookID: "missing"})
	if missing.AnchorFound || len(missing.Items) != 0 {
		t.Fatalf("missing anchor page = %+v", missing)
	}
}

func TestBookNavigationUsesExactSortedOffsets(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{{ID: "z", Title: "Zulu"}, {ID: "a", Title: "The Alpha"}, {ID: "n", Title: "123 Start"}, {ID: "b", Title: "Beta"}})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	got := service.BookNavigation("books", "")
	if len(got) != 4 || got[0].Bucket != "#" || got[0].Offset != 0 || got[1].Bucket != "A" || got[1].Offset != 1 || got[3].Bucket != "Z" || got[3].Offset != 3 {
		t.Fatalf("navigation = %+v", got)
	}
}

func TestSeriesGroupsStandaloneAndSelectsRepresentative(t *testing.T) {
	one, two := 1.0, 2.0
	books := &testBackend{}
	books.replace([]catalog.Book{
		{ID: "later", Title: "Later", Series: "The Expanse", SeriesIndex: &two},
		{ID: "first", Title: "First", Series: "The Expanse", SeriesIndex: &one},
		{ID: "loose", Title: "Loose Book"},
	})
	service := New([]Library{{ID: "books", Name: "Books", Store: books}})
	groups := service.Series(SeriesRequest{}).Items
	if len(groups) != 2 || groups[0].Name != "The Expanse" || groups[0].Bucket != "E" || groups[0].Count != 2 || groups[0].Representative.ID != "first" || !groups[1].Unsorted {
		t.Fatalf("unexpected groups: %+v", groups)
	}
	standalone := service.Series(SeriesRequest{Bucket: "L"}).Items
	if len(standalone) != 1 || !standalone[0].Unsorted || standalone[0].Representative.ID != "loose" {
		t.Fatalf("unexpected standalone bucket: %+v", standalone)
	}
}

func TestCachedProjectionPagesDoNotExposeMutableIndexStorage(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{{ID: "one", Title: "One", Series: "Series", Authors: []string{"Author"}}})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})

	series := service.Series(SeriesRequest{LibraryID: "books"})
	authors := service.Authors(SeriesRequest{LibraryID: "books"})
	series.Items[0].Name = "Mutated"
	authors.Items[0].Name = "Mutated"

	if got := service.Series(SeriesRequest{LibraryID: "books"}).Items[0].Name; got != "Series" {
		t.Fatalf("cached series was mutated through response: %q", got)
	}
	if got := service.Authors(SeriesRequest{LibraryID: "books"}).Items[0].Name; got != "Author" {
		t.Fatalf("cached author was mutated through response: %q", got)
	}
}

func TestSeriesPaginationReportsFullTotal(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{
		{ID: "a", Title: "One", Series: "Alpha"},
		{ID: "b", Title: "Two", Series: "Beta"},
		{ID: "c", Title: "Three", Series: "Gamma"},
	})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	page := service.Series(SeriesRequest{Limit: 1, Offset: 1})
	if page.Total != 3 || len(page.Items) != 1 || page.Items[0].Name != "Beta" {
		t.Fatalf("unexpected series page: %+v", page)
	}
}

func TestAuthorsGroupMultipleCreditsAndFilterBooks(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{
		{ID: "a", Title: "Alpha", Authors: []string{"Ada", "Bea"}},
		{ID: "b", Title: "Beta", Authors: []string{"ada"}},
		{ID: "u", Title: "Untitled"},
	})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	authors := service.Authors(SeriesRequest{LibraryID: "books"})
	if authors.Total != 3 || authors.Items[0].Name != "Ada" || authors.Items[0].Count != 2 || !authors.Items[2].Unknown {
		t.Fatalf("authors = %+v", authors)
	}
	name := "ADA"
	books := service.Books(BooksRequest{LibraryID: "books", Author: &name, Limit: 10})
	if books.Total != 2 || len(books.Items) != 2 {
		t.Fatalf("author books = %+v", books)
	}
}

func TestAuthorKeysetContinuationUsesAuthorIdentity(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{{ID: "same", Title: "Joint", Authors: []string{"Ada", "Bea"}}, {ID: "c", Title: "Other", Authors: []string{"Cia"}}})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	page := service.Authors(SeriesRequest{LibraryID: "books", Limit: 1, AfterLibraryID: "books", AfterID: "Ada"})
	if !page.AnchorFound || len(page.Items) != 1 || page.Items[0].Name != "Bea" || !page.HasMore {
		t.Fatalf("author continuation = %+v", page)
	}
}

func TestSeriesUsesNaturalNameOrder(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{
		{ID: "ten", Title: "Ten", Series: "Series 10"},
		{ID: "two", Title: "Two", Series: "Series 2"},
	})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	groups := service.Series(SeriesRequest{LibraryID: "books"}).Items
	if len(groups) != 2 || groups[0].Name != "Series 2" || groups[1].Name != "Series 10" {
		t.Fatalf("series order = %+v", groups)
	}
}

func TestSeriesKeysetContinuationUsesRepresentativeIdentity(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{{ID: "a", Title: "One", Series: "Alpha"}, {ID: "b", Title: "Two", Series: "Beta"}, {ID: "c", Title: "Three", Series: "Gamma"}})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	page := service.Series(SeriesRequest{LibraryID: "books", Limit: 1, AfterLibraryID: "books", AfterID: "a"})
	if len(page.Items) != 1 || page.Items[0].Name != "Beta" || !page.AnchorFound || !page.HasMore {
		t.Fatalf("series keyset page = %+v", page)
	}
}

func TestSeriesOrdersMainVolumesBeforeNestedSideStories(t *testing.T) {
	one, two, thirteen := 1.0, 2.0, 13.0
	backend := &testBackend{}
	backend.replace([]catalog.Book{
		{ID: "alter-2", Title: "86 Alter, Vol. 2", Path: "86/!Side Stories/alter-2.epub", Series: "86", SeriesIndex: &two},
		{ID: "main-13", Title: "86, Vol. 13", Path: "86/main-13.epub", Series: "86", SeriesIndex: &thirteen},
		{ID: "main-1", Title: "86, Vol. 1", Path: "86/main-1.epub", Series: "86", SeriesIndex: &one},
	})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	series := "86"
	got := service.Books(BooksRequest{LibraryID: "books", Series: &series, Limit: 10}).Items
	if len(got) != 3 || got[0].Book.ID != "main-1" || got[1].Book.ID != "main-13" || got[2].Book.ID != "alter-2" {
		t.Fatalf("unexpected series order: %+v", got)
	}
}

func TestIndexRefreshesOnPublishedVersion(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{{ID: "old", Title: "Old", Authors: []string{"One"}}})
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	if got := service.Books(BooksRequest{Query: "one"}); got.Total != 1 || got.Items[0].Book.ID != "old" {
		t.Fatalf("unexpected initial result: %+v", got)
	}
	backend.replace([]catalog.Book{{ID: "new", Title: "New", Authors: []string{"Two"}}})
	if got := service.Books(BooksRequest{Query: "two"}); got.Total != 1 || got.Items[0].Book.ID != "new" {
		t.Fatalf("index did not refresh: %+v", got)
	}
}

func BenchmarkBooksWarmIndex50000(b *testing.B) {
	backend := &testBackend{}
	books := make([]catalog.Book, 50_000)
	for index := range books {
		books[index] = catalog.Book{ID: fmt.Sprintf("%08d", index), Title: fmt.Sprintf("Book %08d", index), Authors: []string{"Example Author"}, Series: fmt.Sprintf("Series %04d", index/10)}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	service.Books(BooksRequest{Limit: 120})
	b.ResetTimer()
	for range b.N {
		service.Books(BooksRequest{Limit: 120})
	}
}

func BenchmarkLibrariesSummary100000(b *testing.B) {
	backend := &testBackend{}
	books := make([]catalog.Book, 100_000)
	for index := range books {
		books[index] = catalog.Book{ID: fmt.Sprintf("%08d", index), Title: fmt.Sprintf("Book %08d", index)}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}})
	service.Libraries()
	b.ResetTimer()
	for range b.N {
		result := service.Libraries()
		if len(result) != 1 || result[0].Count != 100_000 {
			b.Fatalf("unexpected library summary: %+v", result)
		}
	}
}

func BenchmarkBooksDeepCursor50000(b *testing.B) {
	backend := &testBackend{}
	books := make([]catalog.Book, 50_000)
	for index := range books {
		books[index] = catalog.Book{ID: fmt.Sprintf("%08d", index), Title: fmt.Sprintf("Book %08d", index)}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	service.Books(BooksRequest{LibraryID: "books", Limit: 120})
	b.ResetTimer()
	for range b.N {
		result := service.Books(BooksRequest{LibraryID: "books", Limit: 120, AfterLibraryID: "books", AfterBookID: "00049000"})
		if len(result.Items) != 120 || result.Items[0].Book.ID != "00049001" {
			b.Fatalf("unexpected deep page: %+v", result)
		}
	}
}

func BenchmarkBooksRecentDeepCursor50000(b *testing.B) {
	backend := &testBackend{}
	books := make([]catalog.Book, 50_000)
	base := time.Unix(1_700_000_000, 0)
	for index := range books {
		books[index] = catalog.Book{ID: fmt.Sprintf("%08d", index), Title: fmt.Sprintf("Book %08d", index), Modified: base.Add(time.Duration(index) * time.Second)}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	service.Books(BooksRequest{LibraryID: "books", Sort: "recent", Limit: 120})
	b.ResetTimer()
	for range b.N {
		result := service.Books(BooksRequest{LibraryID: "books", Sort: "recent", Limit: 120, AfterLibraryID: "books", AfterBookID: "00001000"})
		if len(result.Items) != 120 || result.Items[0].Book.ID != "00000999" {
			b.Fatalf("unexpected recent page: %+v", result)
		}
	}
}

func BenchmarkSeriesDeepCursor50000(b *testing.B) {
	backend := &testBackend{}
	books := make([]catalog.Book, 50_000)
	for index := range books {
		books[index] = catalog.Book{ID: fmt.Sprintf("%08d", index), Title: fmt.Sprintf("Book %08d", index), Series: fmt.Sprintf("Series %04d", index/10)}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	service.Series(SeriesRequest{LibraryID: "books", Limit: 80})
	b.ResetTimer()
	for range b.N {
		result := service.Series(SeriesRequest{LibraryID: "books", Limit: 80, AfterLibraryID: "books", AfterID: "00049000"})
		if len(result.Items) != 80 || result.Items[0].Name != "Series 4901" {
			b.Fatalf("unexpected deep series page: %+v", result)
		}
	}
}

func BenchmarkAuthorsDeepCursor50000(b *testing.B) {
	backend := &testBackend{}
	books := make([]catalog.Book, 50_000)
	for index := range books {
		books[index] = catalog.Book{ID: fmt.Sprintf("%08d", index), Title: fmt.Sprintf("Book %08d", index), Authors: []string{fmt.Sprintf("Author %04d", index/10)}}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	service.Authors(SeriesRequest{LibraryID: "books", Limit: 80})
	b.ResetTimer()
	for range b.N {
		result := service.Authors(SeriesRequest{LibraryID: "books", Limit: 80, AfterLibraryID: "books", AfterID: "Author 4900"})
		if len(result.Items) != 80 || result.Items[0].Name != "Author 4901" {
			b.Fatalf("unexpected deep author page: %+v", result)
		}
	}
}

func BenchmarkSeriesIndexPublication50000(b *testing.B) {
	backend := &testBackend{}
	books := make([]catalog.Book, 50_000)
	for index := range books {
		books[index] = catalog.Book{
			ID:      fmt.Sprintf("%08d", index),
			Title:   fmt.Sprintf("Book %08d", index),
			Authors: []string{"Example Author"},
			Series:  fmt.Sprintf("Series %04d", index/10),
		}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	service.Series(SeriesRequest{LibraryID: "books", Limit: 80})
	b.ResetTimer()
	for iteration := range b.N {
		books[len(books)-1].Description = fmt.Sprintf("revision %d", iteration)
		backend.replace(books)
		result := service.Series(SeriesRequest{LibraryID: "books", Limit: 80})
		if result.Total != 5_000 || len(result.Items) != 80 {
			b.Fatalf("unexpected rebuilt series page: %+v", result)
		}
	}
}

func BenchmarkSeriesFiltered50000(b *testing.B) {
	backend := &testBackend{}
	books := make([]catalog.Book, 50_000)
	for index := range books {
		books[index] = catalog.Book{
			ID:      fmt.Sprintf("%08d", index),
			Title:   fmt.Sprintf("Book %08d", index),
			Authors: []string{"Example Author"},
			Series:  fmt.Sprintf("Series %04d", index/10),
		}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	service.Series(SeriesRequest{LibraryID: "books", Limit: 80})
	b.ResetTimer()
	for range b.N {
		result := service.Series(SeriesRequest{LibraryID: "books", Query: "example", Limit: 80})
		if result.Total != 5_000 || len(result.Items) != 80 {
			b.Fatalf("unexpected filtered series page: %+v", result)
		}
	}
}
