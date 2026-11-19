package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type observedWaitContext struct {
	context.Context
	observed chan struct{}
	once     sync.Once
}

func (c *observedWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.observed) })
	return c.Context.Done()
}

func TestPoolBoundsDeduplicatesAndCancels(t *testing.T) {
	p := New(1, 1)
	defer p.Close()
	started := make(chan struct{})
	release := make(chan struct{})
	if err := p.Submit("one", func(ctx context.Context) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := p.Submit("one", func(context.Context) error { return nil }); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate: %v", err)
	}
	if !p.Cancel("one") {
		t.Fatal("cancel failed")
	}
	deadline := time.Now().Add(time.Second)
	for {
		state, _ := p.State("one")
		if state.Status == "canceled" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cancel not visible")
		}
		time.Sleep(time.Millisecond)
	}
}
func TestPoolQueueIsBounded(t *testing.T) {
	p := New(1, 1)
	defer p.Close()
	hold := make(chan struct{})
	if err := p.Submit("running", func(context.Context) error { <-hold; return nil }); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := p.Submit("queued", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := p.Submit("full", func(context.Context) error { return nil }); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("queue: %v", err)
	}
	close(hold)
}

func TestPoolDoCoalescesAndReturnsResult(t *testing.T) {
	p := New(1, 2)
	defer p.Close()
	started := make(chan struct{})
	release := make(chan struct{})
	runs := 0
	var mu sync.Mutex
	run := func(context.Context) error {
		mu.Lock()
		runs++
		mu.Unlock()
		select {
		case <-started:
		default:
			close(started)
		}
		<-release
		return errors.New("expected failure")
	}
	first := make(chan error, 1)
	go func() { first <- p.Do(context.Background(), "shared", run) }()
	<-started
	secondContext := &observedWaitContext{Context: context.Background(), observed: make(chan struct{})}
	second := make(chan error, 1)
	go func() { second <- p.Do(secondContext, "shared", run) }()
	// Do has submitted (and therefore coalesced) once Wait observes the
	// context. Releasing earlier makes this test depend on goroutine scheduling:
	// the first job can finish before the second goroutine calls Submit.
	<-secondContext.observed
	close(release)
	if err := <-first; err == nil || err.Error() != "expected failure" {
		t.Fatalf("first error: %v", err)
	}
	if err := <-second; err == nil || err.Error() != "expected failure" {
		t.Fatalf("second error: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if runs != 1 {
		t.Fatalf("runs = %d", runs)
	}
}

func TestPoolCloseReleasesQueuedWaiters(t *testing.T) {
	p := New(1, 2)
	started := make(chan struct{})
	if err := p.Submit("running", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := p.Submit("queued", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	queued := make(chan error, 1)
	go func() { queued <- p.Wait(context.Background(), "queued") }()
	p.Close()
	if err := <-queued; !errors.Is(err, context.Canceled) {
		t.Fatalf("queued waiter: %v", err)
	}
}

func TestPoolBoundsTerminalHistory(t *testing.T) {
	p := New(1, 1)
	defer p.Close()
	for index := 0; index < 100; index++ {
		key := fmt.Sprintf("job-%03d", index)
		if err := p.Do(context.Background(), key, func(context.Context) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if states := p.Snapshot(); len(states) != 64 {
		t.Fatalf("history contains %d states, want 64", len(states))
	}
	if _, ok := p.State("job-000"); ok {
		t.Fatal("oldest terminal state was retained")
	}
	if _, ok := p.State("job-099"); !ok {
		t.Fatal("newest terminal state was pruned")
	}
}

func TestCanceledKeyCannotBeReusedUntilOldClosureExits(t *testing.T) {
	p := New(1, 2)
	defer p.Close()
	started := make(chan struct{})
	exited := make(chan struct{})
	if err := p.Submit("episode", func(ctx context.Context) error {
		close(started)
		defer close(exited)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	if !p.Cancel("episode") {
		t.Fatal("cancel failed")
	}
	if err := p.Submit("episode", func(context.Context) error { return nil }); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("early reuse: %v", err)
	}
	<-exited
	deadline := time.Now().Add(time.Second)
	for {
		err := p.Submit("episode", func(context.Context) error { return nil })
		if err == nil {
			break
		}
		if !errors.Is(err, ErrDuplicate) || time.Now().After(deadline) {
			t.Fatalf("reuse after exit: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
	if err := p.Wait(context.Background(), "episode"); err != nil {
		t.Fatal(err)
	}
}

func TestPoolReportsBoundedProgressAndRetainsTerminalValue(t *testing.T) {
	p := New(1, 2)
	defer p.Close()
	reported := make(chan struct{})
	release := make(chan struct{})
	if err := p.Submit("download", func(ctx context.Context) error {
		if !Report(ctx, 25, 100, strings.Repeat("x", 140)) {
			t.Fatal("running job did not accept progress")
		}
		close(reported)
		<-release
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	<-reported
	state, ok := p.State("download")
	if !ok || state.ProgressCurrent != 25 || state.ProgressTotal != 100 || len(state.ProgressMessage) != 120 {
		t.Fatalf("running progress = %#v", state)
	}
	if Report(context.Background(), 1, 2, "outside") || Report(context.Background(), 3, 2, "invalid") {
		t.Fatal("progress accepted outside its running job")
	}
	close(release)
	if err := p.Wait(context.Background(), "download"); err != nil {
		t.Fatal(err)
	}
	state, _ = p.State("download")
	if state.Status != "ready" || state.ProgressCurrent != 25 {
		t.Fatalf("terminal progress = %#v", state)
	}
}

func TestAPanickingJobFailsAloneAndTheWorkerSurvives(t *testing.T) {
	pool := New(1, 4)
	defer pool.Close()
	if err := pool.Do(context.Background(), "bad", func(context.Context) error { panic("boom") }); err == nil {
		t.Fatal("a panicking job must report failure")
	}
	if err := pool.Do(context.Background(), "good", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("the worker must keep serving after a panic: %v", err)
	}
}
