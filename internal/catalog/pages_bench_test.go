package catalog

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeBenchmarkCBZ(tb testing.TB, pageCount int) string {
	tb.Helper()
	dir := tb.TempDir()
	path := filepath.Join(dir, "comic.cbz")
	file, err := os.Create(path)
	if err != nil {
		tb.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for index := 0; index < pageCount; index++ {
		entry, createErr := writer.Create(fmt.Sprintf("pages/page%d.jpg", index))
		if createErr != nil {
			tb.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte("page")); writeErr != nil {
			tb.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		tb.Fatal(err)
	}
	if err := file.Close(); err != nil {
		tb.Fatal(err)
	}
	return path
}

// BenchmarkCBZPageStreaming measures one mid-archive page request against a
// thousand-page comic: the index work every page pays before it can stream.
func BenchmarkCBZPageStreaming(b *testing.B) {
	path := writeBenchmarkCBZ(b, 1000)
	store := NewStore(filepath.Dir(path), filepath.Join(b.TempDir(), "cache.json"))
	book := Book{ID: "comic", Path: "comic.cbz", Format: "cbz", PageCount: 1000}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		page, err := store.Page(context.Background(), book, 500)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, page.Reader); err != nil {
			b.Fatal(err)
		}
		if err := page.Reader.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
