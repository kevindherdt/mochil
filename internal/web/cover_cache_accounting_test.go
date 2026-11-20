package web

import (
	"os"
	"path/filepath"
	"testing"
)

func directoryBytes(t *testing.T, directory string) int64 {
	t.Helper()
	var total int64
	_ = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// Thumbnails used to add their full size on every write without subtracting the
// file they replaced, so a cover re-rendered a few times counted several times
// its real size against the disk budget, and variant writes never evicted at all.
func TestThumbnailWritesCountOnceAndRespectTheBudget(t *testing.T) {
	directory := t.TempDir()
	cache := newCoverCache(directory, 1<<30, 300, 450, 75, nil)
	cache.sizeReady = true
	full := testJPEG(t, 400, 600, 7)

	small, ok := cache.thumbnail("books", "one", full, 160)
	if !ok {
		t.Fatal("the thumbnail could not be made")
	}
	first := cache.size
	if first != int64(len(small)) {
		t.Fatalf("size after one thumbnail = %d, want %d", first, len(small))
	}
	if _, ok := cache.thumbnail("books", "one", full, 160); !ok {
		t.Fatal("the second thumbnail could not be made")
	}
	if cache.size != first {
		t.Fatalf("rewriting the same thumbnail counted its %d bytes twice: %d", len(small), cache.size)
	}
	if onDisk := directoryBytes(t, directory); cache.size != onDisk {
		t.Fatalf("accounting says %d but the cache holds %d", cache.size, onDisk)
	}

	// The thumbnail path never enforced the budget; now it evicts like a full
	// cover write, and the accounting follows the deletions.
	cache.maximumBytes = 1
	if _, ok := cache.thumbnail("books", "two", full, 160); !ok {
		t.Fatal("the second book's thumbnail could not be made")
	}
	if onDisk := directoryBytes(t, directory); cache.size != onDisk {
		t.Fatalf("after eviction accounting says %d but the cache holds %d", cache.size, onDisk)
	}
	if cache.size > 0 && cache.size == first*2 {
		t.Fatalf("nothing was evicted under a 1-byte budget: %d bytes", cache.size)
	}
}

// The full-cover path and the thumbnail path share one accounting rule, so a
// replacement counts as a delta rather than as a second copy.
func TestReplacingACoverCountsOnlyTheDifference(t *testing.T) {
	cache := newCoverCache(t.TempDir(), 1<<30, 300, 450, 75, nil)
	cache.sizeReady = true
	path := cache.path("books", "one")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	first := testJPEG(t, 400, 600, 3)
	if err := os.WriteFile(path, first, 0o600); err != nil {
		t.Fatal(err)
	}
	cache.size = int64(len(first))

	second := testJPEG(t, 400, 600, 9)
	cache.publishLocked(path, second)
	if want := int64(len(second)); cache.size != want {
		t.Fatalf("size after replacing %d bytes with %d = %d, want %d", len(first), len(second), cache.size, want)
	}
}
