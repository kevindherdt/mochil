package catalog

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCBZComicInfoAndNaturalPageOrder(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "Series", "Volume 2.cbz")
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	entries := []struct{ name, contents string }{
		{"ComicInfo.xml", `<ComicInfo><Title>Second Volume</Title><Series>A Series</Series><Number>2</Number><Writer>A Writer</Writer><Summary>Summary</Summary></ComicInfo>`},
		{"pages/page10.jpg", "ten"},
		{"pages/page2.jpg", "two"},
		{"pages/page1.jpg", "one"},
	}
	for _, item := range entries {
		entry, createErr := zw.Create(item.name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte(item.contents)); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	store := NewStore(dir, filepath.Join(t.TempDir(), "cache.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	book := store.Books()[0]
	if book.Format != "cbz" || book.PageCount != 3 || book.SeriesIndex == nil || *book.SeriesIndex != 2 {
		t.Fatalf("unexpected CBZ metadata: %+v", book)
	}
	page, err := store.Page(context.Background(), book, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer page.Reader.Close()
	data, err := io.ReadAll(page.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "two" {
		t.Fatalf("natural page order returned %q", data)
	}
}
