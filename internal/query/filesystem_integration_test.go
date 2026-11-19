package query_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/catalog"
	"mochil/internal/query"
)

func TestWarmIndexRefreshesAfterFilesystemScan(t *testing.T) {
	library := t.TempDir()
	store := catalog.NewStore(library, filepath.Join(t.TempDir(), "catalog.json"))
	writeCBZ(t, filepath.Join(library, "First.cbz"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}

	service := query.New([]query.Library{{ID: "books", Name: "Books", Store: store}})
	if got := service.Books(query.BooksRequest{LibraryID: "books"}); got.Total != 1 {
		t.Fatalf("initial warm index contains %d books, want 1", got.Total)
	}

	writeCBZ(t, filepath.Join(library, "Second.cbz"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	got := service.Books(query.BooksRequest{LibraryID: "books"})
	if got.Total != 2 || len(got.Items) != 2 || got.Items[1].Book.Title != "Second" {
		t.Fatalf("warm index did not observe published scan: %+v", got)
	}
}

func writeCBZ(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	page, err := archive.Create("page.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := page.Write([]byte("image")); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
