package document

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTextIndexSearchesContentAndReusesPersistentRevisions(t *testing.T) {
	root, cache := t.TempDir(), filepath.Join(t.TempDir(), "documents.json")
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("A quiet jellyfish waits for a prince in the moonlight."), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.pdf"), []byte("jellyfish"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data.json"), []byte(`{"creature":"jellyfish"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestDOCX(t, filepath.Join(root, "report.docx"), `<w:document xmlns:w="urn:word"><w:body><w:p><w:r><w:t>Quarterly jellyfish report</w:t></w:r></w:p></w:body></w:document>`)
	store := NewStore("notes", root, cache)
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	matches, coverage, err := store.SearchText("jellyfish")
	if err != nil || len(matches) != 3 || coverage.Indexed != 3 || coverage.Eligible != 3 || !coverage.Complete {
		t.Fatalf("search = %#v, %#v, %v", matches, coverage, err)
	}
	for _, snippet := range matches {
		if !strings.Contains(snippet, "jellyfish") {
			t.Fatalf("snippet = %q", snippet)
		}
	}
	reloaded := NewStore("notes", root, cache)
	if _, err := reloaded.Scan(); err != nil {
		t.Fatal(err)
	}
	matches, coverage, err = reloaded.SearchText("moonlight")
	if err != nil || len(matches) != 1 || !coverage.Complete {
		t.Fatalf("persistent search = %#v, %#v, %v", matches, coverage, err)
	}
}

func TestVaultBacklinksUseExactWikiTargetsAndRefreshWithTheStore(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"Home.md":          "# Home",
		"Linked.md":        "See [[Home#Start|the beginning]] and [[Home]].",
		"Path linked.md":   "See [[Projects/Home]].",
		"Plain mention.md": "Home is useful, but this is not a wiki link.",
		"Projects/Home.md": "# Other Home",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	var home, nested Item
	for _, item := range store.Items() {
		switch item.Path {
		case "Home.md":
			home = item
		case "Projects/Home.md":
			nested = item
		}
	}
	backlinks, err := store.Backlinks(home)
	if err != nil || len(backlinks) != 1 || backlinks[0].Path != "Linked.md" {
		t.Fatalf("home backlinks = %#v, %v", backlinks, err)
	}
	backlinks, err = store.Backlinks(nested)
	if err != nil || len(backlinks) != 1 || backlinks[0].Path != "Path linked.md" {
		t.Fatalf("nested backlinks = %#v, %v", backlinks, err)
	}
	if err = os.WriteFile(filepath.Join(root, "Linked.md"), []byte("No link now."), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Scan(); err != nil {
		t.Fatal(err)
	}
	backlinks, err = store.Backlinks(home)
	if err != nil || len(backlinks) != 0 {
		t.Fatalf("refreshed backlinks = %#v, %v", backlinks, err)
	}
}

func TestSearchSnippetSurvivesCaseFoldingThatChangesLength(t *testing.T) {
	root := t.TempDir()
	// "ẞ" (3 bytes) lowercases to "ß" (2 bytes), so a lowercased offset no
	// longer points at the same character in the original text.
	body := strings.Repeat("ẞ", 1000) + " needle " + strings.Repeat("z", 320)
	if err := os.WriteFile(filepath.Join(root, "sharp.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	matches, _, err := store.SearchText("needle")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("matches = %#v", matches)
	}
	for _, snippet := range matches {
		if !strings.Contains(snippet, "needle") {
			t.Fatalf("snippet = %q, want the matched word", snippet)
		}
	}
}
