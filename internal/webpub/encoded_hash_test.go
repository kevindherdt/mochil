package webpub

import (
	"bytes"
	"testing"
)

// A chapter whose file name contains "#" must be percent-encoded in the OPF,
// because a literal "#" there starts a fragment. Unescaping before splitting the
// fragment turned "chapter#1.xhtml" into "chapter" and lost the chapter.
func TestBuildKeepsAnEncodedHashInAChapterName(t *testing.T) {
	data := buildEPUB(t, map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf": `<package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:title>Hashed</dc:title><dc:identifier>urn:x</dc:identifier></metadata>
<manifest><item id="c1" href="text/chapter%231.xhtml" media-type="application/xhtml+xml"/>
<item id="c2" href="text/after.xhtml" media-type="application/xhtml+xml"/></manifest>
<spine><itemref idref="c1"/><itemref idref="c2"/></spine></package>`,
		"OEBPS/text/chapter#1.xhtml": "<html><body><p>one</p></body></html>",
		"OEBPS/text/after.xhtml":     "<html><body><p>two</p></body></html>",
	})
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.ReadingOrder) != 2 {
		t.Fatalf("reading order: %+v", manifest.ReadingOrder)
	}
	if got := manifest.ReadingOrder[0].Href; got != "OEBPS/text/chapter#1.xhtml" {
		t.Fatalf("href = %q, want %q: an encoded # belongs to the file name", got, "OEBPS/text/chapter#1.xhtml")
	}
	if size := manifest.sizes["OEBPS/text/chapter#1.xhtml"]; size == 0 {
		t.Fatalf("chapter was not matched to its archive entry; sizes = %v", manifest.sizes)
	}
}

// The overlay resolves the chapter and audio hrefs again, so it must agree with
// Build about an encoded "#": otherwise a narration clip is dropped (its audio
// never matches an archive entry) and the chapter href no longer matches the
// reading order the client looks it up in.
func TestOverlayResolvesEncodedHashesInChapterAndAudioNames(t *testing.T) {
	data := buildEPUB(t, map[string]string{
		"META-INF/container.xml": container,
		"OEBPS/content.opf": `<package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:title>Hashed</dc:title><dc:identifier>urn:x</dc:identifier></metadata>
<manifest><item id="c1" href="text/chapter%231.xhtml" media-type="application/xhtml+xml" media-overlay="o1"/>
<item id="o1" href="text/overlay.smil" media-type="application/smil+xml"/>
<item id="a1" href="audio/track%232.mp3" media-type="audio/mpeg"/></manifest>
<spine><itemref idref="c1"/></spine></package>`,
		"OEBPS/text/chapter#1.xhtml": "<html><body><p id=\"p1\">one</p></body></html>",
		"OEBPS/text/overlay.smil": `<smil><body><seq><par id="x"><text src="chapter%231.xhtml#p1"/>` +
			`<audio src="../audio/track%232.mp3" clipBegin="0s" clipEnd="2s"/></par></seq></body></smil>`,
		"OEBPS/audio/track#2.mp3": "0123456789",
	})
	overlay, err := BuildOverlay(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(overlay.Chapters) != 1 {
		t.Fatalf("chapters = %+v, want the narrated chapter", overlay.Chapters)
	}
	if got := overlay.Chapters[0].Href; got != "OEBPS/text/chapter#1.xhtml" {
		t.Fatalf("chapter href = %q, want %q: an encoded # belongs to the file name", got, "OEBPS/text/chapter#1.xhtml")
	}
	if clips := overlay.Chapters[0].Clips; len(clips) != 1 || clips[0][0] != "p1" {
		t.Fatalf("clips = %v, want the clip that plays track#2.mp3", clips)
	}
	if len(overlay.Audio) != 1 || overlay.Audio[0] != "OEBPS/audio/track#2.mp3" {
		t.Fatalf("audio = %v", overlay.Audio)
	}
	manifest, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.ReadingOrder) != 1 || manifest.ReadingOrder[0].Href != overlay.Chapters[0].Href {
		t.Fatalf("reading order = %+v, overlay href = %q: the two must agree", manifest.ReadingOrder, overlay.Chapters[0].Href)
	}
}
