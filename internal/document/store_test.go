package document

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIdentifyAcceptsInitialFormatsAndRejectsUnsafePaths(t *testing.T) {
	now := time.Unix(100, 0)
	for _, name := range []string{"Notes/Hello.md", "Guide.markdown", "plain.txt", "manual.PDF", "report.DOCX", "data.JSON", "table.csv", "config.yaml", "book.toml", "paper.tex", "events.log"} {
		item, ok := identify("notes", name, 10, now, false)
		if !ok || item.ID == "" || item.Revision == "" {
			t.Fatalf("identify(%q) = %#v, %v", name, item, ok)
		}
	}
	for _, name := range []string{"../secret.md", "/absolute.txt", "book.epub", "cover.png", "._resource.pdf", ".obsidian/state.json", "Folder/.private.md"} {
		if _, ok := identify("notes", name, 10, now, false); ok {
			t.Fatalf("identify(%q) accepted unsafe or unsupported path", name)
		}
	}
}

func TestVaultImagesAreIndexedButNeverTreatedAsText(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Assets", "cover.png"), []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewVaultStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	items := store.Items()
	if len(items) != 1 {
		t.Fatalf("vault image items = %#v", items)
	}
	item, ok := items[0], true
	if !ok || item.Format != "png" || item.MediaType != "image/png" || TextReadable(item.Format) {
		t.Fatalf("vault image = %#v, %v", item, ok)
	}
}

func TestStorePublishesAndReusesReadOnlySnapshot(t *testing.T) {
	root, cache := t.TempDir(), filepath.Join(t.TempDir(), "documents.json")
	if err := os.WriteFile(filepath.Join(root, "Second.txt"), []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "First.md"), []byte("# one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignore.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("notes", root, cache)
	stats, err := store.Scan()
	if err != nil || stats.Items != 2 || store.Items()[0].Title != "First" {
		t.Fatalf("scan = %#v, %v, items=%#v", stats, err, store.Items())
	}
	version := store.Version()
	stats, err = store.Scan()
	if err != nil || stats.Reused != 2 || store.Version() <= version {
		t.Fatalf("rescan = %#v, %v, version=%d", stats, err, store.Version())
	}
	file, item, err := store.Open(store.Items()[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	if item.Title != "First" {
		t.Fatalf("opened %#v", item)
	}
}

func TestStoreRetainsSnapshotWhenSourceIsOffline(t *testing.T) {
	parent, cache := t.TempDir(), filepath.Join(t.TempDir(), "documents.json")
	root := filepath.Join(parent, "notes")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("note"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("notes", root, cache)
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, root+".offline"); err != nil {
		t.Fatal(err)
	}
	stats, err := store.Scan()
	if err != nil || stats.Reused != 1 || stats.Errors != 1 || len(store.Items()) != 1 {
		t.Fatalf("offline scan = %#v, %v, items=%#v", stats, err, store.Items())
	}
	reloaded := NewStore("notes", root, cache)
	stats, err = reloaded.Scan()
	if err != nil || stats.Reused != 1 || len(reloaded.Items()) != 1 {
		t.Fatalf("cached startup = %#v, %v", stats, err)
	}
}

func TestStoreSkipsSymlinks(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	secret := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "linked.md")); err != nil {
		t.Fatal(err)
	}
	store := NewStore("notes", root, "")
	if stats, err := store.Scan(); err != nil || stats.Items != 0 {
		t.Fatalf("symlink scan = %#v, %v", stats, err)
	}
}

func TestStoreSkipsHiddenVaultInternals(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian", "plugins"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".obsidian", "plugins", "state.json"), []byte("secret settings"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".private.md"), []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Welcome.md"), []byte("# Welcome"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("vault", root, "")
	stats, err := store.Scan()
	if err != nil || stats.Items != 1 || store.Items()[0].Path != "Welcome.md" {
		t.Fatalf("vault scan = %#v, %v, items=%#v", stats, err, store.Items())
	}
}

func TestStorePagesByStableTitleAndRecentIndexes(t *testing.T) {
	root := t.TempDir()
	for index, name := range []string{"Bravo.txt", "Alpha.txt", "Charlie.txt"} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		modified := time.Unix(int64(index+1), 0)
		if err := os.Chtimes(path, modified, modified); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	first, total, version, found, more := store.Page("title", "", 0, 2)
	if total != 3 || version == 0 || !found || !more || len(first) != 2 || first[0].Title != "Alpha" || first[1].Title != "Bravo" {
		t.Fatalf("title page = %#v, total=%d version=%d found=%v more=%v", first, total, version, found, more)
	}
	second, _, _, found, more := store.Page("title", first[1].ID, 0, 2)
	if !found || more || len(second) != 1 || second[0].Title != "Charlie" {
		t.Fatalf("next page = %#v, found=%v more=%v", second, found, more)
	}
	recent, _, _, _, _ := store.Page("recent", "", 0, 3)
	if len(recent) != 3 || recent[0].Title != "Charlie" || recent[2].Title != "Bravo" {
		t.Fatalf("recent page = %#v", recent)
	}
	if page, _, _, found, _ := store.Page("title", "missing", 0, 2); found || page != nil {
		t.Fatalf("missing anchor = %#v, found=%v", page, found)
	}
}
