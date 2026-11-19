package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"mochil/internal/catalog"
)

// TestOPDSRootClonesEachLibraryOnlyOnce guards against opds_root.go's catalog
// handler cloning every library's catalogue twice per request: once while
// building the combined feed and again while building each library's nav
// entry. Both passes need the same book list, so the handler must compute it
// once per library and reuse it.
func TestOPDSRootClonesEachLibraryOnlyOnce(t *testing.T) {
	books := &versionedStaticBackend{staticBackend: staticBackend{books: []catalog.Book{{ID: "one", Title: "One"}}}, version: 1}
	manga := &versionedStaticBackend{staticBackend: staticBackend{books: []catalog.Book{{ID: "two", Title: "Two"}}}, version: 1}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{
		{ID: "books", Name: "Books", Kind: "books", Store: books},
		{ID: "manga", Name: "Manga", Kind: "manga", Store: manga},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/opds", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("root feed status = %d: %s", response.Code, response.Body.String())
	}
	if books.reads != 1 {
		t.Fatalf("books library catalogue was cloned %d times, want 1", books.reads)
	}
	if manga.reads != 1 {
		t.Fatalf("manga library catalogue was cloned %d times, want 1", manga.reads)
	}
}
