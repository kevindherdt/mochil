package readpos

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func makeEPUB(t *testing.T, chapters ...string) *Book {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	add := func(name, content string) {
		w, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(content))
	}
	add("META-INF/container.xml", `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`)
	var manifest, spine strings.Builder
	for i, chapter := range chapters {
		manifest.WriteString(fmt.Sprintf(`<item id="c%d" href="text/c%d.xhtml" media-type="application/xhtml+xml"/>`, i, i))
		spine.WriteString(fmt.Sprintf(`<itemref idref="c%d"/>`, i))
		add(fmt.Sprintf("OEBPS/text/c%d.xhtml", i), chapter)
	}
	add("OEBPS/content.opf", `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0"><manifest>`+manifest.String()+`</manifest><spine>`+spine.String()+`</spine></package>`)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	book, err := Open(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return book
}

func xhtml(body string) string {
	return `<?xml version="1.0" encoding="utf-8"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>t</title><style>p{}</style></head><body>` + body + `</body></html>`
}

const sentence = "The lantern swung over the harbour while the old captain counted every rope twice"

func TestXPointerRoundTripsThroughTextAnchor(t *testing.T) {
	chapter := xhtml(`
  <div class="c">
    <p>Opening paragraph that is long enough to be unique in this chapter alone.</p>
    <p>` + sentence + `, <em>muttering</em> to himself about the weather and the tide.</p>
    <p>Closing paragraph after the interesting one, again with plenty of words to read.</p>
  </div>`)
	book := makeEPUB(t, xhtml(`<p>Another chapter entirely with its own distinct and unrelated prose here.</p>`), chapter)
	for _, original := range []string{
		"/body/DocFragment[2]/body/div/p[2]/text()[1].0",
		"/body/DocFragment[2]/body/div/p[2]/text().0",
		"/body/DocFragment[2]/body/div/p[2]/text().4",
		"/body/DocFragment[2]/body/div/p[3]/text().0",
		"/body/DocFragment[2]/body/div/p[1]/text().9",
	} {
		anchor, level, err := book.AnchorFromXPointer(original)
		if err != nil || level != QualityText {
			t.Fatalf("%s -> %+v %v %v", original, anchor, level, err)
		}
		if anchor.SpineIndex != 1 || anchor.Exact == "" {
			t.Fatalf("%s anchor = %+v", original, anchor)
		}
		restored, level, err := book.XPointerFromAnchor(anchor)
		if err != nil || level != QualityText {
			t.Fatalf("%s anchor %+v -> %q %v %v", original, anchor, restored, level, err)
		}
		again, _, err := book.AnchorFromXPointer(restored)
		if err != nil || again.Exact != anchor.Exact || again.SpineIndex != anchor.SpineIndex {
			t.Fatalf("%s restored %s changes the anchor: %+v vs %+v", original, restored, again, anchor)
		}
	}
}

func TestAnchorStartsAtWordBoundaryAndCarriesContext(t *testing.T) {
	book := makeEPUB(t, xhtml(`<p>`+sentence+` and then continued onwards for quite some time afterwards, until dusk.</p>`))
	anchor, level, err := book.AnchorFromXPointer("/body/DocFragment[1]/body/p/text().7")
	if err != nil || level != QualityText {
		t.Fatalf("%+v %v %v", anchor, level, err)
	}
	if !strings.HasPrefix(anchor.Exact, "lantern swung") && !strings.HasPrefix(anchor.Exact, "The lantern") {
		t.Fatalf("anchor did not snap to a word start: %q", anchor.Exact)
	}
	if len(strings.Fields(anchor.Exact)) != exactWords || anchor.Suffix == "" {
		t.Fatalf("anchor shape = %+v", anchor)
	}
}

func TestInlineElementsDoNotBreakRoundTrip(t *testing.T) {
	book := makeEPUB(t, xhtml(`<p>Before the <b>bold</b> and <i>italic</i> words, the narrator keeps talking for a long while so the quote is unique.</p><p>Second paragraph, <span>entirely wrapped in a span element</span> yet still readable text goes here.</p>`))
	for _, original := range []string{
		"/body/DocFragment[1]/body/p[1]/text()[3].1",
		"/body/DocFragment[1]/body/p[2]/span/text().0",
		"/body/DocFragment[1]/body/p[1]/b/text().0",
	} {
		anchor, level, err := book.AnchorFromXPointer(original)
		if err != nil || level != QualityText {
			t.Fatalf("%s -> %+v %v %v", original, anchor, level, err)
		}
		restored, _, err := book.XPointerFromAnchor(anchor)
		if err != nil {
			t.Fatalf("%s: %v", original, err)
		}
		again, _, _ := book.AnchorFromXPointer(restored)
		if again.Exact != anchor.Exact {
			t.Fatalf("%s -> %s changed anchor %q vs %q", original, restored, again.Exact, anchor.Exact)
		}
	}
}

