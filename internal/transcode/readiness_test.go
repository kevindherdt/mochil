package transcode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Proves eviction now tracks a running size incrementally instead of
// re-walking every session directory on every call: growing an untouched
// session's directory between two evict() calls must NOT be noticed until
// that session itself is next refreshed (e.g. when its own job finishes and
// calls evict with its own key).
func TestEvictTracksSizeIncrementallyInsteadOfReWalkingEveryCall(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-time.Hour)
	for _, key := range []string{"stale-a", "stale-b"} {
		directory := filepath.Join(root, key)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "seg00000.m4s"), []byte("1234"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(directory, old, old); err != nil {
			t.Fatal(err)
		}
	}
	engine := &Engine{root: root, maxBytes: 50, sessions: map[string]*session{}, readyAfter: 20 * time.Second}
	engine.evict("keep") // seeds the running total: 8 bytes, well under budget.

	// Grow stale-b's directory far past the budget WITHOUT ever calling
	// evict with "stale-b" as the key.
	if err := os.WriteFile(filepath.Join(root, "stale-b", "seg00001.m4s"), make([]byte, 1000), 0o600); err != nil {
		t.Fatal(err)
	}

	engine.evict("keep") // same keepKey; stale-b's growth is still untracked.

	if _, err := os.Stat(filepath.Join(root, "stale-a")); err != nil {
		t.Fatalf("untouched session stale-a was evicted based on another session's unreported growth: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "stale-b")); err != nil {
		t.Fatalf("untouched session stale-b was evicted based on its own unreported growth: %v", err)
	}

	// Once stale-b is itself refreshed (as happens when its own job
	// finishes), its real size must be picked up and acted on.
	engine.evict("stale-b")
	if _, err := os.Stat(filepath.Join(root, "stale-a")); !os.IsNotExist(err) {
		t.Fatalf("stale-a should have been evicted once stale-b's real size was learned: %v", err)
	}
}

// Regression guard: normal eviction by size/age still works once the
// running total is accurate.
func TestEvictionProtectsCurrentAndRecentlyUsedSessionsAfterSeeding(t *testing.T) {
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

// Proves readers are woken promptly when the ffmpeg-writing goroutine's
// watcher notices a new segment, instead of waiting for the next fixed poll
// tick.
func TestEngineOpenWakesPromptlyWhenSegmentArrives(t *testing.T) {
	root := t.TempDir()
	engine := &Engine{root: root, sessions: map[string]*session{}, readyAfter: 2 * time.Second}
	key := strings.Repeat("a", 32)
	directory := filepath.Join(root, key)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	current := &session{key: key, directory: directory, lastUsed: time.Now(), jobKey: "job"}
	engine.sessions[key] = current

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchSegments(ctx, current)

	go func() {
		time.Sleep(120 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(directory, "seg00000.m4s"), []byte("data"), 0o600)
	}()

	start := time.Now()
	file, _, err := engine.Open(key, "seg00000.m4s")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	if elapsed > 160*time.Millisecond {
		t.Fatalf("Open() took %v to notice a ready segment after it landed; signaling is not prompt", elapsed)
	}
}

// Proves a waiting reader is woken by a failure too, not just by a segment
// appearing.
func TestEngineOpenWakesPromptlyOnFailure(t *testing.T) {
	root := t.TempDir()
	engine := &Engine{root: root, sessions: map[string]*session{}, readyAfter: 2 * time.Second}
	key := strings.Repeat("c", 32)
	directory := filepath.Join(root, key)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	current := &session{key: key, directory: directory, lastUsed: time.Now(), jobKey: "job"}
	engine.sessions[key] = current

	go func() {
		time.Sleep(120 * time.Millisecond)
		current.mu.Lock()
		current.jobKey = ""
		current.failed = errors.New("conversion failed")
		current.mu.Unlock()
		current.wake()
	}()

	start := time.Now()
	_, _, err := engine.Open(key, "seg00000.m4s")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected the failure to surface")
	}
	if elapsed > 160*time.Millisecond {
		t.Fatalf("Open() took %v to notice the failure; signaling is not prompt", elapsed)
	}
}

// Readiness waiting must still correctly time out when the segment never
// arrives.
func TestEngineOpenTimesOutWhenSegmentNeverArrives(t *testing.T) {
	root := t.TempDir()
	engine := &Engine{root: root, sessions: map[string]*session{}, readyAfter: 120 * time.Millisecond}
	key := strings.Repeat("d", 32)
	directory := filepath.Join(root, key)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	engine.sessions[key] = &session{key: key, directory: directory, lastUsed: time.Now(), jobKey: "job"}

	start := time.Now()
	_, _, err := engine.Open(key, "seg00000.m4s")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed < 100*time.Millisecond || elapsed > 400*time.Millisecond {
		t.Fatalf("timeout elapsed = %v, want close to the 120ms readyAfter", elapsed)
	}
}
