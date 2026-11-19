package webpub

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func readAlongEPUB(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	add := func(name, body string, method uint16) {
		header := &zip.FileHeader{Name: name, Method: method}
		w, err := archive.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(w, body)
	}
	add("META-INF/container.xml", `<container><rootfiles><rootfile full-path="OEBPS/book.opf"/></rootfiles></container>`, zip.Deflate)
	add("OEBPS/book.opf", `<package><manifest>
<item id="c1" href="c1.html" media-type="application/xhtml+xml" media-overlay="o1"/>
<item id="c2" href="c2.html" media-type="application/xhtml+xml"/>
<item id="o1" href="smil/c1.smil" media-type="application/smil+xml"/>
</manifest><spine><itemref idref="c1"/><itemref idref="c2"/></spine></package>`, zip.Deflate)
	add("OEBPS/c1.html", "<p id='a'>One</p>", zip.Deflate)
	add("OEBPS/c2.html", "<p>Two</p>", zip.Deflate)
	add("OEBPS/smil/c1.smil", `<smil><body><seq>
<par id="p1"><text src="../c1.html#a"/><audio src="../Audio/one.mp3" clipBegin="0.000s" clipEnd="1.500s"/></par>
<par id="p2"><text src="../c1.html#b"/><audio src="../Audio/one.mp3" clipBegin="0:00:01.5" clipEnd="2500ms"/></par>
<par id="p3"><text src="../c1.html#c"/><audio src="../Audio/missing.mp3" clipBegin="0s" clipEnd="1s"/></par>
</seq></body></smil>`, zip.Deflate)
	add("OEBPS/Audio/one.mp3", "0123456789", zip.Store)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestOverlayListsNarratedClipsPerChapter(t *testing.T) {
	data := readAlongEPUB(t)
	overlay, err := BuildOverlay(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(overlay.Chapters) != 1 || overlay.Chapters[0].Href != "OEBPS/c1.html" {
		t.Fatalf("chapters = %+v", overlay.Chapters)
	}
	clips := overlay.Chapters[0].Clips
	if len(clips) != 2 {
		t.Fatalf("clips = %v, want the clip with missing audio dropped", clips)
	}
	if clips[0][0] != "a" || clips[0][3] != 1.5 || clips[1][2] != 1.5 || clips[1][3] != 2.5 {
		t.Fatalf("clips = %v", clips)
	}
	if len(overlay.Audio) != 1 || overlay.Audio[0] != "OEBPS/Audio/one.mp3" {
		t.Fatalf("audio = %v", overlay.Audio)
	}
}

func TestStoredEntriesAreSeekableAndCompressedOnesAreNot(t *testing.T) {
	data := readAlongEPUB(t)
	seeker, size, ok := OpenSeekable(bytes.NewReader(data), int64(len(data)), "OEBPS/Audio/one.mp3")
	if !ok || size != 10 {
		t.Fatalf("seekable = %v size %d", ok, size)
	}
	_, _ = seeker.Seek(4, io.SeekStart)
	rest, _ := io.ReadAll(seeker)
	if string(rest) != "456789" {
		t.Fatalf("read %q", rest)
	}
	if _, _, ok := OpenSeekable(bytes.NewReader(data), int64(len(data)), "OEBPS/c1.html"); ok {
		t.Fatal("a deflated entry cannot be sliced in place")
	}
}

func TestClockValues(t *testing.T) {
	for in, want := range map[string]float64{"12.5s": 12.5, "1500ms": 1.5, "0:01:02.5": 62.5, "2min": 120, "3": 3, "": 0} {
		if got := clock(in); got != want {
			t.Errorf("clock(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestMalformedOverlaysNeverPanic(t *testing.T) {
	for name, smil := range map[string]string{
		"truncated":  `<smil><body><seq><par id="x"><text src="../c1.html#a"/><audio src=`,
		"empty":      ``,
		"no clip":    `<smil><body><seq><par id="x"><text src="../c1.html#a"/></par></seq></body></smil>`,
		"backwards":  `<smil><body><par><text src="../c1.html#a"/><audio src="../Audio/one.mp3" clipBegin="5s" clipEnd="1s"/></par></body></smil>`,
		"escapes":    `<smil><body><par><text src="../../../../etc/passwd#a"/><audio src="../../../../etc/passwd" clipBegin="0s" clipEnd="1s"/></par></body></smil>`,
		"not a time": `<smil><body><par><text src="../c1.html#a"/><audio src="../Audio/one.mp3" clipBegin="x" clipEnd="y"/></par></body></smil>`,
	} {
		var buffer bytes.Buffer
		archive := zip.NewWriter(&buffer)
		for file, body := range map[string]string{
			"META-INF/container.xml": `<container><rootfiles><rootfile full-path="OEBPS/book.opf"/></rootfiles></container>`,
			"OEBPS/book.opf":         `<package><manifest><item id="c1" href="c1.html" media-overlay="o1"/><item id="o1" href="smil/c1.smil"/></manifest><spine><itemref idref="c1"/></spine></package>`,
			"OEBPS/c1.html":          "<p/>",
			"OEBPS/smil/c1.smil":     smil,
			"OEBPS/Audio/one.mp3":    "x",
		} {
			w, _ := archive.Create(file)
			_, _ = io.WriteString(w, body)
		}
		_ = archive.Close()
		overlay, err := BuildOverlay(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(overlay.Chapters) != 0 {
			t.Errorf("%s: expected no playable chapters, got %+v", name, overlay.Chapters)
		}
	}
}

func TestWriteTextOnlyDropsAudioAndKeepsTheRest(t *testing.T) {
	var source bytes.Buffer
	archive := zip.NewWriter(&source)
	add := func(name, body string, method uint16) {
		entry, _ := archive.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		_, _ = entry.Write([]byte(body))
	}
	add("mimetype", "application/epub+zip", zip.Store)
	add("META-INF/container.xml", `<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`, zip.Deflate)
	add("OEBPS/content.opf", `<package><manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/><item id="a1" href="../Audio/one.mp4" media-type="audio/mp4"/></manifest><spine><itemref idref="c1"/></spine></package>`, zip.Deflate)
	add("OEBPS/c1.xhtml", "<p>Hello</p>", zip.Deflate)
	add("Audio/one.mp4", strings.Repeat("sound", 1000), zip.Store)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	var lite bytes.Buffer
	skipped, err := WriteTextOnly(&lite, bytes.NewReader(source.Bytes()), int64(source.Len()))
	if err != nil || skipped != 1 {
		t.Fatalf("skipped=%d err=%v", skipped, err)
	}
	read, err := zip.NewReader(bytes.NewReader(lite.Bytes()), int64(lite.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, file := range read.File {
		names = append(names, file.Name)
	}
	if strings.Join(names, ",") != "mimetype,META-INF/container.xml,OEBPS/content.opf,OEBPS/c1.xhtml" {
		t.Fatalf("entries = %v", names)
	}
	chapter, _ := read.File[3].Open()
	text, _ := io.ReadAll(chapter)
	if string(text) != "<p>Hello</p>" || lite.Len() >= source.Len() {
		t.Fatalf("chapter=%q lite=%d source=%d", text, lite.Len(), source.Len())
	}
}

func TestWriteTextOnlyAlsoDropsAudioOnlyTheNarrationNames(t *testing.T) {
	data := readAlongEPUB(t) // its audio is not declared in the manifest
	var lite bytes.Buffer
	skipped, err := WriteTextOnly(&lite, bytes.NewReader(data), int64(len(data)))
	if err != nil || skipped != 1 {
		t.Fatalf("skipped=%d err=%v", skipped, err)
	}
	read, _ := zip.NewReader(bytes.NewReader(lite.Bytes()), int64(lite.Len()))
	for _, file := range read.File {
		if file.Name == "OEBPS/Audio/one.mp3" {
			t.Fatal("the narration audio was kept")
		}
	}
}
