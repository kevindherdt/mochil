package ingest

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/opf"
)

// A small archive can carry an entry that expands to gigabytes. The rewrite used
// to decompress and recompress every entry just to copy it, so one crafted EPUB
// cost the ingest worker that much CPU and memory. Unchanged entries are now
// copied in their stored (compressed) form.
func TestEmbeddingDoesNotExpandUnchangedEntries(t *testing.T) {
	executor, _, managed, _ := newTestExecutor(t, 1<<20)
	book := filepath.Join(managed, "Series", "Series - Vol 01.epub")

	big := bytes.Repeat([]byte("chapter text, highly compressible. "), 300_000) // ~11 MB expanded
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	stored, _ := writer.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	io.WriteString(stored, "application/epub+zip")
	for name, body := range map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":      []byte(epub3Package),
		"OEBPS/big.xhtml":        big,
	} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(book), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(book, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(book)
	if err != nil {
		t.Fatal(err)
	}

	if err := executor.EmbedMetadata("Series/Series - Vol 01.epub", opf.Metadata{Title: "Expanded"}); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(book)
	if err != nil {
		t.Fatal(err)
	}
	// Recompressing an 11 MB entry would have produced roughly the same small file,
	// so the point is that it stayed small AND the bytes are still exact.
	if after.Size() > before.Size()*2 {
		t.Fatalf("the rewrite grew the archive from %d to %d bytes", before.Size(), after.Size())
	}
	reader, err := zip.OpenReader(book)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, entry := range reader.File {
		if entry.Name != "OEBPS/big.xhtml" {
			continue
		}
		if entry.UncompressedSize64 != uint64(len(big)) {
			t.Fatalf("the entry now claims %d bytes, want %d", entry.UncompressedSize64, len(big))
		}
		stream, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		restored, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(restored, big) {
			t.Fatalf("the copied entry changed: %d bytes, want %d", len(restored), len(big))
		}
		if !strings.HasPrefix(string(restored[:32]), "chapter text") {
			t.Fatal("the copied entry does not start with its original text")
		}
		return
	}
	t.Fatal("the large entry was dropped by the rewrite")
}
