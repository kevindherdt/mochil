package web

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/catalog"
)

func TestABookWithoutItsOwnCoverUsesTheImageBesideIt(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "Author", "Book")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestEPUB(t, filepath.Join(directory, "One.epub"), `<p>Words.</p>`)
	picture := image.NewRGBA(image.Rect(0, 0, 40, 60))
	for x := 0; x < 40; x++ {
		for y := 0; y < 60; y++ {
			picture.Set(x, y, color.RGBA{200, 40, 40, 255})
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, picture, nil); err != nil {
		t.Fatal(err)
	}
	store := catalog.NewFilesystemStore(root, filepath.Join(t.TempDir(), "cache.json"), catalog.FilesystemOptions{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	book := store.Books()[0]
	request := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/covers/books/"+book.ID, nil))
		return w
	}
	if w := request(); w.Code != http.StatusNotFound {
		t.Fatalf("with no image anywhere the cover must be missing, got %d", w.Code)
	}
	if err := os.WriteFile(filepath.Join(directory, "cover.jpg"), encoded.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if w := request(); w.Code != http.StatusOK || w.Body.Len() == 0 {
		t.Fatalf("the image beside the book should be the cover, got %d with %d bytes", w.Code, w.Body.Len())
	}
}
