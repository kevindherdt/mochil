package document

import (
	"os"
	"path/filepath"
	"testing"
)

// A note saved as .markdown is a Markdown note everywhere else in the store, so
// its [[links]] must show up as backlinks too.
func TestBacklinksIncludeMarkdownExtensionNotes(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		"Home.md":           "Welcome.",
		"Other.md":          "See [[Home]].",
		"Longform.markdown": "Also [[Home]].",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	var home Item
	for _, item := range store.Items() {
		if item.Path == "Home.md" {
			home = item
		}
	}
	backlinks, err := store.Backlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backlinks) != 2 {
		t.Fatalf("backlinks = %v, want both notes that link to Home", backlinks)
	}
}
