package web

import (
	"strings"
	"testing"
)

const sampleIndex = `<html>
  <head>
    <link rel="stylesheet" href="/assets/app.css?v=3" />
    <link rel="stylesheet" href="/assets/theme.css?v=2" />
    <link rel="stylesheet" href="/assets/other.css" />
  </head>
  <body><script type="module" src="/assets/app.js?v=150"></script></body>
</html>`

const sampleManifest = `{"script":"dist/app-AAA.js","style":"dist/core-bbb.css","styles":["app.css","theme.css"],"preload":["dist/chunks/c-1.js"]}`

func TestBundledIndexSwapsInHashedAssets(t *testing.T) {
	page := string(bundledIndex([]byte(sampleIndex), []byte(sampleManifest)))
	for _, want := range []string{`href="/assets/dist/core-bbb.css"`, `src="/assets/dist/app-AAA.js"`, `rel="modulepreload" href="/assets/dist/chunks/c-1.js"`, `href="/assets/other.css"`} {
		if !strings.Contains(page, want) {
			t.Fatalf("missing %q in\n%s", want, page)
		}
	}
	for _, gone := range []string{"app.css", "theme.css", "app.js?v=150"} {
		if strings.Contains(page, gone) {
			t.Fatalf("%q should have been replaced", gone)
		}
	}
	if strings.Count(page, "dist/core-bbb.css") != 1 {
		t.Fatal("bundled stylesheet must be linked exactly once")
	}
}

func TestBundledIndexFallsBackWithoutAUsableManifest(t *testing.T) {
	for _, manifest := range []string{"", "not json", `{"script":"","style":""}`} {
		if got := string(bundledIndex([]byte(sampleIndex), []byte(manifest))); got != sampleIndex {
			t.Fatalf("manifest %q altered the page", manifest)
		}
	}
	if got := string(bundledIndex([]byte("<html></html>"), []byte(sampleManifest))); got != "<html></html>" {
		t.Fatal("a page without bundled links must stay unchanged")
	}
}
