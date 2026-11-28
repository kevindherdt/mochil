package video

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A cache write that fails must not throw away a scan that succeeded: a full disk used to empty the library on the next
// start, because the items were only published after the cache file was written.
func TestScanPublishesEvenWhenTheCacheCannotBeWritten(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Show.S01E01.mkv"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The cache path is a directory, so writing the cache file fails for anyone, root included.
	cache := filepath.Join(t.TempDir(), "cache-dir")
	if err := os.Mkdir(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	store := NewStore("tv", root, cache)
	stats, err := store.Scan()
	if err == nil {
		t.Fatal("the scan should report the cache write failure")
	}
	if stats.Items != 1 {
		t.Fatalf("stats.Items = %d, want the discovered item counted", stats.Items)
	}
	if items := store.Items(); len(items) != 1 || !strings.HasPrefix(items[0].Path, "Show") {
		t.Fatalf("the published catalogue lost the scanned item: %+v", items)
	}
}
