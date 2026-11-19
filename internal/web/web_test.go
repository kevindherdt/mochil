package web

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/catalog"
)

func TestEmptyCatalogIsValidAtom(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(t.TempDir(), "cache.json")
	store := catalog.NewStore(dir, cache)
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	h, err := New(Config{}, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/opds", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `xmlns="http://www.w3.org/2005/Atom"`) {
		t.Fatalf("not Atom: %s", w.Body.String())
	}
}

type streamingBackend struct{ book catalog.Book }

func (b streamingBackend) Scan() (catalog.Stats, error)        { return catalog.Stats{Books: 1}, nil }
func (b streamingBackend) Books() []catalog.Book               { return []catalog.Book{b.book} }
func (b streamingBackend) Book(id string) (catalog.Book, bool) { return b.book, id == b.book.ID }
func (b streamingBackend) Acquire(context.Context, catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{}, nil
}
func (b streamingBackend) Page(context.Context, catalog.Book, int) (catalog.Page, error) {
	return catalog.Page{Reader: io.NopCloser(bytes.NewReader([]byte("page"))), MediaType: "image/jpeg", Size: 4}, nil
}

func TestFeedAdvertisesPageStreaming(t *testing.T) {
	backend := streamingBackend{book: catalog.Book{
		ID: "comic", Path: "Comic.cbz", Title: "Comic", Authors: []string{"Writer"},
		Format: "cbz", MediaType: "application/vnd.comicbook+zip", PageCount: 3,
		PageMediaType: "image/jpeg", Modified: time.Unix(1, 0),
	}}
	handler, err := New(Config{PageStreaming: true}, backend, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/opds/titles?bucket=C", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	if !strings.Contains(body, `xmlns:pse="http://vaemendis.net/opds-pse/ns"`) || !strings.Contains(body, `pse:count="3"`) {
		t.Fatalf("missing OPDS-PSE metadata: %s", body)
	}
}

func TestBookViewAdvertisesUsablePageStreaming(t *testing.T) {
	book := catalog.Book{ID: "comic", Path: "Comic.cbz", Title: "Comic", Format: "cbz", PageCount: 3, PageMediaType: "image/jpeg"}
	backend := streamingBackend{book: book}
	server := &multiServer{}
	view := server.bookView(Library{ID: "manga", Name: "Manga", PageStreaming: true, Store: backend}, book)
	if view.PageStreamURL != "/stream/manga/comic?page={pageNumber}" {
		t.Fatalf("page stream URL = %q", view.PageStreamURL)
	}
	view = server.bookView(Library{ID: "manga", Name: "Manga", PageStreaming: false, Store: backend}, book)
	if view.PageStreamURL != "" {
		t.Fatalf("disabled page streaming leaked URL %q", view.PageStreamURL)
	}
}

func TestBasicAuth(t *testing.T) {
	dir := t.TempDir()
	store := catalog.NewStore(dir, filepath.Join(t.TempDir(), "c.json"))
	_, _ = store.Scan()
	h, _ := New(Config{Username: "reader", Password: "secret"}, store, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	r := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("status %d", w.Code)
	}
}

func TestOPDSEntryTellsAReadAlongBookApart(t *testing.T) {
	server := &Server{cfg: Config{AcquisitionPath: "/download"}}
	plain := server.bookEntry(catalog.Book{ID: "a", Title: "Plain", Path: "plain.epub"}, false)
	narrated := server.bookEntry(catalog.Book{ID: "b", Title: "Narrated", Path: "narrated.epub", ReadAlong: true, Description: "A story."}, false)
	if plain.Title != "Plain" || strings.Contains(plain.Content.Text, "recording") {
		t.Fatalf("plain book was marked: %+v", plain)
	}
	if narrated.Title != "Narrated (read-along)" || !strings.HasPrefix(narrated.Content.Text, "Has its own recording.") || !strings.Contains(narrated.Content.Text, "A story.") {
		t.Fatalf("read-along book was not marked: %+v", narrated)
	}
}

type countingBooksBackend struct {
	books []catalog.Book
	reads int
}

func (b *countingBooksBackend) Scan() (catalog.Stats, error) {
	return catalog.Stats{Books: len(b.books)}, nil
}
func (b *countingBooksBackend) Books() []catalog.Book {
	b.reads++
	return append([]catalog.Book(nil), b.books...)
}
func (b *countingBooksBackend) Book(id string) (catalog.Book, bool) {
	for _, book := range b.books {
		if book.ID == id {
			return book, true
		}
	}
	return catalog.Book{}, false
}
func (b *countingBooksBackend) Acquire(context.Context, catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{}, nil
}

// TestOPDSHandlersCloneTheCatalogueOnlyOnce guards against the catalog,
// series, authors, titles, recent and search handlers cloning the whole
// catalogue (store.Books() deep-copies it) more than once while serving a
// single request.
func TestOPDSHandlersCloneTheCatalogueOnlyOnce(t *testing.T) {
	backend := &countingBooksBackend{books: []catalog.Book{
		{ID: "one", Title: "One", Series: "Saga", AuthorSort: []string{"Writer"}},
		{ID: "two", Title: "Two", Series: "Saga", AuthorSort: []string{"Other"}},
	}}
	handler, err := New(Config{}, backend, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/opds", "/opds/series", "/opds/authors", "/opds/titles", "/opds/recent", "/opds/search?q=one"} {
		backend.reads = 0
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 200 {
			t.Fatalf("%s status %d: %s", path, response.Code, response.Body.String())
		}
		if backend.reads != 1 {
			t.Fatalf("%s cloned the catalogue %d times, want 1", path, backend.reads)
		}
	}
}
