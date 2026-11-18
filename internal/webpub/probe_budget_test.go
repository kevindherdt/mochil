package webpub

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Every spine item is inflated to test whether it is an image-only page, so the
// total has to be bounded: a book with thousands of large chapters must not cost
// gigabytes per manifest build.
func TestBuildBoundsTheImageProbeAcrossTheSpine(t *testing.T) {
	// An image-only page needs at least minimumImages images to be split.
	var images strings.Builder
	for n := 1; n <= 8; n++ {
		images.WriteString(fmt.Sprintf(`<img src="page%d.jpg"/>`, n))
	}
	chapter := "<html><body>" + images.String() + "</body></html>" + strings.Repeat(" ", 256<<10)
	files := map[string]string{"META-INF/container.xml": container}
	for n := 1; n <= 8; n++ {
		files[fmt.Sprintf("OEBPS/page%d.jpg", n)] = "jpeg"
	}
	spine := make([]string, 0, 200)
	manifest := make([]string, 0, 200)
	// 200 chapters of ~256 KiB each is about 50 MiB of spine bodies, three times the
	// probe budget, so a bounded build cannot split all of them.
	for i := 0; i < 200; i++ {
		name := fmt.Sprintf("c%03d.xhtml", i)
		files["OEBPS/"+name] = chapter
		id := fmt.Sprintf("i%03d", i)
		manifest = append(manifest, `<item id="`+id+`" href="`+name+`" media-type="application/xhtml+xml"/>`)
		spine = append(spine, `<itemref idref="`+id+`"/>`)
	}
	files["OEBPS/content.opf"] = `<package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Probe</dc:title><dc:identifier>urn:x</dc:identifier></metadata><manifest>` + strings.Join(manifest, "") + `</manifest><spine>` + strings.Join(spine, "") + `</spine></package>`

	data := buildEPUB(t, files)
	built, err := Build(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(built.pages) == 0 {
		t.Fatal("no image pages were generated at all")
	}
	if len(built.pages) >= 200*8 {
		t.Fatalf("pages = %d: every chapter was inflated, so the probe is unbounded", len(built.pages))
	}
}
