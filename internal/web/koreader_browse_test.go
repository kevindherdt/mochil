package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"mochil/internal/catalog"
	catalogquery "mochil/internal/query"
	"mochil/internal/reading"
)

func ptr(value float64) *float64 { return &value }

func browseTestServer(t *testing.T) (*multiServer, string) {
	t.Helper()
	server, token := updateTestServer(t)
	backend := staticBackend{books: []catalog.Book{
		{ID: "a2", Path: "Dune/Dune 2.epub", Title: "Dune Messiah", Authors: []string{"Frank Herbert"}, Series: "Dune", SeriesIndex: ptr(2), Size: 20},
		{ID: "a1", Path: "Dune/Dune 1.epub", Title: "Dune", Authors: []string{"Frank Herbert"}, Series: "Dune", SeriesIndex: ptr(1), Size: 10},
		{ID: "n1", Path: "Neuromancer.epub", Title: "Neuromancer", Authors: []string{"William Gibson"}, Size: 30},
	}}
	library := Library{ID: "books", Name: "Books", Kind: "books", Store: backend}
	server.libraries = []Library{library}
	server.query = catalogquery.New([]catalogquery.Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}})
	collections, err := newCollectionStore(filepath.Join(t.TempDir(), "collections.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.collections = collections
	return server, token
}

func browseRequest(t *testing.T, server *multiServer, token, query string) koReaderBrowse {
	t.Helper()
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/browse?"+query, token)
	if response.Code != http.StatusOK {
		t.Fatalf("browse %s = %d %s", query, response.Code, response.Body.String())
	}
	var result koReaderBrowse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestKoReaderBookReadAlongMetadata(t *testing.T) {
	for _, readAlong := range []bool{false, true} {
		book := koReaderBookOf("library", catalog.Book{ID: "a", Title: "Neuromancer", Path: "book.epub", ReadAlong: readAlong})
		data, err := json.Marshal(book)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if (got["read_along"] == true) != readAlong {
			t.Fatalf("read_along=%v: %s", readAlong, data)
		}
	}
}

func TestKoReaderBrowseAdvertisesStreamingOnlyWhenLibraryHasPageStreamingEnabled(t *testing.T) {
	for _, pageStreaming := range []bool{false, true} {
		server, token := updateTestServer(t)
		backend := streamingBackend{book: catalog.Book{
			ID: "comic", Path: "Comic.cbz", Title: "Comic", Format: "cbz",
			MediaType: "application/vnd.comicbook+zip", PageCount: 3, PageMediaType: "image/jpeg",
		}}
		library := Library{ID: "manga", Name: "Manga", Kind: "manga", Store: backend, PageStreaming: pageStreaming}
		server.libraries = []Library{library}
		server.query = catalogquery.New([]catalogquery.Library{{ID: "manga", Name: "Manga", Kind: "manga", Store: backend}})
		books := browseRequest(t, server, token, "kind=books")
		if len(books.Books) != 1 {
			t.Fatalf("pageStreaming=%v books=%+v", pageStreaming, books)
		}
		book := books.Books[0]
		if book.Streamable != pageStreaming {
			t.Fatalf("pageStreaming=%v streamable=%v", pageStreaming, book.Streamable)
		}
		wantPageCount := 0
		if pageStreaming {
			wantPageCount = 3
		}
		if book.PageCount != wantPageCount {
			t.Fatalf("pageStreaming=%v page_count=%d", pageStreaming, book.PageCount)
		}
	}
}

func TestKoReaderBrowseResumesAtLastReadPageForStreamableBooks(t *testing.T) {
	server, token := updateTestServer(t)
	backend := streamingBackend{book: catalog.Book{
		ID: "comic", Path: "Comic.cbz", Title: "Comic", Format: "cbz",
		MediaType: "application/vnd.comicbook+zip", PageCount: 10, PageMediaType: "image/jpeg",
	}}
	library := Library{ID: "manga", Name: "Manga", Kind: "manga", Store: backend, PageStreaming: true}
	server.libraries = []Library{library}
	server.query = catalogquery.New([]catalogquery.Library{{ID: "manga", Name: "Manga", Kind: "manga", Store: backend}})

	books := browseRequest(t, server, token, "kind=books")
	if len(books.Books) != 1 || books.Books[0].LastPage != 0 {
		t.Fatalf("with no reading position, last_page should be unset (page 1): %+v", books.Books)
	}

	if _, err := server.reading.Put("reader", reading.Position{LibraryID: "manga", BookID: "comic", Fraction: 0.5}, 0); err != nil {
		t.Fatal(err)
	}
	books = browseRequest(t, server, token, "kind=books")
	if len(books.Books) != 1 || books.Books[0].LastPage != 6 {
		t.Fatalf("expected last_page=6 (50%% of 10 pages, 1-indexed): %+v", books.Books)
	}
}

func TestPluginBrowseNeedsAPairedDeviceAndOnlyReads(t *testing.T) {
	server, token := browseTestServer(t)
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/browse?kind=books", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("browse without a token = %d", response.Code)
	}
	if response := pluginUpdateRequest(server, http.MethodPost, "/api/koreader/v1/browse?kind=books", token); response.Code != http.StatusNotFound {
		t.Fatalf("browse accepted a POST: %d", response.Code)
	}
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/download/books/n1", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("download without a token = %d", response.Code)
	}
}

