package narration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeWorker speaks the worker protocol without a model: every request answers
// with two samples per character, and "crash" kills the process.
const fakeWorker = `
import sys, os, json, base64, struct, time
print(json.dumps({"ready": True, "load_seconds": 0, "sample_rate": 8000}), flush=True)
for line in sys.stdin:
    r = json.loads(line)
    if r.get("op") == "warm":
        print(json.dumps({"id": r["id"], "ok": True}), flush=True)
        continue
    if r.get("op") == "encode":
        if os.path.exists(os.path.join(os.path.dirname(__file__), "noclone")):
            print(json.dumps({"id": r["id"], "ok": False, "code": "cloning_unavailable"}), flush=True)
            continue
        open(r["state"], "wb").write(b"state:" + r["language"].encode())
        print(json.dumps({"id": r["id"], "ok": True}), flush=True)
        continue
    if r["text"] == "flood":
        open(os.path.join(os.path.dirname(__file__), "pid"), "w").write(str(os.getpid()))
        for _ in range(40):
            print(json.dumps({"id": r["id"], "more": True, "sample_rate": 8000, "pcm": ""}), flush=True)
        time.sleep(30)
    if r["text"] == "crash":
        sys.exit(3)
    if r["text"] == "slow":
        time.sleep(30)
    if r["text"] == "bad":
        print(json.dumps({"id": r["id"], "ok": False, "code": "generation_failed"}), flush=True)
        continue
    pcm = struct.pack("<h", 100) * (len(r["text"]) * 2)
    if r.get("stream"):
        # three pieces, then the final reply without audio
        for _ in range(3):
            print(json.dumps({"id": r["id"], "more": True, "sample_rate": 8000, "pcm": base64.b64encode(pcm).decode()}), flush=True)
        print(json.dumps({"id": r["id"], "ok": True, "sample_rate": 8000}), flush=True)
        continue
    print(json.dumps({"id": r["id"], "ok": True, "sample_rate": 8000, "pcm": base64.b64encode(pcm).decode()}), flush=True)
`

func newFake(t *testing.T, idle int) *Engine {
	t.Helper()
	script := filepath.Join(t.TempDir(), "fake.py")
	if err := os.WriteFile(script, []byte(fakeWorker), 0o600); err != nil {
		t.Fatal(err)
	}
	e := New(Config{IdleSecs: idle, command: []string{"python3", "-u", script}})
	t.Cleanup(e.Close)
	return e
}

func TestSpeakReturnsAudioAndKeepsTheModelLoaded(t *testing.T) {
	e := newFake(t, 60)
	if e.Loaded() {
		t.Fatal("nothing should be loaded before the first request")
	}
	audio, err := e.Speak(context.Background(), "Hello there.", "alba", "english")
	if err != nil {
		t.Fatal(err)
	}
	if audio.SampleRate != 8000 || len(audio.PCM) != len("Hello there.")*4 {
		t.Fatalf("unexpected audio: %d Hz, %d bytes", audio.SampleRate, len(audio.PCM))
	}
	if !e.Loaded() {
		t.Fatal("the worker should stay loaded between sentences")
	}
	if _, err := e.Speak(context.Background(), "Again.", "alba", "english"); err != nil {
		t.Fatal(err)
	}
}

func TestAWorkerFailureIsReportedAndTheNextRequestStillWorks(t *testing.T) {
	e := newFake(t, 60)
	if _, err := e.Speak(context.Background(), "bad", "alba", "english"); !errors.Is(err, ErrFailed) {
		t.Fatalf("got %v", err)
	}
	if _, err := e.Speak(context.Background(), "fine", "alba", "english"); err != nil {
		t.Fatalf("the engine did not recover: %v", err)
	}
}

func TestACrashedWorkerIsRestarted(t *testing.T) {
	e := newFake(t, 60)
	if _, err := e.Speak(context.Background(), "crash", "alba", "english"); !errors.Is(err, ErrFailed) {
		t.Fatalf("got %v", err)
	}
	if e.Loaded() {
		t.Fatal("a dead worker must not be kept")
	}
	if _, err := e.Speak(context.Background(), "back again", "alba", "english"); err != nil {
		t.Fatalf("no restart: %v", err)
	}
}

