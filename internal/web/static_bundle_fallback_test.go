package web

import "testing"

// A browser that kept the shell from the previous build asks for a bundle hash this one does not
// ship. The answer used to be 404, which is why the app came up completely unstyled after an update
// and stayed that way until a hard refresh.
func TestASupersededBundleGetsTheCurrentStylesheet(t *testing.T) {
	manifest := []byte(`{"script":"dist/index-aaa111.js","style":"dist/index-bbb222.css","styles":["app.css"],"preload":[]}`)
	got, ok := currentBundle("static/dist/index-000000.css", manifest)
	if !ok || got != "static/dist/index-bbb222.css" {
		t.Fatalf("want the current stylesheet, got %q (ok=%v)", got, ok)
	}
}

func TestASupersededBundleGetsTheCurrentScript(t *testing.T) {
	manifest := []byte(`{"script":"dist/index-aaa111.js","style":"dist/index-bbb222.css"}`)
	for _, ext := range []string{".js", ".mjs"} {
		got, ok := currentBundle("static/dist/chunk-old"+ext, manifest)
		if !ok || got != "static/dist/index-aaa111.js" {
			t.Fatalf("%s: want the current entry script, got %q (ok=%v)", ext, got, ok)
		}
	}
}

func TestOnlyHashedBundlesAreRedirected(t *testing.T) {
	manifest := []byte(`{"script":"dist/index-aaa111.js","style":"dist/index-bbb222.css"}`)
	cases := []string{
		"static/app.css",                   // source stylesheet, a real 404 if missing
		"static/dist/manifest.json",        // the manifest itself
		"static/dist/index-aaa111.css.map", // no same-kind replacement
		"static/index.html",
	}
	for _, name := range cases {
		if got, ok := currentBundle(name, manifest); ok {
			t.Fatalf("%q must not be answered with %q", name, got)
		}
	}
}

func TestNoManifestMeansNoFallback(t *testing.T) {
	if got, ok := currentBundle("static/dist/index-000000.css", nil); ok {
		t.Fatalf("without a manifest nothing can be substituted, got %q", got)
	}
	if got, ok := currentBundle("static/dist/index-000000.css", []byte("not json")); ok {
		t.Fatalf("a broken manifest must not produce a substitution, got %q", got)
	}
}

func TestAnEmptyManifestStaysQuiet(t *testing.T) {
	if _, ok := currentBundle("static/dist/index-000000.css", []byte(`{"script":"","style":""}`)); ok {
		t.Fatal("an empty manifest has no current bundle to offer")
	}
}