func TestPluginBrowseHugeGroupOffsetReturnsEmptyPage(t *testing.T) {
	server, token := browseTestServer(t)
	offset := int(^uint(0) >> 1)
	got := browseRequest(t, server, token, fmt.Sprintf("kind=series&offset=%d", offset))
	if got.Total != 1 || len(got.Groups) != 0 {
		t.Fatalf("huge offset returned %+v; want total 1 and no groups", got)
	}
}

func TestPluginBrowseListsBooksSeriesAndAuthors(t *testing.T) {
	server, token := browseTestServer(t)
	books := browseRequest(t, server, token, "kind=books")
	if books.Total != 3 || len(books.Books) != 3 {
		t.Fatalf("books = %+v", books)
	}
	found := browseRequest(t, server, token, "kind=books&q=gibson")
	if len(found.Books) != 1 || found.Books[0].ID != "n1" || found.Books[0].Format != "epub" || found.Books[0].Library != "books" {
		t.Fatalf("search = %+v", found)
	}
	series := browseRequest(t, server, token, "kind=series")
	if len(series.Groups) != 1 || series.Groups[0].Name != "Dune" || series.Groups[0].Count != 2 || series.Groups[0].Library != "books" {
		t.Fatalf("series = %+v", series)
	}
	authors := browseRequest(t, server, token, "kind=authors")
	if authors.Total < 2 {
		t.Fatalf("authors = %+v", authors)
	}
}

func TestPluginBrowseOpensASeriesInVolumeOrderAndPages(t *testing.T) {
	server, token := browseTestServer(t)
	inSeries := browseRequest(t, server, token, "kind=series&series="+url.QueryEscape("Dune")+"&library=books")
	if len(inSeries.Books) != 2 || inSeries.Books[0].ID != "a1" || inSeries.Books[1].ID != "a2" || inSeries.Kind != "books" {
		t.Fatalf("series books = %+v (volumes must be in order)", inSeries)
	}
	second := browseRequest(t, server, token, "kind=series&series=Dune&library=books&limit=1&offset=1")
	if second.Total != 2 || len(second.Books) != 1 || second.Books[0].ID != "a2" {
		t.Fatalf("second page = %+v", second)
	}
	byAuthor := browseRequest(t, server, token, "kind=authors&author="+url.QueryEscape("Frank Herbert")+"&library=books")
	if len(byAuthor.Books) != 2 {
		t.Fatalf("author books = %+v", byAuthor)
	}
}

