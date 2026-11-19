package catalog

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Standard Ebooks writes <dc:creator id="author"> into the OPF and declares the dc namespace on the
// metadata element rather than on the package element. A real Standard Ebooks EPUB reported every
// author as "Unknown" even though its description parsed, so this pins the exact shape.
func TestStandardEbooksCreatorIsRead(t *testing.T) {
	opf := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid" dir="ltr">
	<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
		<dc:identifier id="uid">https://standardebooks.org/ebooks/bram-stoker/dracula</dc:identifier>
		<dc:date>2014-05-25T00:00:00Z</dc:date>
		<dc:title id="title">Dracula</dc:title>
		<meta property="file-as" refines="#title">Dracula</meta>
		<dc:creator id="author">Bram Stoker</dc:creator>
		<meta property="file-as" refines="#author">Stoker, Bram</meta>
		<meta property="role" refines="#author" scheme="marc:relators">aut</meta>
		<meta property="role" refines="#author" scheme="marc:relators">dto</meta>
		<meta property="role" refines="#author" scheme="marc:relators">wpr</meta>
		<dc:contributor id="translator">A Translator</dc:contributor>
		<dc:creator id="translator2">Someone Else</dc:creator>
		<meta property="role" refines="#translator2" scheme="marc:relators">trl</meta>
		<dc:language>en-GB</dc:language>
		<dc:description><p>Count Dracula awaits.</p></dc:description>
	</metadata>
	<manifest>
		<item id="chapter" href="text/chapter-1.xhtml" media-type="application/xhtml+xml"/>
	</manifest>
	<spine><itemref idref="chapter"/></spine>
</package>`

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	write := func(name, body string) {
		w, err := zipWriter.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	write("mimetype", "application/epub+zip")
	write("META-INF/container.xml", `<?xml version="1.0" encoding="utf-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	<rootfiles><rootfile full-path="epub/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`)
	write("epub/content.opf", opf)
	write("epub/text/chapter-1.xhtml", `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head><body><p>Text.</p></body></html>`)
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "dracula.epub")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	book, err := readEPUB(path, "dracula.epub", info)
	if err != nil {
		t.Fatalf("readEPUB: %v", err)
	}
	if len(book.Authors) != 1 || book.Authors[0] != "Bram Stoker" {
		t.Fatalf("want the creator from the OPF, got %#v", book.Authors)
	}
	if len(book.AuthorSort) != 1 || book.AuthorSort[0] != "Stoker, Bram" {
		t.Fatalf("want the refined file-as, got %#v", book.AuthorSort)
	}
}
