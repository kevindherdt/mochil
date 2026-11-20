package transcode

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEvictionProtectsCurrentAndRecentlyUsedSessions(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-time.Hour)
	for _, key := range []string{"old", "recent", "current"} {
		directory := filepath.Join(root, key)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "seg00000.m4s"), []byte("12345678"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(directory, old, old); err != nil {
			t.Fatal(err)
		}
	}
	engine := &Engine{root: root, maxBytes: 16, sessions: map[string]*session{}, readyAfter: 20 * time.Second}
	engine.sessions["recent"] = &session{key: "recent", directory: filepath.Join(root, "recent"), lastUsed: time.Now()}
	engine.evict("current")
	if _, err := os.Stat(filepath.Join(root, "old")); !os.IsNotExist(err) {
		t.Fatalf("old cache was not evicted: %v", err)
	}
	for _, key := range []string{"recent", "current"} {
		if _, err := os.Stat(filepath.Join(root, key)); err != nil {
			t.Fatalf("protected cache %q removed: %v", key, err)
		}
	}
}

// TestOverlappingEvictsDoNotSubtractTheSameSessionTwice guards the cache
// accounting when two encodes finish together. Each evict snapshots
// statsEntries, then deletes. If the second one runs between the first one's
// snapshot and delete, both remove the same session: RemoveAll on an already
// missing directory succeeds, so the size used to be subtracted from
// statsTotal twice and the total undershot the real size, which is the only
// thing that decides whether eviction runs. The first evict is held in that
// window by a session mutex it takes before deleting.
func TestOverlappingEvictsDoNotSubtractTheSameSessionTwice(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-time.Hour)
	for _, key := range []string{"stale", "a", "b"} {
		directory := filepath.Join(root, key)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "seg00000.m4s"), []byte("12345678"), 0o600); err != nil {
			t.Fatal(err)
		}
		// "stale" is the oldest so both evicts try it first.
		when := old
		if key != "stale" {
			when = old.Add(time.Minute)
		}
		if err := os.Chtimes(directory, when, when); err != nil {
			t.Fatal(err)
		}
	}
	engine := &Engine{root: root, maxBytes: 16, sessions: map[string]*session{}, readyAfter: 20 * time.Second}
	// Seed the stats now (24 bytes, over the 16 limit) so each evict goes straight to snapshot + delete.
	engine.statsMu.Lock()
	engine.seedCacheStatsLocked()
	engine.statsMu.Unlock()

	// Holding this mutex freezes the first evict right after its snapshot, before it deletes "stale".
	frozen := &session{key: "stale", directory: filepath.Join(root, "stale"), lastUsed: old}
	engine.sessions["stale"] = frozen
	frozen.mu.Lock()
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		engine.evict("a")
	}()
	time.Sleep(100 * time.Millisecond) // let the first evict reach the frozen mutex

	// The second evict finishes while the first is frozen, and removes "stale" itself.
	engine.mu.Lock()
	delete(engine.sessions, "stale")
	engine.mu.Unlock()
	engine.evict("b")
	frozen.mu.Unlock()

	select {
	case <-firstDone:
	case <-time.After(5 * time.Second):
		t.Fatal("the first evict never finished")
	}

	engine.statsMu.Lock()
	defer engine.statsMu.Unlock()
	var sum int64
	for _, stat := range engine.statsEntries {
		sum += stat.size
	}
	if engine.statsTotal != sum {
		t.Fatalf("statsTotal=%d but the remaining entries add up to %d: a session removed by two evicts was subtracted twice", engine.statsTotal, sum)
	}
}
