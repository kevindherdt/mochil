package jobs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func receiveWithin(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestCanceledRunningJobRetainsKeyUnderHistoryPressure(t *testing.T) {
	pool, err := NewPersistent(2, 2, filepath.Join(t.TempDir(), "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer func() { releaseOnce.Do(func() { close(release) }); pool.Close() }()
	if err := pool.Submit("episode", func(ctx context.Context) error {
		close(started)
		<-release // Deliberately ignore cancellation while unwinding.
		Report(ctx, 99, 100, "stale progress")
		return errors.New("old download failed")
	}); err != nil {
		t.Fatal(err)
	}
	receiveWithin(t, started, "old job to start")
	if !pool.Cancel("episode") {
		t.Fatal("cancel failed")
	}
	for i := 0; i < 64; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := pool.Do(ctx, fmt.Sprintf("history-%03d", i), func(context.Context) error { return nil })
		cancel()
		if err != nil {
			t.Fatalf("history job %d: %v", i, err)
		}
	}
	if state, ok := pool.State("episode"); !ok || state.Status != "canceled" {
		t.Errorf("canceled running owner lost under history pressure: state=%+v exists=%v", state, ok)
	}
	if err := pool.Submit("episode", func(context.Context) error { return nil }); !errors.Is(err, ErrDuplicate) {
		t.Errorf("same key accepted before old job exited: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	waitResult := make(chan error, 1)
	observed := &observedWaitContext{Context: ctx, observed: make(chan struct{})}
	go func() { waitResult <- pool.Wait(observed, "episode") }()
	receiveWithin(t, observed.observed, "canceled job waiter")
	select {
	case err := <-waitResult:
		t.Fatalf("running canceled job completed before exiting: %v", err)
	default:
	}
	releaseOnce.Do(func() { close(release) })
	if err := <-waitResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled job result: %v", err)
	}
	if err := pool.Do(ctx, "episode", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("reuse after exit: %v", err)
	}
	state, ok := pool.State("episode")
	if !ok || state.Status != "ready" || state.Error != "" {
		t.Fatalf("new owner state = %+v exists=%v", state, ok)
	}
}

func TestCanceledQueuedJobRetainsKeyUnderHistoryPressure(t *testing.T) {
	pool, err := NewPersistent(2, 2, filepath.Join(t.TempDir(), "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	aStarted, bStarted, aRelease, bRelease := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	aFinished := make(chan struct{})
	var aOnce sync.Once
	defer func() { aOnce.Do(func() { close(aRelease) }); close(bRelease); pool.Close() }()
	if err := pool.Submit("a", func(context.Context) error { close(aStarted); <-aRelease; defer close(aFinished); return nil }); err != nil {
		t.Fatal(err)
	}
	if err := pool.Submit("b", func(context.Context) error { close(bStarted); <-bRelease; return nil }); err != nil {
		t.Fatal(err)
	}
	receiveWithin(t, aStarted, "first blocker")
	receiveWithin(t, bStarted, "second blocker")
	if err := pool.Submit("queued", func(context.Context) error { t.Error("canceled queued job ran"); return nil }); err != nil {
		t.Fatal(err)
	}
	if !pool.Cancel("queued") {
		t.Fatal("cancel failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Wait(ctx, "queued"); !errors.Is(err, context.Canceled) {
		t.Fatalf("queued wait = %v", err)
	}
	pool.mu.Lock()
	pool.history = 2
	pool.pruneLocked()
	pool.mu.Unlock()
	if state, ok := pool.State("queued"); !ok || state.Status != "canceled" {
		t.Fatalf("queued owner pruned: %+v exists=%v", state, ok)
	}
	if err := pool.Submit("queued", func(context.Context) error { return nil }); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("queued key reused before dequeue: %v", err)
	}
	// A FIFO barrier proves the canceled entry has been consumed before retry. Wait on the job's
	// own signal rather than pool.Wait("a"): with the history shrunk to 2, completing "a" prunes
	// its bookkeeping, and Wait racing that prune reports "job does not exist" — a CI-only flake.
	aOnce.Do(func() { close(aRelease) })
	receiveWithin(t, aFinished, "first job finished")
	if err := pool.Do(ctx, "barrier", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := pool.Do(ctx, "queued", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("reuse consumed canceled key: %v", err)
	}
}

func TestReleasedCanceledOwnerIsPrunedImmediately(t *testing.T) {
	pool := New(1, 1)
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer func() { releaseOnce.Do(func() { close(release) }); pool.Close() }()
	pool.mu.Lock()
	pool.history = 0
	pool.mu.Unlock()
	if err := pool.Submit("episode", func(context.Context) error { close(started); <-release; return nil }); err != nil {
		t.Fatal(err)
	}
	receiveWithin(t, started, "job to start")
	if !pool.Cancel("episode") {
		t.Fatal("cancel failed")
	}
	pool.mu.RLock()
	completion := pool.done["episode"]
	pool.mu.RUnlock()
	releaseOnce.Do(func() { close(release) })
	receiveWithin(t, completion.done, "canceled owner exit")
	if _, ok := pool.State("episode"); ok {
		t.Fatal("released canceled owner remains above history limit")
	}
}