func TestMinifiedChapterAndEntities(t *testing.T) {
	book := makeEPUB(t, xhtml(`<h1>Title&nbsp;Page</h1><p>Caf&eacute; owners &amp; patrons gathered round the fire to hear the very long tale told.</p><p>Next paragraph begins here and runs on and on for a considerable stretch of text.</p>`))
	anchor, level, err := book.AnchorFromXPointer("/body/DocFragment[1]/body/p[2]/text().0")
	if err != nil || level != QualityText {
		t.Fatalf("%+v %v %v", anchor, level, err)
	}
	restored, _, err := book.XPointerFromAnchor(anchor)
	if err != nil || restored != "/body/DocFragment[1]/body/p[2]/text().0" {
		t.Fatalf("restored = %q %v", restored, err)
	}
}

func TestRepeatedSentenceResolvesNearReaderProgress(t *testing.T) {
	repeated := `<p>` + sentence + ` and nothing else changed for a very long time indeed.</p>`
	filler := func(n int) string {
		return xhtml(strings.Repeat(fmt.Sprintf("<p>Chapter %d filler text, unrelated to everything else in the story so far.</p>", n), 40))
	}
	book := makeEPUB(t, xhtml(repeated), filler(1), filler(2), xhtml(repeated))
	late, _, err := book.AnchorFromXPointer("/body/DocFragment[4]/body/p/text().0")
	if err != nil {
		t.Fatal(err)
	}
	xpointer, _, err := book.XPointerFromAnchor(late)
	if err != nil || !strings.HasPrefix(xpointer, "/body/DocFragment[4]/") {
		t.Fatalf("late repeat resolved to %q %v", xpointer, err)
	}
	early := late
	early.SpineIndex, early.Progress = 0, 0
	xpointer, _, err = book.XPointerFromAnchor(early)
	if err != nil || !strings.HasPrefix(xpointer, "/body/DocFragment[1]/") {
		t.Fatalf("early repeat resolved to %q %v", xpointer, err)
	}
}

func TestMissingTextIsNotGuessed(t *testing.T) {
	book := makeEPUB(t, xhtml(`<p>Nothing resembling the requested words appears anywhere in this short chapter.</p>`))
	_, _, err := book.XPointerFromAnchor(Anchor{SpineIndex: 0, Exact: "completely different words that never occur in the publication at all"})
	if err != ErrNotFound {
		t.Fatalf("err = %v", err)
	}
	if _, _, err = book.XPointerFromAnchor(Anchor{SpineIndex: 9, Exact: sentence}); err != ErrInvalid {
		t.Fatalf("out-of-range spine err = %v", err)
	}
}

func TestUnresolvableStructureDegradesToChapterOrElement(t *testing.T) {
	book := makeEPUB(t, xhtml(`<div><p>A reasonably long paragraph that will be the nearest resolvable element here.</p></div>`))
	anchor, level, err := book.AnchorFromXPointer("/body/DocFragment[1]/body/div/section[4]/p[9]/text().3")
	if err != nil || level != QualityApproximate || anchor.Exact == "" {
		t.Fatalf("partial structure: %+v %v %v", anchor, level, err)
	}
	anchor, level, err = book.AnchorFromXPointer("/body/DocFragment[1]")
	if err != nil || level != QualityChapter || anchor.Exact != "" || anchor.SpineIndex != 0 {
		t.Fatalf("fragment only: %+v %v %v", anchor, level, err)
	}
	for _, bad := range []string{"", "/body/DocFragment[0]/body", "/body/DocFragment[7]/body/p", "garbage", "/body/DocFragment[1]/body/p/text()/b"} {
		if _, _, err := book.AnchorFromXPointer(bad); err != ErrInvalid {
			t.Fatalf("%q err = %v", bad, err)
		}
	}
}

func TestParseXPointerForms(t *testing.T) {
	for value, want := range map[string]xpointer{
		"/body/DocFragment[3]/body/div[2]/p/text()[2].15": {fragment: 3, steps: []xpStep{{"body", 1}, {"div", 2}, {"p", 1}}, textIndex: 2, offset: 15},
		"/body/DocFragment/body/p[4]":                     {fragment: 1, steps: []xpStep{{"body", 1}, {"p", 4}}, offset: -1},
		"/body/DocFragment[2]/body/p.12":                  {fragment: 2, steps: []xpStep{{"body", 1}, {"p", 1}}, textIndex: 1, offset: 12},
	} {
		got, ok := parseXPointer(value)
		if !ok || fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("%s = %+v %v, want %+v", value, got, ok, want)
		}
	}
}
