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

const epub3Package = `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="bookid">urn:uuid:11111111-2222-3333-4444-555555555555</dc:identifier>
    <dc:title id="t1">Old Title</dc:title>
    <meta refines="#t1" property="title-type">main</meta>
    <dc:creator id="c1">Old Author</dc:creator>
    <meta refines="#c1" property="role" scheme="marc:relators">aut</meta>
    <dc:language>en</dc:language>
    <meta property="belongs-to-collection" id="col1">Old Series</meta>
    <meta refines="#col1" property="collection-type">series</meta>
    <meta refines="#col1" property="group-position">9</meta>
    <meta property="dcterms:modified">2020-01-01T00:00:00Z</meta>
  </metadata>
  <manifest><item id="c" href="c.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="c"/></spine>
</package>`

func buildEPUB(t *testing.T, path, packageText string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	stored, _ := writer.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	io.WriteString(stored, "application/epub+zip")
	for name, body := range map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`,
		"OEBPS/content.opf":      packageText,
		"OEBPS/c.xhtml":          "<html><body>chapter one</body></html>",
	} {
		w, _ := writer.Create(name)
		io.WriteString(w, body)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func zipText(t *testing.T, path, name string) string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, entry := range reader.File {
		if entry.Name == name {
			rc, _ := entry.Open()
			defer rc.Close()
			raw, _ := io.ReadAll(rc)
			return string(raw)
		}
	}
	t.Fatalf("%s missing from %s", name, path)
	return ""
}

func TestEmbeddingReplacesTheFieldsAndKeepsEverythingElse(t *testing.T) {
	executor, _, managed, _ := newTestExecutor(t, 1<<20)
	book := filepath.Join(managed, "Series", "Series - Vol 01.epub")
	buildEPUB(t, book, epub3Package)
	index := 2.0
	meta := opf.Metadata{Title: "New & Improved", Authors: []string{"First Author", "Second Author"}, Series: "New Series", SeriesIndex: &index, Tags: []string{"fantasy", "cosy"},
		Publisher: "Pub", Published: "2021-05-01", Description: "A <b>story</b>.", Rating: 8, TitleSort: "New and Improved", Identifiers: map[string]string{"hardcover": "42"}}
	if err := executor.EmbedMetadata("Series/Series - Vol 01.epub", meta); err != nil {
		t.Fatal(err)
	}
	got, err := ReadEmbedded(book)
	if err != nil || got.Title != "New & Improved" || strings.Join(got.Authors, "|") != "First Author|Second Author" || got.Series != "New Series" || got.SeriesIndex == nil || *got.SeriesIndex != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	pkg := zipText(t, book, "OEBPS/content.opf")
	for _, gone := range []string{"Old Title", "Old Author", "Old Series", "title-type", `refines="#c1"`, "group-position", "collection-type", "belongs-to-collection"} {
		if strings.Contains(pkg, gone) {
			t.Errorf("%q should be gone:\n%s", gone, pkg)
		}
	}
	for _, kept := range []string{`unique-identifier="bookid"`, "urn:uuid:11111111", "<dc:language>en</dc:language>", "dcterms:modified", "<manifest>", `<meta name="calibre:rating" content="8"/>`, `calibre:title_sort`, "<dc:subject>fantasy</dc:subject>", "A &lt;b&gt;story&lt;/b&gt;.", `opf:scheme="HARDCOVER"`, `xmlns:opf=`} {
		if !strings.Contains(pkg, kept) {
			t.Errorf("%q should be there:\n%s", kept, pkg)
		}
	}
	if zipText(t, book, "OEBPS/c.xhtml") != "<html><body>chapter one</body></html>" {
		t.Fatal("a chapter changed")
	}
	reader, _ := zip.OpenReader(book)
	defer reader.Close()
	if reader.File[0].Name != "mimetype" || reader.File[0].Method != zip.Store {
		t.Fatalf("mimetype must stay first and stored: %s %d", reader.File[0].Name, reader.File[0].Method)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(managed, "Series", ".bobobooks-embed-*")); len(leftovers) != 0 {
		t.Fatalf("temporary files left behind: %v", leftovers)
	}
	// embedding again does not pile up duplicates
	if err := executor.EmbedMetadata("Series/Series - Vol 01.epub", meta); err != nil {
		t.Fatal(err)
	}
	again := zipText(t, book, "OEBPS/content.opf")
	if strings.Count(again, "<dc:title>") != 1 || strings.Count(again, "calibre:series\"") != 1 || strings.Count(again, "HARDCOVER") != 1 || strings.Count(again, "<dc:creator") != 2 {
		t.Fatalf("a second embed duplicated fields:\n%s", again)
	}
}

func TestEmptyFieldsAreLeftAloneAndABadBookIsNotTouched(t *testing.T) {
	executor, _, managed, _ := newTestExecutor(t, 1<<20)
	book := filepath.Join(managed, "a.epub")
	buildEPUB(t, book, epub3Package)
	if err := executor.EmbedMetadata("a.epub", opf.Metadata{Title: "Only Title"}); err != nil {
		t.Fatal(err)
	}
	got, _ := ReadEmbedded(book)
	if got.Title != "Only Title" || strings.Join(got.Authors, "") != "Old Author" || got.Series != "Old Series" {
		t.Fatalf("fields without a value must be left alone: %+v", got)
	}

	noDC := filepath.Join(managed, "b.epub")
	buildEPUB(t, noDC, `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata></metadata></package>`)
	before, _ := os.ReadFile(noDC)
	if err := executor.EmbedMetadata("b.epub", opf.Metadata{Title: "X"}); err == nil {
		t.Fatal("a package without Dublin Core was changed")
	}
	if after, _ := os.ReadFile(noDC); !bytes.Equal(before, after) {
		t.Fatal("a refused embed still changed the file")
	}
	os.WriteFile(filepath.Join(managed, "c.epub"), []byte("not a zip"), 0o644)
	for name, path := range map[string]string{"not an epub": "c.epub", "pdf": "x.pdf", "escape": "../x.epub", "journal": ".ingest/a.epub", "missing": "nope.epub"} {
		if err := executor.EmbedMetadata(path, opf.Metadata{Title: "X"}); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if err := executor.EmbedMetadata("a.epub", opf.Metadata{Title: "  "}); err == nil {
		t.Fatal("a blank title was accepted")
	}
}

func TestAnEmbeddedLinkOutsideTheManagedLibraryIsRefused(t *testing.T) {
	executor, _, managed, _ := newTestExecutor(t, 1<<20)
	outside := filepath.Join(t.TempDir(), "outside.epub")
	buildEPUB(t, outside, epub3Package)
	if err := os.Symlink(outside, filepath.Join(managed, "link.epub")); err != nil {
		t.Skip("symlinks unavailable")
	}
	before, _ := os.ReadFile(outside)
	if err := executor.EmbedMetadata("link.epub", opf.Metadata{Title: "X"}); err == nil {
		t.Fatal("a link out of the library was followed")
	}
	if after, _ := os.ReadFile(outside); !bytes.Equal(before, after) {
		t.Fatal("a file outside the library changed")
	}
}
