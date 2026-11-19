package isbn

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

type file struct{ *bytes.Reader }

func (file) Close() error { return nil }

// book builds an EPUB. pages are in reading order; guide maps a page name to a guide type.
func book(t *testing.T, identifiers []string, pages [][2]string, guide map[string]string) file {
	t.Helper()
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	put := func(name, body string) {
		w, _ := archive.Create(name)
		_, _ = w.Write([]byte(body))
	}
	var manifest, spine, ids, refs strings.Builder
	for _, id := range identifiers {
		fmt.Fprintf(&ids, `<dc:identifier>%s</dc:identifier>`, id)
	}
	for i, page := range pages {
		fmt.Fprintf(&manifest, `<item id="p%d" href="%s"/>`, i, page[0])
		fmt.Fprintf(&spine, `<itemref idref="p%d"/>`, i)
		put("OEBPS/"+page[0], `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>x</title></head><body>`+page[1]+`</body></html>`)
	}
	for name, kind := range guide {
		fmt.Fprintf(&refs, `<reference type="%s" href="%s"/>`, kind, name)
	}
	put("META-INF/container.xml", `<container><rootfiles><rootfile full-path="OEBPS/c.opf"/></rootfiles></container>`)
	put("OEBPS/c.opf", `<package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata>`+ids.String()+`</metadata><manifest>`+manifest.String()+`</manifest><spine>`+spine.String()+`</spine><guide>`+refs.String()+`</guide></package>`)
	_ = archive.Close()
	return file{bytes.NewReader(out.Bytes())}
}

func volume(n float64) *float64 { return &n }

