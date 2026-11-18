package document

import (
	"os"
	"path/filepath"
	"testing"
)

// codeVault writes one file per entry into a fresh vault and returns the store.
func codeVault(t *testing.T, files map[string]string) *Store {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	return store
}

func backlinkPaths(t *testing.T, store *Store, page string) []string {
	t.Helper()
	for _, item := range store.Items() {
		if item.Path != page {
			continue
		}
		items, err := store.Backlinks(item)
		if err != nil {
			t.Fatal(err)
		}
		paths := make([]string, 0, len(items))
		for _, linked := range items {
			paths = append(paths, linked.Path)
		}
		return paths
	}
	t.Fatalf("no page %q in the vault", page)
	return nil
}

// A [[link]] written inside a fenced block or inline code is example text, not
// a link: Obsidian does not list it as a linked mention, and neither should the
// backlink panel or the graph.
func TestBacklinksIgnoreLinksInsideCode(t *testing.T) {
	store := codeVault(t, map[string]string{
		"Home.md":    "Welcome.",
		"Example.md": "# Example\n\n```go\nlink := \"[[Home]]\"\n```\n\nNothing else.\n",
		"Sidebar.md": "Use `[[Home]]` or `[[Home|alias]]` as the syntax.",
		"Real.md":    "See [[Home]] and a real mention after `code`.\n",
	})
	got := backlinkPaths(t, store, "Home.md")
	if len(got) != 1 || got[0] != "Real.md" {
		t.Fatalf("backlinks = %v, want [Real.md]: only the real link is a mention", got)
	}
}

// `oops [[broken then [[Good]]` is a link to Good with an unmatched opener before
// it; scanning must resynchronise on the inner opener instead of swallowing Good.
func TestBacklinksResyncAtNestedOpener(t *testing.T) {
	store := codeVault(t, map[string]string{
		"Good.md":   "Target.",
		"Broken.md": "oops [[broken then [[Good]]\n",
		"Real.md":   "See [[Good]].\n",
	})
	got := backlinkPaths(t, store, "Good.md")
	if len(got) != 2 {
		t.Fatalf("backlinks = %v, want Broken.md and Real.md", got)
	}
}
