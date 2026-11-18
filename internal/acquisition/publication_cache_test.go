package acquisition

import (
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"mochil/internal/catalog"
)

func TestPublicationCacheStoresAndExpiresBooks(t *testing.T) {
	cache := NewPublicationCache(t.TempDir(), 64<<20, time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	book := catalog.Book{ID: "one", Path: "/Series/One.epub", Modified: time.Unix(100, 0), Size: 4}
	temporary, target, ok := cache.Begin("library", book)
	if !ok {
		t.Fatal("cache refused a small publication")
	}
	if _, err := temporary.Write([]byte("book")); err != nil {
		t.Fatal(err)
	}
	if !cache.Commit(temporary, target) {
		t.Fatal("cache commit failed")
	}
	file, info, hit := cache.Open("library", book)
	if !hit {
		t.Fatal("expected publication cache hit")
	}
	file.Close()
	if info.Size() != 4 {
		t.Fatalf("cached size = %d, want 4", info.Size())
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(target, old, old); err != nil {
		t.Fatal(err)
	}
	if file, _, hit := cache.Open("library", book); hit {
		file.Close()
		t.Fatal("expected expired publication to be removed")
	}
}

func TestPublicationCacheKeyChangesWithBookRevision(t *testing.T) {
	cache := NewPublicationCache(t.TempDir(), 64<<20, time.Hour, slog.Default())
	first := catalog.Book{ID: "one", Path: "/One.epub", Modified: time.Unix(100, 0)}
	second := first
	second.Modified = time.Unix(200, 0)
	if cache.Path("library", first) == cache.Path("library", second) {
		t.Fatal("book revisions must not share a cache entry")
	}
}

func BenchmarkPublicationCacheHit(b *testing.B) {
	cache := NewPublicationCache(b.TempDir(), 64<<20, time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	book := catalog.Book{ID: "one", Path: "/Series/One.epub", Modified: time.Unix(100, 0), Size: 4}
	temporary, target, ok := cache.Begin("library", book)
	if !ok {
		b.Fatal("cache refused benchmark publication")
	}
	if _, err := temporary.Write([]byte("book")); err != nil {
		b.Fatal(err)
	}
	if !cache.Commit(temporary, target) {
		b.Fatal("cache commit failed")
	}
	b.ResetTimer()
	for range b.N {
		file, _, hit := cache.Open("library", book)
		if !hit {
			b.Fatal("expected cache hit")
		}
		if err := file.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
