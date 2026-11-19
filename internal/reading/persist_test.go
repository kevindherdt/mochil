package reading

import (
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// A reader must never wait for a whole-file write: that was the point of moving
// persistence off the lock. A blocked writer is the sharpest version of the
// problem, because before this change the reader waited inside the same lock.
func TestABlockedWriterDoesNotDelayReaders(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	store.persist.window = time.Millisecond
	release := make(chan struct{})
	var entered atomic.Bool
	store.persist.write = func(string, storeData) error {
		entered.Store(true)
		<-release
		return nil
	}
	saved, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .3}, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Wait for the writer to be inside the blocked write.
	deadline := time.Now().Add(2 * time.Second)
	for !entered.Load() && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if !entered.Load() {
		t.Fatal("the background writer never started")
	}
	started := time.Now()
	if _, ok := store.Get("reader", "books", "one"); !ok {
		t.Fatal("the acknowledged position is not readable")
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("a read waited %v behind the writer", elapsed)
	}
	close(release)
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	if got, ok := store.Get("reader", "books", "one"); !ok || got.Revision != saved.Revision {
		t.Fatal("the position changed after the write completed")
	}
}

// Progress arrives at the pace of page turns; each one used to rewrite the file.
func TestRapidProgressSavesCoalesceIntoFewerWrites(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	store.persist.window = 50 * time.Millisecond
	var writes atomic.Int64
	store.persist.write = func(path string, state storeData) error {
		writes.Add(1)
		return writeStateFile(path, state)
	}
	const saves = 40
	for index := 0; index < saves; index++ {
		current, ok := store.Get("reader", "books", "one")
		base := int64(0)
		if ok {
			base = current.Revision
		}
		position := Position{LibraryID: "books", BookID: "one", Fraction: float64(index) / saves}
		if _, err := store.Put("reader", position, base); err != nil {
			t.Fatalf("save %d: %v", index, err)
		}
	}
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := writes.Load(); got >= saves {
		t.Fatalf("%d saves produced %d writes; they are not being coalesced", saves, got)
	}
	reopened, err := Open(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if last, ok := reopened.Get("reader", "books", "one"); !ok || last.Fraction <= 0 {
		t.Fatal("the newest position did not reach the file")
	}
}

// Flush is what shutdown and an account purge rely on.
func TestFlushWritesAcknowledgedProgress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reading.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	store.persist.window = time.Hour // never fires on its own
	if _, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .6}, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if position, ok := reopened.Get("reader", "books", "one"); !ok || position.Fraction != .6 {
		t.Fatalf("flushed position = %+v, ok=%v", position, ok)
	}
}
