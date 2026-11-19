package web

import (
	"testing"
	"time"

	"mochil/internal/catalog"
)

func TestDerivedCacheEvictsOldestAndIgnoresHugeBodies(t *testing.T) {
	cache := newDerivedCache(2, 1000)
	cache.put("a", []byte("1"))
	cache.put("b", []byte("2"))
	cache.get("a")
	cache.put("c", []byte("3"))
	if _, ok := cache.get("b"); ok {
		t.Fatal("the least recently used entry should have been evicted")
	}
	if _, ok := cache.get("a"); !ok {
		t.Fatal("a recently read entry should stay")
	}
	cache.put("big", make([]byte, 600))
	if _, ok := cache.get("big"); ok {
		t.Fatal("one body must not be allowed to take over the cache")
	}
}

func TestDerivedKeyChangesWithTheFile(t *testing.T) {
	book := catalog.Book{ID: "x", Size: 10, Modified: time.Unix(1, 0), Title: "T"}
	first := derivedKey("lib", book, "overlay")
	book.Modified = time.Unix(2, 0)
	if derivedKey("lib", book, "overlay") == first {
		t.Fatal("a changed file must not reuse the cached result")
	}
}
