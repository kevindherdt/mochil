package document

import (
	"os"
	"path/filepath"
	"testing"
)

// A rebuild that started before the index moved on must not answer from the
// previous reverse index: the newly indexed note would be missing from the
// backlink list until something else rebuilt it.
func TestBacklinksRebuildsWhenTheIndexMovesDuringTheBuild(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("Home.md", "Welcome.")
	write("Linked.md", "See [[Home]].")
	store := NewStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	index := store.text
	if backlinks, err := index.Backlinks("Home"); err != nil || len(backlinks) != 1 {
		t.Fatalf("backlinks before = %v, %v", backlinks, err)
	}

	// The index moves once before the request, so the rebuild has to run...
	write("Late.md", "Also [[Home]].")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	// ...and again while that rebuild is in flight, which is what a watcher does.
	once := true
	backlinkBuildHook = func() {
		if !once {
			return
		}
		once = false
		write("During.md", "And [[Home]].")
		if _, err := store.Scan(); err != nil {
			t.Errorf("scan: %v", err)
		}
		if err := index.ensure(); err != nil {
			t.Errorf("index rebuild: %v", err)
		}
	}
	defer func() { backlinkBuildHook = nil }()

	backlinks, err := index.Backlinks("Home")
	if err != nil {
		t.Fatal(err)
	}
	if len(backlinks) != 3 {
		t.Fatalf("backlinks = %v, want all three notes that link to Home", backlinks)
	}
}