func TestCancellingDropsAbandonedWork(t *testing.T) {
	e := newFake(t, 60)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := e.Speak(ctx, "slow", "alba", "english"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	if time.Since(started) > 5*time.Second {
		t.Fatal("cancel did not return promptly")
	}
	if _, err := e.Speak(context.Background(), "quick", "alba", "english"); err != nil {
		t.Fatalf("the next request was stuck behind abandoned work: %v", err)
	}
}

func TestTheModelIsUnloadedWhenIdle(t *testing.T) {
	e := newFake(t, 1)
	if _, err := e.Speak(context.Background(), "hi", "alba", "english"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for e.Loaded() && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if e.Loaded() {
		t.Fatal("the worker should be unloaded after the idle period")
	}
	if _, err := e.Speak(context.Background(), "hi again", "alba", "english"); err != nil {
		t.Fatalf("it should start again on demand: %v", err)
	}
}

func TestLimitsAndAvailability(t *testing.T) {
	e := New(Config{})
	if e.Available() {
		t.Fatal("no interpreter configured")
	}
	if _, err := e.Speak(context.Background(), "hi", "alba", "english"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
	if _, err := newFake(t, 60).Speak(context.Background(), strings.Repeat("a", MaxTextBytes+1), "alba", "english"); !errors.Is(err, ErrTooLong) {
		t.Fatalf("got %v", err)
	}
	if New(Config{Python: "/definitely/not/here"}).Available() {
		t.Fatal("a missing interpreter is not available")
	}
}

func TestWAVHasAValidHeader(t *testing.T) {
	wav := Audio{SampleRate: 24000, PCM: make([]byte, 48000)}.WAV()
	if string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" || len(wav) != 44+48000 {
		t.Fatalf("bad WAV: %q len %d", wav[:12], len(wav))
	}
	if got := (Audio{SampleRate: 24000, PCM: make([]byte, 48000)}).Seconds(); got != 1 {
		t.Fatalf("seconds = %v", got)
	}
}

func TestWarmLoadsTheModelWithoutSpeaking(t *testing.T) {
	e := newFake(t, 60)
	if err := e.Warm(context.Background(), "alba", "english"); err != nil {
		t.Fatal(err)
	}
	if !e.Loaded() {
		t.Fatal("warming should leave the worker loaded")
	}
}

func TestSpeakStreamHandsOverEachPieceAndStopsWhenTheCallerLeaves(t *testing.T) {
	e := newFake(t, 600)
	var pieces, bytes int
	err := e.SpeakStream(context.Background(), "hello", "alba", "english", func(a Audio) error {
		pieces++
		bytes += len(a.PCM)
		if a.SampleRate != 8000 {
			t.Fatalf("sample rate %d", a.SampleRate)
		}
		return nil
	})
	if err != nil || pieces != 3 || bytes != 3*len("hello")*2*2 {
		t.Fatalf("pieces=%d bytes=%d err=%v", pieces, bytes, err)
	}
	// A caller that leaves part-way (its emit fails) ends the sentence; the next one still works.
	stop := errors.New("client left")
	if err := e.SpeakStream(context.Background(), "hello", "alba", "english", func(Audio) error { return stop }); !errors.Is(err, stop) {
		t.Fatalf("emit error should end the sentence: %v", err)
	}
	if _, err := e.Speak(context.Background(), "again", "alba", "english"); err != nil {
		t.Fatalf("the engine recovers after an abandoned sentence: %v", err)
	}
}

func TestAbandonedStreamingWorkerIsReaped(t *testing.T) {
	directory := t.TempDir()
	script := filepath.Join(directory, "fake.py")
	if err := os.WriteFile(script, []byte(fakeWorker), 0o600); err != nil {
		t.Fatal(err)
	}
	e := New(Config{IdleSecs: 60, command: []string{"python3", "-u", script}})
	t.Cleanup(e.Close)
	left := errors.New("caller left")
	err := e.SpeakStream(context.Background(), "flood", "alba", "english", func(Audio) error { return left })
	if !errors.Is(err, left) {
		t.Fatalf("err=%v", err)
	}
	pid, err := os.ReadFile(filepath.Join(directory, "pid"))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat("/proc/" + string(pid)); os.IsNotExist(err) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the abandoned worker was left running or as a zombie")
}