func TestPluginBrowseLargeSeriesPagesInVolumeOrder(t *testing.T) {
	server, token := browseTestServer(t)
	books := make([]catalog.Book, 502)
	for i := range books {
		// Reverse title order and vary depth so the query's default order differs
		// from the plugin's volume order.
		volume := float64(502 - i)
		path := fmt.Sprintf("nested/%d/book.epub", i)
		if i%2 == 0 {
			path = fmt.Sprintf("%d.epub", i)
		}
		books[i] = catalog.Book{ID: fmt.Sprintf("%03d", 502-i), Title: fmt.Sprintf("Book %03d", i), Path: path, Series: "Saga", SeriesIndex: &volume}
	}
	backend := staticBackend{books: books}
	server.query = catalogquery.New([]catalogquery.Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}})
	for _, offset := range []int{0, 99, 499, 500, 501, 502, 503} {
		got := browseRequest(t, server, token, fmt.Sprintf("kind=series&series=Saga&library=books&limit=1&offset=%d", offset))
		if got.Total != 502 {
			t.Fatalf("offset %d: total = %d", offset, got.Total)
		}
		if offset >= 502 {
			if len(got.Books) != 0 {
				t.Fatalf("offset %d: got %v", offset, got.Books)
			}
		} else if len(got.Books) != 1 || got.Books[0].ID != fmt.Sprintf("%03d", offset+1) {
			t.Fatalf("offset %d: books = %+v, want volume %d", offset, got.Books, offset+1)
		}
	}
}

// A series/authors group LIST (kind=series or kind=authors with no series/author selected) must return
// every matching group, not just the first MaxPageSize of them, now that koReaderGroups asks the query
// layer for the whole set in one unlimited call instead of paging through it in MaxPageSize chunks.
func TestPluginBrowseSeriesGroupListExceedsMaxPageSizeWithoutTruncation(t *testing.T) {
	server, token := browseTestServer(t)
	const seriesCount = 600
	books := make([]catalog.Book, seriesCount)
	for i := range books {
		volume := 1.0
		books[i] = catalog.Book{ID: fmt.Sprintf("book-%03d", i), Title: fmt.Sprintf("Title %03d", i), Series: fmt.Sprintf("Series %03d", i), SeriesIndex: &volume}
	}
	backend := staticBackend{books: books}
	server.query = catalogquery.New([]catalogquery.Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}})
	first := browseRequest(t, server, token, "kind=series&library=books&limit=50&offset=0")
	if first.Total != seriesCount || len(first.Groups) != 50 {
		t.Fatalf("first page: total=%d groups=%d, want total=%d groups=50", first.Total, len(first.Groups), seriesCount)
	}
	last := browseRequest(t, server, token, fmt.Sprintf("kind=series&library=books&limit=50&offset=%d", seriesCount-10))
	if last.Total != seriesCount || len(last.Groups) != 10 {
		t.Fatalf("last page: total=%d groups=%d, want total=%d groups=10", last.Total, len(last.Groups), seriesCount)
	}
}

func TestPluginBrowseSeriesSortsAcrossLibrariesAndMissingVolumes(t *testing.T) {
	server, token := browseTestServer(t)
	a := staticBackend{books: []catalog.Book{
		{ID: "later", Title: "A", Series: "Saga", SeriesIndex: ptr(2)},
		{ID: "missing", Title: "Missing", Series: "Saga"},
	}}
	b := staticBackend{books: []catalog.Book{
		{ID: "first", Title: "Z", Series: "Saga", SeriesIndex: ptr(1)},
		{ID: "tied", Title: "B", Series: "Saga", SeriesIndex: ptr(2)},
	}}
	server.query = catalogquery.New([]catalogquery.Library{
		{ID: "a", Name: "A", Kind: "books", Store: a},
		{ID: "b", Name: "B", Kind: "books", Store: b},
	})
	want := []string{"first", "later", "tied", "missing"}
	for offset, id := range want {
		got := browseRequest(t, server, token, fmt.Sprintf("kind=series&series=Saga&limit=1&offset=%d", offset))
		if got.Total != len(want) || len(got.Books) != 1 || got.Books[0].ID != id {
			t.Fatalf("offset %d: got %+v, want %s", offset, got, id)
		}
	}
}

func TestPluginBrowseRejectsAnUnknownView(t *testing.T) {
	server, token := browseTestServer(t)
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/browse?kind=videos", token); response.Code != http.StatusBadRequest {
		t.Fatalf("unknown view = %d", response.Code)
	}
}

