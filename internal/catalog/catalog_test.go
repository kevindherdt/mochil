package catalog

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSortKeyAndBucket(t *testing.T) {
	tests := map[string]string{"The Expanse": "E", "A Fire": "F", "1984": "#", "mistborn": "M"}
	for input, want := range tests {
		if got := Bucket(input); got != want {
			t.Errorf("Bucket(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestCalibreStoreUsesSidecarMetadataCoverAndPreferredFormat(t *testing.T) {
	library := t.TempDir()
	bookDirectory := filepath.Join(library, "Brandon Sanderson", "The Final Empire (1)")
	if err := os.MkdirAll(bookDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	epubPath := filepath.Join(bookDirectory, "The Final Empire.epub")
	writeTestEPUB(t, epubPath)
	if err := os.WriteFile(filepath.Join(bookDirectory, "The Final Empire.pdf"), []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	metadata := `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Mistborn: The Final Empire</dc:title><dc:creator file-as="Sanderson, Brandon">Brandon Sanderson</dc:creator><dc:language>en</dc:language><meta name="calibre:series" content="Mistborn"/><meta name="calibre:series_index" content="1"/></metadata></package>`
	if err := os.WriteFile(filepath.Join(bookDirectory, "metadata.opf"), []byte(metadata), 0o644); err != nil {
		t.Fatal(err)
	}
	cover := []byte("calibre-cover")
	if err := os.WriteFile(filepath.Join(bookDirectory, "cover.jpg"), cover, 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewCalibreStore(library, filepath.Join(t.TempDir(), "calibre-cache.json"))
	stats, err := store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Books != 1 {
		t.Fatalf("expected one preferred format, got %+v", stats)
	}
	book := store.Books()[0]
	if book.Title != "Mistborn: The Final Empire" || book.Series != "Mistborn" || book.SeriesIndex == nil || *book.SeriesIndex != 1 || book.Format != "epub" {
		t.Fatalf("unexpected Calibre metadata: %+v", book)
	}
	data, err := store.Cover(context.Background(), book)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(cover) {
		t.Fatalf("cover = %q", data)
	}
	stats, err = store.Scan()
	if err != nil || stats.Reused != 1 {
		t.Fatalf("Calibre cache was not reused: %+v %v", stats, err)
	}
}

func TestNaturalLess(t *testing.T) {
	if !NaturalLess("Mushoku Tensei Volume 2", "Mushoku Tensei Volume 10") {
		t.Fatal("volume 2 should sort before volume 10")
	}
	if NaturalLess("Volume 10", "Volume 2") {
		t.Fatal("volume 10 should not sort before volume 2")
	}
}

func TestScanReadsMetadataAndReusesCache(t *testing.T) {
	library := t.TempDir()
	path := filepath.Join(library, "Mistborn", "01.epub")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestEPUB(t, path)

	store := NewStore(library, filepath.Join(t.TempDir(), "catalog.json"))
	stats, err := store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Books != 1 || stats.Read != 1 {
		t.Fatalf("unexpected first scan: %+v", stats)
	}
	book := store.Books()[0]
	if book.Title != "The Final Empire" || book.Series != "Mistborn" || book.SeriesIndex == nil || *book.SeriesIndex != 1 {
		t.Fatalf("unexpected metadata: %+v", book)
	}
	if len(book.AuthorSort) != 1 || book.AuthorSort[0] != "Sanderson, Brandon" {
		t.Fatalf("unexpected author sort: %+v", book.AuthorSort)
	}

	stats, err = store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Reused != 1 || stats.Read != 0 {
		t.Fatalf("cache was not reused: %+v", stats)
	}
}

func TestScanPrefersEPUB3SeriesAndAuthorRoles(t *testing.T) {
	library := t.TempDir()
	path := filepath.Join(library, "Modern.epub")
	writeTestEPUBFiles(t, path, map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OPS/package.opf"/></rootfiles></container>`,
		"OPS/package.opf": `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
			<dc:title>Modern Volume</dc:title><dc:creator id="author">A Writer</dc:creator><dc:creator id="artist">An Artist</dc:creator>
			<meta refines="#author" property="role">aut</meta><meta refines="#author" property="file-as">Writer, A</meta><meta refines="#artist" property="role">ill</meta>
			<meta id="series" property="belongs-to-collection">Modern Saga</meta><meta refines="#series" property="collection-type">series</meta><meta refines="#series" property="group-position">2.5</meta>
			<meta name="calibre:series" content="Legacy Saga"/><meta name="calibre:series_index" content="9"/>
		</metadata></package>`,
	})

	store := NewStore(library, filepath.Join(t.TempDir(), "catalog.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	book := store.Books()[0]
	if book.Series != "Modern Saga" || book.SeriesIndex == nil || *book.SeriesIndex != 2.5 {
		t.Fatalf("EPUB 3 collection did not override legacy metadata: %+v", book)
	}
	if len(book.Authors) != 1 || book.Authors[0] != "A Writer" || len(book.AuthorSort) != 1 || book.AuthorSort[0] != "Writer, A" {
		t.Fatalf("creator roles or refined sort name were not honored: %+v", book)
	}
}

func TestScanPublishesOnlyChangedCatalogueGenerations(t *testing.T) {
	library := t.TempDir()
	cache := filepath.Join(t.TempDir(), "catalog.json")
	writeTestEPUB(t, filepath.Join(library, "First.epub"))

	store := NewStore(library, cache)
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	firstVersion := store.CatalogVersion()
	if firstVersion == 0 {
		t.Fatal("initial scan did not publish a catalogue generation")
	}

	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	if got := store.CatalogVersion(); got != firstVersion {
		t.Fatalf("unchanged scan advanced generation from %d to %d", firstVersion, got)
	}

	writeTestEPUB(t, filepath.Join(library, "Second.epub"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	if got := store.CatalogVersion(); got <= firstVersion {
		t.Fatalf("changed scan left generation at %d after %d", got, firstVersion)
	}
	if got := len(store.Books()); got != 2 {
		t.Fatalf("published books = %d, want 2", got)
	}
}

func TestScanIgnoresHiddenTemporaryPublications(t *testing.T) {
	library := t.TempDir()
	visible := filepath.Join(library, "Visible.epub")
	hidden := filepath.Join(library, ".ebook-workbench-3901597878.epub")
	writeTestEPUB(t, visible)
	writeTestEPUB(t, hidden)

	store := NewStore(library, filepath.Join(t.TempDir(), "catalog.json"))
	stats, err := store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Books != 1 || len(store.Books()) != 1 || store.Books()[0].Path != "Visible.epub" {
		t.Fatalf("hidden temporary publication was indexed: stats=%+v books=%+v", stats, store.Books())
	}
}

func writeTestEPUB(t *testing.T, path string) {
	t.Helper()
	writeTestEPUBFiles(t, path, map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="content.opf"/></rootfiles></container>`,
		"content.opf":            `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>The Final Empire</dc:title><dc:creator file-as="Sanderson, Brandon">Brandon Sanderson</dc:creator><dc:language>en</dc:language><meta name="calibre:series" content="Mistborn"/><meta name="calibre:series_index" content="1"/></metadata></package>`,
	})
}

func writeTestEPUBFiles(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, contents := range files {
		entry, createErr := zw.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte(contents)); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestScanMarksEPUBsThatCarryTheirOwnRecording(t *testing.T) {
	library := t.TempDir()
	container := `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OPS/package.opf"/></rootfiles></container>`
	writeTestEPUBFiles(t, filepath.Join(library, "Narrated.epub"), map[string]string{
		"META-INF/container.xml": container,
		"OPS/package.opf": `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Narrated</dc:title></metadata>
			<manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml" media-overlay="o1"/><item id="o1" href="c1.smil" media-type="application/smil+xml"/></manifest></package>`,
	})
	writeTestEPUBFiles(t, filepath.Join(library, "Timed.epub"), map[string]string{
		"META-INF/container.xml": container,
		"OPS/package.opf":        `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Timed</dc:title><meta property="media:duration">1:02:03</meta></metadata></package>`,
	})
	writeTestEPUBFiles(t, filepath.Join(library, "Plain.epub"), map[string]string{
		"META-INF/container.xml": container,
		"OPS/package.opf": `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Plain</dc:title></metadata>
			<manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest></package>`,
	})
	store := NewStore(library, filepath.Join(t.TempDir(), "catalog.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, book := range store.Books() {
		got[book.Title] = book.ReadAlong
	}
	if !got["Narrated"] || !got["Timed"] || got["Plain"] || len(got) != 3 {
		t.Fatalf("read-along flags = %v", got)
	}
}

// A book whose series is its own title is a per-book folder that older scans mistook for a series.
func TestNormalizeSeriesDropsASeriesThatRepeatsTheTitle(t *testing.T) {
	index := 1.0
	books := []Book{
		{Title: "1984", Series: "1984", SeriesIndex: &index},
		{Title: "Dies the Fire", Series: "Emberverse", SeriesIndex: &index},
		{Title: "The Age of Ra", Series: "", SeriesIndex: &index},
	}
	normalizeSeries(books)
	if books[0].Series != "" || books[0].SeriesIndex != nil {
		t.Fatalf("a series that repeats the title should be dropped: %+v", books[0])
	}
	if books[1].Series != "Emberverse" || books[1].SeriesIndex == nil {
		t.Fatalf("a real series must survive: %+v", books[1])
	}
}
