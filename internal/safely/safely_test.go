package safely

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunContainsAPanicAndReturns(t *testing.T) {
	ran := false
	Run("test run", func() { panic("boom") })
	Run("test run", func() { ran = true })
	if !ran {
		t.Fatal("work after a panic did not run")
	}
}

func TestRecoverEndsOnlyTheGoroutineItGuards(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer Recover("test recover")
		panic("boom")
	}()
	<-done
}

func find(what string) *Failure {
	for _, failure := range Failures() {
		if failure.What == what {
			return &failure
		}
	}
	return nil
}

func TestAFailureIsRememberedWithItsMessageAndCount(t *testing.T) {
	Run("test remembered", func() { panic("first") })
	Run("test remembered", func() { panic("second") })
	got := find("test remembered")
	if got == nil || got.Count != 2 || got.Message != "second" || got.At == 0 {
		t.Fatalf("got %+v", got)
	}
	Run("test long", func() { panic(string(make([]byte, 500))) })
	if got := find("test long"); len(got.Message) > 210 {
		t.Fatalf("message not shortened: %d", len(got.Message))
	}
}

func TestKeepStartsAPanickedWorkerAgainUntilItFinishes(t *testing.T) {
	firstDelay = time.Millisecond
	var runs atomic.Int32
	Keep(context.Background(), "test keep", func() {
		if runs.Add(1) < 3 {
			panic("again")
		}
	})
	if runs.Load() != 3 {
		t.Fatalf("ran %d times, want 3", runs.Load())
	}
	if got := find("test keep"); got.Count != 2 || got.Restarts != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestKeepStopsRestartingWhenTheServerIsShuttingDown(t *testing.T) {
	firstDelay = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { Keep(ctx, "test stop", func() { panic("always") }); close(done) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Keep did not return after the context ended")
	}
}

func TestRestartsWaitLongerEachTimeUpToACeiling(t *testing.T) {
	saved := longestDelay
	defer func() { longestDelay = saved }()
	longestDelay = 30 * time.Second
	delay := 5 * time.Second
	var seen []time.Duration
	for range 5 {
		delay = nextDelay(delay)
		seen = append(seen, delay)
	}
	want := []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("delays %v, want %v", seen, want)
		}
	}
}
