package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestPersistentPoolRestoresTerminalProgress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	pool, err := NewPersistent(1, 2, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Do(context.Background(), "download", func(ctx context.Context) error {
		Report(ctx, 10, 10, "Downloaded")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	pool.Close()
	restored, err := NewPersistent(1, 2, path)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	state, ok := restored.State("download")
	if !ok || state.Status != "ready" || state.ProgressCurrent != 10 || state.ProgressTotal != 10 || state.ProgressMessage != "Downloaded" {
		t.Fatalf("restored state = %#v", state)
	}
	if err := restored.Wait(context.Background(), "download"); err != nil {
		t.Fatalf("restored completion = %v", err)
	}
}

func TestPersistentPoolMarksInterruptedStateFailedAndAllowsRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	queued, started := time.Now().Add(-time.Minute).UTC(), time.Now().Add(-30*time.Second).UTC()
	fixture := persistentState{Version: persistentStateVersion, States: []State{{Key: "episode", Status: "running", QueuedAt: queued, StartedAt: &started, ProgressCurrent: 40, ProgressTotal: 100, ProgressMessage: "Downloading episode"}}}
	data, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	pool, err := NewPersistent(1, 2, path)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	state, ok := pool.State("episode")
	if !ok || state.Status != "failed" || state.FinishedAt == nil || state.ProgressCurrent != 40 || !strings.Contains(state.Error, "restart") {
		t.Fatalf("interrupted state = %#v", state)
	}
	if err := pool.Wait(context.Background(), "episode"); err == nil || !strings.Contains(err.Error(), "restart") {
		t.Fatalf("interrupted completion = %v", err)
	}
	if err := pool.Do(context.Background(), "episode", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("retry = %v", err)
	}
	state, _ = pool.State("episode")
	if state.Status != "ready" || state.ProgressCurrent != 0 {
		t.Fatalf("retried state = %#v", state)
	}
}

func TestPersistentPoolRejectsMalformedAndOversizedState(t *testing.T) {
	root := t.TempDir()
	malformed := filepath.Join(root, "malformed.json")
	if err := os.WriteFile(malformed, []byte(`{"version":1,"states":[{"key":"x"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if pool, err := NewPersistent(1, 1, malformed); err == nil || pool != nil {
		t.Fatalf("malformed state accepted: %#v, %v", pool, err)
	}
	oversized := filepath.Join(root, "oversized.json")
	if err := os.WriteFile(oversized, make([]byte, (4<<20)+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if pool, err := NewPersistent(1, 1, oversized); err == nil || pool != nil {
		t.Fatalf("oversized state accepted: %#v, %v", pool, err)
	}
}

func TestPersistentPoolDoesNotRestoreRejectedQueueEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	pool, err := NewPersistent(1, 1, path)
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	if err := pool.Submit("running", func(context.Context) error { close(started); <-release; return nil }); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := pool.Submit("queued", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := pool.Submit("rejected", func(context.Context) error { return nil }); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("full queue error = %v", err)
	}
	close(release)
	pool.Close()
	restored, err := NewPersistent(1, 1, path)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if _, ok := restored.State("rejected"); ok {
		t.Fatal("rejected queue entry was persisted")
	}
}

func TestRejectedSubmissionIsNotOnDiskImmediately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	pool, err := NewPersistent(1, 1, path)
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	defer func() { close(release); pool.Close() }()
	if err := pool.Submit("running", func(context.Context) error { close(started); <-release; return nil }); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := pool.Submit("queued", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := pool.Submit("rejected", func(context.Context) error { return nil }); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("full queue error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "rejected") {
		t.Fatalf("rejected job is still recorded on disk: %s", data)
	}
}

func TestFailedStateSaveIsReportedNotDropped(t *testing.T) {
	pool, err := NewPersistent(1, 1, filepath.Join(t.TempDir(), "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	pool.persistPath = filepath.Join(blocker, "jobs.json") // the parent is a file, so every save fails
	var logged strings.Builder
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, nil)))
	defer slog.SetDefault(previous)
	pool.mu.Lock()
	pool.persistOrLog("finished")
	pool.mu.Unlock()
	if !strings.Contains(logged.String(), "job state could not be saved") || !strings.Contains(logged.String(), "finished") {
		t.Fatalf("a failed save left no trace: %q", logged.String())
	}
}

// The state loader rejects a message over its byte limit, and a truncation inside
// a multi-byte character re-encodes wider than the limit: the pool then refused to
// load the file it had just written, dropping every queued job with it.
func TestTruncationNeverSplitsARune(t *testing.T) {
	text := strings.Repeat("吾輩は猫である", 20)
	cut := truncateUTF8(text, maximumProgressMessageBytes)
	if len(cut) > maximumProgressMessageBytes || !utf8.ValidString(cut) {
		t.Fatalf("truncated message is not a valid %d-byte string: %q (%d bytes)", maximumProgressMessageBytes, cut, len(cut))
	}
	state := State{Key: "scan", Status: "ready", QueuedAt: time.Now().UTC(), ProgressCurrent: 1, ProgressTotal: 2, ProgressMessage: cut}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var decoded State
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.ProgressMessage) > maximumProgressMessageBytes {
		t.Fatalf("the stored message grew to %d bytes, which the loader rejects", len(decoded.ProgressMessage))
	}
}