func TestPluginDownloadSendsTheBookFileOrNotFound(t *testing.T) {
	server, token := browseTestServer(t)
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/download/books/n1", token)
	if response.Code != http.StatusOK || response.Body.String() != "Neuromancer" {
		t.Fatalf("download = %d %q", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != "attachment; filename*=UTF-8''Neuromancer.epub" {
		t.Fatalf("disposition = %q", disposition)
	}
	for _, path := range []string{"/api/koreader/v1/download/books/missing", "/api/koreader/v1/download/nowhere/n1", "/api/koreader/v1/download/books", "/api/koreader/v1/download/books/n1%2F..%2Fsecret"} {
		if response := pluginUpdateRequest(server, http.MethodGet, path, token); response.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", path, response.Code)
		}
	}
}

func TestPluginHomeCountsMatchTheListsItOpens(t *testing.T) {
	server, token := browseTestServer(t)
	home := server.koReaderHomeView("reader", "https://books.example")
	if series := browseRequest(t, server, token, "kind=series"); home.Series != series.Total || home.Series != 1 {
		t.Fatalf("home says %d series, the list has %d (\"Standalone books\" is not a series)", home.Series, series.Total)
	}
	if authors := browseRequest(t, server, token, "kind=authors"); home.Authors != authors.Total {
		t.Fatalf("home says %d authors, the list has %d", home.Authors, authors.Total)
	}
}

func TestPluginHomeContinueReadingSaysWhichBookItIs(t *testing.T) {
	server, _ := browseTestServer(t)
	if _, err := server.reading.Put("reader", reading.Position{LibraryID: "books", BookID: "a1", Fraction: 0.4, UpdatedAt: 100}, 0); err != nil {
		t.Fatal(err)
	}
	home := server.koReaderHomeView("reader", "https://books.example")
	if len(home.Continue) != 1 || home.Continue[0].Book == nil {
		t.Fatalf("continue = %+v", home.Continue)
	}
	book := home.Continue[0].Book
	if book.ID != "a1" || book.Library != "books" || book.Series != "Dune" || book.Format != "epub" {
		t.Fatalf("book = %+v (the plugin needs it to open or fetch the book)", book)
	}
}

func TestPluginHomeHTTPIncludesMangaSeriesAndEditionMetadata(t *testing.T) {
	server, token := browseTestServer(t)
	backend := staticBackend{books: []catalog.Book{
		{ID: "chapter", Path: "chapter.cbz", Title: "Chapter 0", Series: "Aku no Kyouten", ReadAlong: true},
	}}
	server.libraries = []Library{{ID: "manga", Name: "Manga", Kind: "manga", Store: backend}}
	server.query = catalogquery.New([]catalogquery.Library{{ID: "manga", Name: "Manga", Kind: "manga", Store: backend}})
	if _, err := server.reading.Put("reader", reading.Position{LibraryID: "manga", BookID: "chapter", Fraction: 0.16, UpdatedAt: 100}, 0); err != nil {
		t.Fatal(err)
	}
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/home", token)
	if response.Code != http.StatusOK {
		t.Fatalf("home status = %d", response.Code)
	}
	var home koReaderHome
	if err := json.Unmarshal(response.Body.Bytes(), &home); err != nil {
		t.Fatal(err)
	}
	if len(home.Continue) != 1 || home.Continue[0].Book == nil {
		t.Fatalf("continue = %+v", home.Continue)
	}
	entry := home.Continue[0]
	if entry.Title != "Chapter 0" || entry.Book.Series != "Aku no Kyouten" || entry.Book.ID != "chapter" || entry.Book.Library != "manga" || !entry.Book.ReadAlong {
		t.Fatalf("home entry = %+v book = %+v", entry, entry.Book)
	}
}

func TestPluginLibrariesAreChosenFirstAndFilterTheirSeriesAndAuthors(t *testing.T) {
	server, token := browseTestServer(t)
	second := staticBackend{books: []catalog.Book{
		{ID: "r1", Path: "Rebuild/Rebuild 1.epub", Title: "Rebuild World 1", Authors: []string{"Nahuse"}, Series: "Rebuild World", SeriesIndex: ptr(1), Size: 5},
		{ID: "r2", Path: "Rebuild/Rebuild 2.epub", Title: "Rebuild World 2", Authors: []string{"Nahuse"}, Series: "Rebuild World", SeriesIndex: ptr(2), Size: 5},
	}}
	server.libraries = append(server.libraries,
		Library{ID: "light-novels", Name: "Light novels", Kind: "books", Store: second},
		Library{ID: "music", Name: "Music", Kind: "music", Store: second})
	server.query = catalogquery.New([]catalogquery.Library{
		{ID: "books", Name: "Books", Kind: "books", Store: server.libraries[0].Store},
		{ID: "light-novels", Name: "Light novels", Kind: "books", Store: second},
	})
	if _, err := server.collections.create("reader", "Favourites", "light-novels", nil); err != nil {
		t.Fatal(err)
	}
	libraries := browseRequest(t, server, token, "kind=libraries")
	if libraries.Total != 2 || libraries.Groups[0].Name != "Books" || libraries.Groups[1].Name != "Light novels" {
		t.Fatalf("libraries = %+v (only book libraries, in configured order; music is not one)", libraries.Groups)
	}
	if got := libraries.Groups[1]; got.Library != "light-novels" || got.Count != 2 || got.Series != 1 || got.Authors != 1 {
		t.Fatalf("light novels = %+v", got)
	}
	if got := libraries.Groups[0]; got.Count != 3 || got.Series != 1 || got.Authors != 2 {
		t.Fatalf("books = %+v", got)
	}
	if series := browseRequest(t, server, token, "kind=series&library=light-novels"); series.Total != 1 || series.Groups[0].Name != "Rebuild World" {
		t.Fatalf("series of one library = %+v", series)
	}
	if series := browseRequest(t, server, token, "kind=series&library=books"); series.Total != 1 || series.Groups[0].Name != "Dune" {
		t.Fatalf("series of the other library = %+v", series)
	}
	if authors := browseRequest(t, server, token, "kind=authors&library=light-novels"); authors.Total != 1 || authors.Groups[0].Name != "Nahuse" {
		t.Fatalf("authors of one library = %+v", authors)
	}
	if mine := server.koReaderLibraries("reader", ""); mine[1].Collections != 1 || mine[0].Collections != 0 {
		t.Fatalf("collections per library = %+v", mine)
	}
	home := server.koReaderHomeView("reader", "")
	if home.Libraries != 2 {
		t.Fatalf("home libraries = %d", home.Libraries)
	}
}

func TestKoReaderBrowseReportsWhetherABookIsMarkedFinished(t *testing.T) {
	server, token := browseTestServer(t)
	unfinished := browseRequest(t, server, token, "kind=books")
	for _, book := range unfinished.Books {
		if book.Finished {
			t.Fatalf("a1 must not start out finished: %+v", book)
		}
	}
	if _, err := server.reading.SetFinished("reader", "books", "a1", true); err != nil {
		t.Fatal(err)
	}
	finished := browseRequest(t, server, token, "kind=books")
	var found bool
	for _, book := range finished.Books {
		if book.ID == "a1" {
			found = true
			if !book.Finished {
				t.Fatalf("a1 should now report finished: %+v", book)
			}
		} else if book.Finished {
			t.Fatalf("only a1 should be finished, got %+v", book)
		}
	}
	if !found {
		t.Fatal("a1 missing from the browse list")
	}
}

// A paging request that names an offset past the end of the list must answer an
// empty page, not overflow the offset+limit arithmetic and panic while slicing.
func TestKoReaderCollectionBrowseClampsHugeOffset(t *testing.T) {
	server, token := browseTestServer(t)
	if _, err := server.collections.create("reader", "Picks", "books", nil); err != nil {
		t.Fatal(err)
	}
	got := browseRequest(t, server, token, "kind=collections&offset=9223372036854775807&limit=100")
	if got.Total != 1 || len(got.Groups) != 0 {
		t.Fatalf("got total %d with %d groups, want total 1 and no groups", got.Total, len(got.Groups))
	}
}
