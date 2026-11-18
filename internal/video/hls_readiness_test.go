package video

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Proves eviction now tracks a running size incrementally instead of
// re-walking every session directory on every call.
func TestHLSEvictTracksSizeIncrementallyInsteadOfReWalkingEveryCall(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-time.Hour)
	for _, key := range []string{"stale-a", "stale-b"} {
		directory := filepath.Join(root, key)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "seg00000.ts"), []byte("1234"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(directory, old, old); err != nil {
			t.Fatal(err)
		}
	}
	store := &HLSStore{root: root, maxBytes: 50, sessions: map[string]*hlsSession{}, readyAfter: 20 * time.Second}
	store.evict("keep")

	if err := os.WriteFile(filepath.Join(root, "stale-b", "seg00001.ts"), make([]byte, 1000), 0o600); err != nil {
		t.Fatal(err)
	}

	store.evict("keep")

	if _, err := os.Stat(filepath.Join(root, "stale-a")); err != nil {
		t.Fatalf("untouched session stale-a was evicted based on another session's unreported growth: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "stale-b")); err != nil {
		t.Fatalf("untouched session stale-b was evicted based on its own unreported growth: %v", err)
	}

	store.evict("stale-b")
	if _, err := os.Stat(filepath.Join(root, "stale-a")); !os.IsNotExist(err) {
		t.Fatalf("stale-a should have been evicted once stale-b's real size was learned: %v", err)
	}
}

func TestHLSEvictionProtectsCurrentAndRecentlyUsedSessionsAfterSeeding(t *testing.T) {
	root := t.TempDir()
	oldTime := time.Now().Add(-time.Hour)
	for _, key := range []string{"old", "recent", "current"} {
		directory := filepath.Join(root, key)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "seg00000.ts"), []byte("12345678"), 0o600); err != nil {
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

func TestHLSOpenWakesPromptlyWhenSegmentArrives(t *testing.T) {
	root := t.TempDir()
	store := &HLSStore{root: root, sessions: map[string]*hlsSession{}, readyAfter: 2 * time.Second}
	key := strings.Repeat("a", 32)
	directory := filepath.Join(root, key)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	session := &hlsSession{key: key, directory: directory, lastUsed: time.Now(), jobKey: "job"}
	store.sessions[key] = session

	stop := make(chan struct{})
	defer close(stop)
	go watchHLSSegments(stop, session)

	go func() {
		time.Sleep(120 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(directory, "seg00000.ts"), []byte("data"), 0o600)
	}()

	start := time.Now()
	file, _, err := store.Open(key, "seg00000.ts")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	if elapsed > 160*time.Millisecond {
		t.Fatalf("Open() took %v to notice a ready segment after it landed; signaling is not prompt", elapsed)
	}
}

func TestHLSOpenWakesPromptlyOnFailure(t *testing.T) {
	root := t.TempDir()
	store := &HLSStore{root: root, sessions: map[string]*hlsSession{}, readyAfter: 2 * time.Second}
	key := strings.Repeat("c", 32)
	directory := filepath.Join(root, key)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	session := &hlsSession{key: key, directory: directory, lastUsed: time.Now(), jobKey: "job"}
	store.sessions[key] = session

	go func() {
		time.Sleep(120 * time.Millisecond)
		session.mu.Lock()
		session.jobKey = ""
		session.failed = errors.New("conversion failed")
		session.mu.Unlock()
		session.wake()
	}()

	start := time.Now()
	_, _, err := store.Open(key, "seg00000.ts")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected the failure to surface")
	}
	if elapsed > 160*time.Millisecond {
		t.Fatalf("Open() took %v to notice the failure; signaling is not prompt", elapsed)
	}
}

func TestHLSOpenTimesOutWhenSegmentNeverArrives(t *testing.T) {
	root := t.TempDir()
	store := &HLSStore{root: root, sessions: map[string]*hlsSession{}, readyAfter: 120 * time.Millisecond}
	key := strings.Repeat("d", 32)
	directory := filepath.Join(root, key)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	store.sessions[key] = &hlsSession{key: key, directory: directory, lastUsed: time.Now(), jobKey: "job"}

	start := time.Now()
	_, _, err := store.Open(key, "seg00000.ts")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed < 100*time.Millisecond || elapsed > 400*time.Millisecond {
		t.Fatalf("timeout elapsed = %v, want close to the 120ms readyAfter", elapsed)
	}
}
