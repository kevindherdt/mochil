package web

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/catalog"
)

func TestCollectionStoreRejectsMalformedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collections.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"accounts":{}} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newCollectionStore(path); err == nil {
		t.Fatal("collection state with trailing JSON was accepted")
	}
}

func TestCollectionStoreRollsBackFailedPersistence(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := newCollectionStore("")
	if err != nil {
		t.Fatal(err)
	}
	collection, err := store.create("alice", "Favorites", "books", nil)
	if err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "collections.json")
	member := collectionMember{Type: "book", LibraryID: "books", BookID: "one"}
	if err = store.member("alice", collection.ID, member, true); err == nil {
		t.Fatal("member write through an invalid state path succeeded")
	}
	if got := store.list("alice")[0].Members; len(got) != 0 {
		t.Fatalf("failed member write remained visible: %+v", got)
	}
	if err = store.remove("alice", collection.ID); err == nil {
		t.Fatal("delete through an invalid state path succeeded")
	}
	if got := store.list("alice"); len(got) != 1 {
		t.Fatalf("failed delete remained visible: %+v", got)
	}
}

func TestCollectionDynamicSeriesPersistenceAndNaturalOrder(t *testing.T) {
	backend := staticBackend{books: []catalog.Book{
		{ID: "10", Title: "Story Volume 10", Series: "Story"},
		{ID: "2", Title: "Story Volume 2", Series: "Story"},
	}}
	path := filepath.Join(t.TempDir(), "collections.json")
	handler, err := NewMulti(MultiConfig{AllowOpen: true, CollectionsState: path}, []Library{{ID: "books", Name: "Books", Store: backend}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, target, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, target, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if response := request(http.MethodPost, "/api/collections", `{"name":"Favorites","library_id":"books"} {}`); response.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON status = %d, want 400", response.Code)
	}
	oversized := `{"name":"` + strings.Repeat("x", maximumCollectionRequestBytes) + `","library_id":"books"}`
	if response := request(http.MethodPost, "/api/collections", oversized); response.Code != http.StatusBadRequest {
		t.Fatalf("oversized JSON status = %d, want 400", response.Code)
	}
	created := request(http.MethodPost, "/api/collections", `{"name":"Favorites","library_id":"books"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", created.Code, created.Body.String())
	}
	var collection savedCollection
	if err := json.Unmarshal(created.Body.Bytes(), &collection); err != nil {
		t.Fatal(err)
	}
	member := request(http.MethodPost, "/api/collections/"+collection.ID+"/members", `{"type":"series","library_id":"books","series":"Story"}`)
	if member.Code != http.StatusNoContent {
		t.Fatalf("member status %d: %s", member.Code, member.Body.String())
	}
	books := request(http.MethodGet, "/api/collections/"+collection.ID+"/books", "")
	if books.Code != http.StatusOK {
		t.Fatalf("books status %d", books.Code)
	}
	var result []bookView
	if err := json.Unmarshal(books.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0].Title != "Story Volume 2" || result[1].Title != "Story Volume 10" {
		t.Fatalf("unexpected order: %+v", result)
	}
	opds := request(http.MethodGet, "/opds/collections/"+collection.ID, "")
	if opds.Code != http.StatusOK || !bytes.Contains(opds.Body.Bytes(), []byte("Story Volume 2")) {
		t.Fatalf("collection OPDS unavailable: %d %s", opds.Code, opds.Body.String())
	}
	reloaded, err := newCollectionStore(path)
	if err != nil || len(reloaded.list("default")) != 1 {
		t.Fatalf("collection did not persist: %v", err)
	}
	if reloaded.list("default")[0].LibraryID != "books" {
		t.Fatalf("collection lost library scope: %+v", reloaded.list("default")[0])
	}
	filtered := request(http.MethodGet, "/api/collections?library=other", "")
	if filtered.Code != http.StatusOK || strings.TrimSpace(filtered.Body.String()) != "[]" {
		t.Fatalf("collection leaked across libraries: %s", filtered.Body.String())
	}
}
