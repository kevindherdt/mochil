package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/catalog"
)

// TestResolveCollectionClonesEachLibraryOnceRegardlessOfMemberCount guards
// against resolveCollection calling Store.Books() once per collection
// member, which turns a single /api/collections poll into
// O(collections x members) full catalogue clones.
func TestResolveCollectionClonesEachLibraryOnceRegardlessOfMemberCount(t *testing.T) {
	backend := &versionedStaticBackend{staticBackend: staticBackend{books: []catalog.Book{
		{ID: "one", Title: "One", Series: "Saga"},
		{ID: "two", Title: "Two", Series: "Saga"},
		{ID: "three", Title: "Three"},
	}}, version: 1}
	path := filepath.Join(t.TempDir(), "collections.json")
	handler, err := NewMulti(MultiConfig{AllowOpen: true, CollectionsState: path}, []Library{{ID: "books", Name: "Books", Store: backend}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, target, body string) *httptest.ResponseRecorder {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, target, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
		} else {
			r = httptest.NewRequest(method, target, nil)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	created := request(http.MethodPost, "/api/collections", `{"name":"Favorites","library_id":"books"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", created.Code, created.Body.String())
	}
	var collection struct{ ID string }
	if err := json.Unmarshal(created.Body.Bytes(), &collection); err != nil {
		t.Fatal(err)
	}
	for _, bookID := range []string{"one", "two", "three"} {
		member := request(http.MethodPost, "/api/collections/"+collection.ID+"/members", `{"type":"book","library_id":"books","book_id":"`+bookID+`"}`)
		if member.Code != http.StatusNoContent {
			t.Fatalf("add member %s status %d: %s", bookID, member.Code, member.Body.String())
		}
	}
	backend.reads = 0
	books := request(http.MethodGet, "/api/collections/"+collection.ID+"/books", "")
	if books.Code != http.StatusOK {
		t.Fatalf("books status %d: %s", books.Code, books.Body.String())
	}
	if backend.reads != 1 {
		t.Fatalf("library catalogue was cloned %d times for 3 members, want 1", backend.reads)
	}
}

// TestOPDSCollectionsRootClonesEachLibraryOnlyOnce guards against
// opdsCollections building the combined book list from every library and
// then cloning each library's catalogue again to compute per-library stats.
func TestOPDSCollectionsRootClonesEachLibraryOnlyOnce(t *testing.T) {
	backend := &versionedStaticBackend{staticBackend: staticBackend{books: []catalog.Book{{ID: "one", Title: "One"}}}, version: 1}
	path := filepath.Join(t.TempDir(), "collections.json")
	handler, err := NewMulti(MultiConfig{AllowOpen: true, CollectionsState: path}, []Library{{ID: "books", Name: "Books", Store: backend}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/collections", strings.NewReader(`{"name":"Favorites","library_id":"books"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(created, createRequest)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", created.Code, created.Body.String())
	}

	backend.reads = 0
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/opds/collections", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("collections feed status %d: %s", response.Code, response.Body.String())
	}
	if backend.reads != 1 {
		t.Fatalf("library catalogue was cloned %d times, want 1", backend.reads)
	}
}
