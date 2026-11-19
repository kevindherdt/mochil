package video

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHLSEvictionProtectsCurrentAndRecentlyUsedSessions(t *testing.T) {
	root := t.TempDir()
	oldTime := time.Now().Add(-time.Hour)
	for _, key := range []string{"old", "recent", "current"} {
		directory := filepath.Join(root, key)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "seg00000.m4s"), []byte("12345678"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(directory, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
	}
	store := &HLSStore{root: root, maxBytes: 16, sessions: map[string]*hlsSession{}, readyAfter: 20 * time.Second}
	store.sessions["recent"] = &hlsSession{key: "recent", directory: filepath.Join(root, "recent"), lastUsed: time.Now()}
	store.evict("current")
	if _, err := os.Stat(filepath.Join(root, "old")); !os.IsNotExist(err) {
		t.Fatalf("old cache was not evicted: %v", err)
	}
	for _, key := range []string{"recent", "current"} {
		if _, err := os.Stat(filepath.Join(root, key)); err != nil {
			t.Fatalf("protected cache %q was removed: %v", key, err)
		}
	}
}

func TestHighestContiguousSegmentIgnoresDistantCachedRange(t *testing.T) {
	directory := t.TempDir()
	for _, segment := range []int{0, 1, 2, 20, 21} {
		name := filepath.Join(directory, fmt.Sprintf("seg%05d.ts", segment))
		if err := os.WriteFile(name, []byte("segment"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := highestContiguousSegment(directory, 0); got != 2 {
		t.Fatalf("highest contiguous segment = %d, want 2", got)
	}
	if got := highestContiguousSegment(directory, 20); got != 21 {
		t.Fatalf("distant contiguous segment = %d, want 21", got)
	}
}
