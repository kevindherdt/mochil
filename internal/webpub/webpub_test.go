package webpub

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func buildEPUB(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(w, body)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

const container = `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`

func TestBuildReadsMetadataSpineResourcesAndNestedNavigation(t *testing.T) {
	data := buildEPUB(t, map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf": `<package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:title>Neuromancer</dc:title><dc:creator>William Gibson</dc:creator><dc:language>en</dc:language><dc:identifier>urn:x</dc:identifier></metadata>
<manifest><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
<item id="c1" href="text/c%201.xhtml" media-type="application/xhtml+xml"/><item id="c2" href="text/c2.xhtml" media-type="application/xhtml+xml"/>
<item id="css" href="style.css" media-type="text/css"/><item id="cover" href="cover.jpg" media-type="image/jpeg" properties="cover-image"/></manifest>
<spine page-progression-direction="ltr"><itemref idref="c1"/><itemref idref="c2"/></spine></package>`,
		"OEBPS/nav.xhtml": `<html xmlns:epub="http://www.idpf.org/2007/ops"><body><nav epub:type="toc"><ol>
<li><a href="text/c%201.xhtml">One</a><ol><li><a href="text/c%201.xhtml#s1">Section</a></li></ol></li>
<li><a href="text/c2.xhtml">Two</a></li></ol></nav></body></html>`,
		"OEBPS/text/c 1.xhtml": "<html/>", "OEBPS/text/c2.xhtml": "<html/>", "OEBPS/style.css": "", "OEBPS/cover.jpg": "x",
	})
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Metadata.Title != "Neuromancer" || manifest.Metadata.Author[0] != "William Gibson" || manifest.Metadata.ReadingProgress != "ltr" {
		t.Fatalf("metadata: %+v", manifest.Metadata)
	}
	if len(manifest.ReadingOrder) != 2 || manifest.ReadingOrder[0].Href != "OEBPS/text/c 1.xhtml" {
		t.Fatalf("reading order: %+v", manifest.ReadingOrder)
	}
	var cover bool
	for _, link := range manifest.Resources {
		if link.Href == "OEBPS/cover.jpg" && len(link.Rel) == 1 && link.Rel[0] == "cover" {
			cover = true
		}
		if link.Href == "OEBPS/text/c2.xhtml" {
			t.Fatal("spine items must not repeat in resources")
		}
	}
	if !cover {
		t.Fatalf("cover missing: %+v", manifest.Resources)
	}
	if len(manifest.TOC) != 2 || manifest.TOC[0].Title != "One" || len(manifest.TOC[0].Children) != 1 ||
		manifest.TOC[0].Children[0].Href != "OEBPS/text/c 1.xhtml#s1" || manifest.TOC[1].Href != "OEBPS/text/c2.xhtml" {
		t.Fatalf("toc: %+v", manifest.TOC)
	}
}

func TestBuildFallsBackToNCXAndRejectsEmptySpine(t *testing.T) {
	data := buildEPUB(t, map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf":      `<package><metadata><title>T</title></metadata><manifest><item id="n" href="toc.ncx" media-type="application/x-dtbncx+xml"/><item id="a" href="a.html" media-type="application/xhtml+xml"/></manifest><spine toc="n"><itemref idref="a"/></spine></package>`,
		"OEBPS/toc.ncx":          `<ncx><navMap><navPoint><navLabel><text>Start</text></navLabel><content src="a.html"/></navPoint></navMap></ncx>`,
		"OEBPS/a.html":           "<html/>",
	})
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(manifest.TOC) != 1 || manifest.TOC[0].Title != "Start" {
		t.Fatalf("ncx: %v %+v", err, manifest)
	}
	empty := buildEPUB(t, map[string]string{"META-INF/container.xml": container, "OEBPS/content.opf": `<package><manifest/><spine/></package>`})
	if _, err := Build(bytes.NewReader(empty), int64(len(empty))); err == nil {
		t.Fatal("an EPUB without a spine must be rejected")
	}
}

func TestOpenReturnsEntriesAndNeverEscapes(t *testing.T) {
	data := buildEPUB(t, map[string]string{"a/b.txt": "hello"})
	handle, size, ok := Open(bytes.NewReader(data), int64(len(data)), "/a/b.txt")
	if !ok || size != 5 {
		t.Fatal("entry not found")
	}
	handle.Close()
	if _, _, ok := Open(bytes.NewReader(data), int64(len(data)), "../a/b.txt"); ok {
		t.Fatal("parent traversal must not resolve")
	}
}

func TestPositionsSpanTheBookInOrder(t *testing.T) {
	data := buildEPUB(t, map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf":      `<package><metadata><title>T</title></metadata><manifest><item id="a" href="a.html" media-type="application/xhtml+xml"/><item id="b" href="b.html" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="a"/><itemref idref="b"/></spine></package>`,
		"OEBPS/a.html":           string(make([]byte, 2500)), "OEBPS/b.html": "tiny",
	})
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	list, _ := manifest.Positions()
	if list.Total != 4 {
		t.Fatalf("want 3 positions in a.html and 1 in b.html, got %d", list.Total)
	}
	last := list.Positions[3]["locations"].(map[string]any)
	if list.Positions[3]["href"] != "OEBPS/b.html" || last["position"].(int64) != 4 || last["totalProgression"].(float64) != 0.75 {
		t.Fatalf("last position: %+v", list.Positions[3])
	}
}

func TestImageOnlyDocumentsSplitIntoOnePagePerImage(t *testing.T) {
	var body, images = "<html><head><title>x</title></head><body>", map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf":      `<package><metadata><title>Scan</title></metadata><manifest><item id="i" href="index.html" media-type="application/xhtml+xml"/><item id="n" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/></manifest><spine><itemref idref="i"/></spine></package>`,
		"OEBPS/nav.xhtml":        `<nav epub:type="toc"><ol><li><a href="index.html#page_3">Three</a></li></ol></nav>`,
	}
	for n := 1; n <= 10; n++ {
		name := "img " + string(rune('a'+n)) + ".jpg"
		body += `<p id="page_` + string(rune('0'+n%10)) + `"><img src="img%20` + string(rune('a'+n)) + `.jpg"/></p>`
		images["OEBPS/"+name] = "x"
	}
	images["OEBPS/index.html"] = body + "</body></html>"
	data := buildEPUB(t, images)
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.ReadingOrder) != 10 || !IsVirtual(manifest.ReadingOrder[0].Href) {
		t.Fatalf("want 10 generated pages: %+v", manifest.ReadingOrder)
	}
	page, ok := manifest.VirtualPage(manifest.ReadingOrder[2].Href)
	if !ok || !strings.Contains(page, `src="../../OEBPS/img%20d.jpg"`) {
		t.Fatalf("page: %v %s", ok, page)
	}
	if manifest.TOC[0].Href != manifest.ReadingOrder[0].Href {
		t.Fatalf("navigation must follow the split document: %+v", manifest.TOC)
	}
	if list, _ := manifest.Positions(); list.Total != 10 {
		t.Fatal("each generated page is one position")
	}
}

func TestDocumentsWithTextAreNeverSplit(t *testing.T) {
	body := "<html><body><p>" + strings.Repeat("A real paragraph of prose. ", 20) + "</p>"
	files := map[string]string{"META-INF/container.xml": container,
		"OEBPS/content.opf": `<package><metadata><title>T</title></metadata><manifest><item id="i" href="c.html" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="i"/></spine></package>`}
	for n := 0; n < 10; n++ {
		body += `<img src="i.jpg"/>`
	}
	files["OEBPS/c.html"], files["OEBPS/i.jpg"] = body+"</body></html>", "x"
	data := buildEPUB(t, files)
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(manifest.ReadingOrder) != 1 || IsVirtual(manifest.ReadingOrder[0].Href) {
		t.Fatalf("prose with figures must stay one chapter: %v %+v", err, manifest.ReadingOrder)
	}
}

func TestPositionsRefuseDeclaredSizesBeyondTheBudget(t *testing.T) {
	manifest := &Manifest{
		ReadingOrder: []Link{{Href: "a.xhtml"}, {Href: "b.xhtml"}},
		sizes:        map[string]int64{"a.xhtml": 1 << 62, "b.xhtml": -5},
	}
	if _, err := manifest.Positions(); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("an absurd declared size must be refused, got %v", err)
	}
	many := &Manifest{sizes: map[string]int64{}}
	for i := 0; i < 20000; i++ {
		href := fmt.Sprintf("c%d.xhtml", i)
		many.ReadingOrder = append(many.ReadingOrder, Link{Href: href})
		many.sizes[href] = 8 << 20
	}
	if _, err := many.Positions(); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("a total beyond the budget must be refused, got %v", err)
	}
	small := &Manifest{ReadingOrder: []Link{{Href: "a.xhtml"}}, sizes: map[string]int64{"a.xhtml": -5}}
	if list, err := small.Positions(); err != nil || list.Total != 1 {
		t.Fatalf("a bogus negative size still gives one position: %d %v", list.Total, err)
	}
}

func TestCommentedImagesAreNotPages(t *testing.T) {
	body, files := "<html><head><title>x</title></head><body>", map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf":      `<package><metadata><title>C</title></metadata><manifest><item id="i" href="index.html" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="i"/></spine></package>`,
	}
	for n := 0; n < 8; n++ {
		name := string(rune('a' + n))
		body += `<p><img src="i` + name + `.jpg"/></p>`
		files["OEBPS/i"+name+".jpg"] = "x"
	}
	body += `<!-- <img src="ghost.jpg"/> --></body></html>`
	files["OEBPS/ghost.jpg"] = "x"
	files["OEBPS/index.html"] = body
	data := buildEPUB(t, files)
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.ReadingOrder) != 8 {
		t.Fatalf("pages = %d, want 8 (a commented-out image is not a page)", len(manifest.ReadingOrder))
	}
}

func TestRepeatedSpineItemsReuseOneSplit(t *testing.T) {
	body, files := "<html><body>", map[string]string{
		"META-INF/container.xml": container,
	}
	for n := 0; n < 8; n++ {
		name := string(rune('a' + n))
		body += `<img src="p` + name + `.jpg"/>`
		files["OEBPS/p"+name+".jpg"] = "x"
	}
	files["OEBPS/index.html"] = body + "</body></html>"
	spine := ""
	for n := 0; n < 4; n++ {
		spine += `<itemref idref="i"/>`
	}
	files["OEBPS/content.opf"] = `<package><metadata><title>R</title></metadata><manifest><item id="i" href="index.html" media-type="application/xhtml+xml"/></manifest><spine>` + spine + `</spine></package>`
	data := buildEPUB(t, files)
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.ReadingOrder) != 8 {
		t.Fatalf("pages = %d, want 8 (a repeated spine item must not be split again)", len(manifest.ReadingOrder))
	}
	seen := map[string]bool{}
	for _, link := range manifest.ReadingOrder {
		if seen[link.Href] && IsVirtual(link.Href) {
			t.Fatalf("generated page %q appears twice; the split must be reused", link.Href)
		}
		seen[link.Href] = true
	}
}

func TestBuildKeepsInlineMarkupInsideATOCLabel(t *testing.T) {
	data := buildEPUB(t, map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf": `<package><metadata><title>T</title></metadata><manifest>` +
			`<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>` +
			`<item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest>` +
			`<spine><itemref idref="c1"/></spine></package>`,
		"OEBPS/nav.xhtml": `<html xmlns:epub="http://www.idpf.org/2007/ops"><body><nav epub:type="toc"><ol>` +
			`<li><a href="c1.xhtml">Chapter <span>One</span> continued</a></li></ol></nav></body></html>`,
		"OEBPS/c1.xhtml": "<html/>",
	})
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.TOC) != 1 {
		t.Fatalf("toc: %+v", manifest.TOC)
	}
	if got := manifest.TOC[0].Title; got != "Chapter One continued" {
		t.Fatalf("a label's text after nested markup was dropped: title = %q", got)
	}
}

func TestBuildKeepsFragmentOnlyTOCLinks(t *testing.T) {
	data := buildEPUB(t, map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf": `<package><metadata><title>T</title></metadata><manifest>` +
			`<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>` +
			`<item id="c1" href="nav.xhtml" media-type="application/xhtml+xml"/></manifest>` +
			`<spine><itemref idref="c1"/></spine></package>`,
		"OEBPS/nav.xhtml": `<html xmlns:epub="http://www.idpf.org/2007/ops"><body><nav epub:type="toc"><ol>` +
			`<li><a href="#part1">Part One</a></li></ol></nav></body></html>`,
	})
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.TOC) != 1 || manifest.TOC[0].Href != "OEBPS/nav.xhtml#part1" {
		t.Fatalf("a fragment-only link points inside the navigating document and must survive: toc: %+v", manifest.TOC)
	}
}
