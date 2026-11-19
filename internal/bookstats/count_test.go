package bookstats

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
)

func epub(t *testing.T, files map[string]string) *bytes.Reader {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, body := range files {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte(body))
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buffer.Bytes())
}

const container = `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`

func TestAnEPUBIsCountedFromTheTextOfItsReadingOrderOnly(t *testing.T) {
	book := epub(t, map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf":      `<package><manifest><item id="a" href="one.xhtml"/><item id="b" href="Two%20Part.xhtml"/><item id="x" href="unused.xhtml"/></manifest><spine><itemref idref="a"/><itemref idref="b"/></spine></package>`,
		"OEBPS/one.xhtml":        `<html><head><title>Ignored title words</title><style>p{color:red}</style></head><body><h1>Chapter One</h1><p>It was a dark &amp; stormy night.</p><script>var skipped = 1;</script></body></html>`,
		"OEBPS/Two Part.xhtml":   `<html><body><p>The end — of the <em>second</em> chapter.</p><p>   </p></body></html>`,
		"OEBPS/unused.xhtml":     `<html><body><p>Not in the spine at all, so not counted.</p></body></html>`,
	})
	stats, err := Count(book, "EPUB", 0)
	if err != nil {
		t.Fatal(err)
	}
	// "Chapter One" is 2 words; "It was a dark & stormy night." is 6 (a bare & is not a word); "The end — of the second
	// chapter." is 6 (so is a bare dash); the head, style, script and unused chapter add nothing.
	if stats.Words != 14 {
		t.Fatalf("words = %d, want 14", stats.Words)
	}
	if stats.Pages != 1 {
		t.Fatalf("pages = %d, want 1 for 14 words", stats.Pages)
	}
}

func TestPagesAreEstimatedAtTwoHundredFiftyWordsAndRoundUp(t *testing.T) {
	for words, want := range map[int]int{0: 0, 1: 1, 250: 1, 251: 2, 500: 2, 501: 3} {
		if got := pagesFor(words); got != want {
			t.Errorf("%d words = %d pages, want %d", words, got, want)
		}
	}
}

func TestAComicKnowsItsPagesAndAnUnknownFormatIsRefused(t *testing.T) {
	stats, err := Count(bytes.NewReader(nil), "cbz", 182)
	if err != nil || stats.Pages != 182 || stats.Words != 0 {
		t.Fatalf("cbz = %+v err=%v", stats, err)
	}
	if _, err := Count(bytes.NewReader(nil), "mobi", 0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
	if _, err := Count(strings.NewReader("not a zip at all"), "epub", 0); err == nil {
		t.Fatal("a broken EPUB must be an error, not zero words")
	}
}