func TestNormalizeAcceptsOnlyCorrectISBNsAndConvertsTen(t *testing.T) {
	for in, want := range map[string]string{
		"978-0-306-40615-7": "9780306406157",
		"9780306406157":     "9780306406157",
		"0306406152":        "9780306406157",
		"0-8044-2957-X":     "9780804429573",
	} {
		if got, ok := Normalize(in); !ok || got != want {
			t.Errorf("Normalize(%q) = %q %v, want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"9780306406158", "0306406153", "123", "", "9770306406159", "abcdefghij"} {
		if got, ok := Normalize(bad); ok {
			t.Errorf("Normalize(%q) = %q, want it refused", bad, got)
		}
	}
}

func TestTheBooksOwnISBNBeatsTheOnesInAnAlsoByList(t *testing.T) {
	epub := book(t, nil, [][2]string{
		{"cover.xhtml", `<p>Cover</p>`},
		{"copyright.xhtml", `<p>Copyright 2021 Example Press</p><p>ISBN 978-0-306-40615-7 (ebook)</p><p>Series Vol. 3</p>`},
		{"ch1.xhtml", `<p>Once upon a time.</p>`},
		{"also.xhtml", `<h2>Also by this author</h2><p>Series Volume 4 ISBN 978-0-7432-7356-5</p><p>Series Volume 2 ISBN 978-0-8044-2957-3</p>`},
	}, map[string]string{"copyright.xhtml": "copyright-page"})
	result, err := Extract(epub, "epub", Hints{Title: "Series - Volume 03", Volume: volume(3)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Own != "9780306406157" || result.Ambiguous {
		t.Fatalf("own = %q ambiguous=%v: %+v", result.Own, result.Ambiguous, result.Candidates)
	}
	if len(result.Candidates) != 3 {
		t.Fatalf("the other volumes are listed too: %+v", result.Candidates)
	}
	for _, other := range result.Candidates[1:] {
		if !strings.Contains(strings.Join(other.Reasons, ";"), "also by") {
			t.Errorf("%s should say why it is not the book's own: %v", other.ISBN, other.Reasons)
		}
	}
	if !strings.Contains(strings.Join(result.Candidates[0].Reasons, ";"), "copyright page") {
		t.Errorf("the winner says why: %v", result.Candidates[0].Reasons)
	}
}

func TestAnISBNInTheMetadataIsTheBooksOwn(t *testing.T) {
	epub := book(t, []string{"urn:isbn:0306406152", "uuid:abc"}, [][2]string{{"a.xhtml", `<p>Hello</p>`}}, nil)
	result, _ := Extract(epub, "epub", Hints{})
	if result.Own != "9780306406157" {
		t.Fatalf("%+v", result)
	}
}

func TestACallThatIsTooCloseIsNotMadeButEveryCandidateIsListed(t *testing.T) {
	epub := book(t, nil, [][2]string{{"a.xhtml", `<p>ISBN 978-0-306-40615-7</p><p>ISBN 978-0-7432-7356-5</p>`}}, nil)
	result, _ := Extract(epub, "epub", Hints{})
	if result.Own != "" || !result.Ambiguous || len(result.Candidates) != 2 {
		t.Fatalf("%+v", result)
	}
}

func TestANumberWithABadCheckDigitIsNotAnISBN(t *testing.T) {
	epub := book(t, nil, [][2]string{{"a.xhtml", `<p>ISBN 978-0-306-40615-8 and order number 1234567890123</p>`}}, nil)
	result, _ := Extract(epub, "epub", Hints{})
	if len(result.Candidates) != 0 {
		t.Fatalf("%+v", result)
	}
}

func TestAnISBNRightAfterAYearIsStillFound(t *testing.T) {
	epub := book(t, nil, [][2]string{{"copyright.xhtml", `<p>First published 2021 9780306406157</p>`}}, nil)
	result, _ := Extract(epub, "epub", Hints{})
	if len(result.Candidates) != 1 || result.Candidates[0].ISBN != "9780306406157" {
		t.Fatalf("%+v", result)
	}
}

func TestALabelFarAwayGivesAnISBNNoCredit(t *testing.T) {
	epub := book(t, nil, [][2]string{{"a.xhtml", `<p>ISBN</p>` + strings.Repeat("<p>filler words here</p>", 40) + `<p>978-0-306-40615-7</p>`}}, nil)
	result, _ := Extract(epub, "epub", Hints{})
	if len(result.Candidates) != 1 {
		t.Fatalf("%+v", result)
	}
	for _, reason := range result.Candidates[0].Reasons {
		if strings.Contains(reason, "labelled ISBN") {
			t.Fatalf("the label is far away: %v", result.Candidates[0].Reasons)
		}
	}
}

func comic(t *testing.T, info string) file {
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	w, _ := archive.Create("ComicInfo.xml")
	_, _ = w.Write([]byte(info))
	_ = archive.Close()
	return file{bytes.NewReader(out.Bytes())}
}

func TestAComicsGTINIsItsISBN(t *testing.T) {
	result, err := Extract(comic(t, `<ComicInfo><Title>x</Title><GTIN>9780306406157</GTIN></ComicInfo>`), "cbz", Hints{})
	if err != nil || result.Own != "9780306406157" {
		t.Fatalf("%+v %v", result, err)
	}
	if result, _ := Extract(comic(t, `<ComicInfo><Title>x</Title></ComicInfo>`), "cbz", Hints{}); len(result.Candidates) != 0 {
		t.Fatalf("%+v", result)
	}
}

func TestOtherFormatsAreRefusedAndBrokenFilesReportAnError(t *testing.T) {
	if _, err := Extract(file{bytes.NewReader([]byte("x"))}, "pdf", Hints{}); err != ErrUnsupported {
		t.Fatalf("err = %v", err)
	}
	if _, err := Extract(file{bytes.NewReader([]byte("not a zip"))}, "epub", Hints{}); err == nil {
		t.Fatal("a file that is not an EPUB must be an error")
	}
}

func TestOnlyThePageAndTheListDecideWhenNothingElseIsLabelled(t *testing.T) {
	epub := book(t, nil, [][2]string{
		{"copyright.xhtml", `<p>978-0-306-40615-7</p>`},
		{"ch1.xhtml", `<p>Story.</p>`},
		{"end.xhtml", `<p>Also by this author</p><p>978-0-7432-7356-5</p>`},
	}, map[string]string{"copyright.xhtml": "copyright-page"})
	result, _ := Extract(epub, "epub", Hints{})
	if result.Own != "9780306406157" {
		t.Fatalf("the copyright page's number is the book's own: %+v", result)
	}
	last := result.Candidates[len(result.Candidates)-1]
	if last.ISBN != "9780743273565" || last.Score >= 0 {
		t.Fatalf("the also-by number ranks below zero: %+v", last)
	}
}

func TestAPageNamedCopyrightCountsEvenWithoutAGuide(t *testing.T) {
	epub := book(t, nil, [][2]string{
		{"copyright.xhtml", `<p>978-0-306-40615-7</p>`},
		{"ch9.xhtml", `<p>978-0-7432-7356-5</p>`},
	}, nil)
	if result, _ := Extract(epub, "epub", Hints{}); result.Own != "9780306406157" {
		t.Fatalf("%+v", result)
	}
}

func TestACatalogueEntryThatMarksEachNumberWithItsVolumeIsRead(t *testing.T) {
	epub := book(t, nil, [][2]string{{"copyright.xhtml", `<p>Identifiers: ISBN 9780316558570 (v1 : pbk) | ISBN 9780316559911 (v2 : pbk) | ISBN 9780316559966 (v3 : pbk)</p>`}}, nil)
	result, _ := Extract(epub, "epub", Hints{Volume: volume(2)})
	if result.Own != "9780316559911" {
		t.Fatalf("volume 2's number: %+v", result)
	}
}

func TestALabelBelongsToTheNumberNextToItNotItsNeighbour(t *testing.T) {
	epub := book(t, nil, [][2]string{{"copyright.xhtml", `<p>978-0-306-40615-7 (paperback)</p><p>978-0-7432-7356-5 (ebook)</p>`}}, nil)
	result, _ := Extract(epub, "epub", Hints{})
	if result.Own != "9780743273565" {
		t.Fatalf("the ebook's number: %+v", result)
	}
	if reasons := strings.Join(result.Candidates[1].Reasons, ";"); !strings.Contains(reasons, "print edition") || strings.Contains(reasons, "ebook") {
		t.Fatalf("the paperback number must not borrow the ebook label: %v", reasons)
	}
}
