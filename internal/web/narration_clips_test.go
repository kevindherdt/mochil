package web

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTwoRequestsForOneSentenceShareTheWork(t *testing.T) {
	clips := newNarrationClips(1 << 20)
	var made atomic.Int32
	release := make(chan struct{})
	speak := func(context.Context) ([]byte, error) {
		made.Add(1)
		<-release
		return []byte("audio"), nil
	}
	var wait sync.WaitGroup
	results := make([]string, 3)
	for i := range results {
		wait.Add(1)
		go func() {
			defer wait.Done()
			wav, err := clips.get(context.Background(), "k", speak)
			if err != nil {
				t.Error(err)
			}
			results[i] = string(wav)
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wait.Wait()
	if made.Load() != 1 {
		t.Fatalf("the sentence was made %d times", made.Load())
	}
	for _, r := range results {
		if r != "audio" {
			t.Fatalf("got %q", r)
		}
	}
	// and a later request is answered from what is kept
	if _, err := clips.get(context.Background(), "k", speak); err != nil || made.Load() != 1 {
		t.Fatalf("a kept clip should not be made again: %v, made %d", err, made.Load())
	}
}

func TestAFailedSentenceIsTriedAgainNextTime(t *testing.T) {
	clips := newNarrationClips(1 << 20)
	calls := 0
	speak := func(context.Context) ([]byte, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("boom")
		}
		return []byte("ok"), nil
	}
	if _, err := clips.get(context.Background(), "k", speak); err == nil {
		t.Fatal("the first attempt should fail")
	}
	if clips.has("k") {
		t.Fatal("a failure must not be remembered")
	}
	if wav, err := clips.get(context.Background(), "k", speak); err != nil || string(wav) != "ok" {
		t.Fatalf("second attempt: %q %v", wav, err)
	}
}

func TestTheOldestClipsAreForgottenWhenTheCacheIsFull(t *testing.T) {
	clips := newNarrationClips(25)
	make10 := func(context.Context) ([]byte, error) { return make([]byte, 10), nil }
	for _, key := range []string{"a", "b", "c"} {
		if _, err := clips.get(context.Background(), key, make10); err != nil {
			t.Fatal(err)
		}
	}
	// The cleanup runs just after the answer is handed out.
	deadline := time.Now().Add(time.Second)
	for clips.has("a") && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if clips.has("a") || !clips.has("b") || !clips.has("c") {
		t.Fatalf("kept: a=%v b=%v c=%v", clips.has("a"), clips.has("b"), clips.has("c"))
	}
}

func TestOnlySomanySentencesAreMadeAtOnce(t *testing.T) {
	clips := newNarrationClips(1 << 20)
	hold := make(chan struct{})
	defer close(hold)
	slow := func(context.Context) ([]byte, error) { <-hold; return []byte("x"), nil }
	for i := 0; i < maxClipsInProgress; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		_, _ = clips.get(ctx, string(rune('A'+i)), slow)
		cancel()
	}
	if _, err := clips.get(context.Background(), "one-too-many", slow); !errors.Is(err, errNarrationBusy) {
		t.Fatalf("got %v", err)
	}
}

func TestACallerThatGivesUpDoesNotCancelTheWork(t *testing.T) {
	clips := newNarrationClips(1 << 20)
	release := make(chan struct{})
	speak := func(context.Context) ([]byte, error) { <-release; return []byte("late"), nil }
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := clips.get(ctx, "k", speak); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	close(release)
	if wav, err := clips.get(context.Background(), "k", speak); err != nil || string(wav) != "late" {
		t.Fatalf("the work should have carried on: %q %v", wav, err)
	}
}

func TestAPanickingVoiceFailsTheClipInsteadOfHangingIt(t *testing.T) {
	clips := newNarrationClips(1 << 20)
	explode := func(context.Context) ([]byte, error) { panic("voice engine bug") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := clips.get(ctx, "a", explode); err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a panic should fail the clip at once, got %v", err)
	}
	wav, err := clips.get(ctx, "a", func(context.Context) ([]byte, error) { return []byte("ok"), nil })
	if err != nil || string(wav) != "ok" {
		t.Fatalf("the clip must be tried again afterwards: %q %v", wav, err)
	}
}

func TestContainedSwallowsAPanicAndReturns(t *testing.T) {
	ran := false
	contained("test", func() { panic("boom") })
	contained("test", func() { ran = true })
	if !ran {
		t.Fatal("work after a contained panic must still run")
	}
}
