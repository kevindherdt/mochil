package narration

import (
	"context"
	"errors"
	"testing"
)

func TestTranscribeReadsTheWorkersAnswer(t *testing.T) {
	run := func(script string) (string, error) {
		e := New(Config{transcribeCommand: []string{"sh", "-c", "cat >/dev/null; " + script}})
		return e.Transcribe(context.Background(), []byte("audio"))
	}
	if text, err := run(`echo 'NARRATION_TRANSCRIPT:{"ok":true,"text":" hello there "}'`); err != nil || text != "hello there" {
		t.Fatalf("got %q %v", text, err)
	}
	if _, err := run(`echo 'NARRATION_TRANSCRIPT:{"ok":false,"code":"no_speech"}'; exit 1`); !errors.Is(err, ErrNoSpeech) {
		t.Fatalf("silence: %v", err)
	}
	if _, err := run(`echo 'NARRATION_TRANSCRIPT:{"ok":false,"code":"unavailable"}'; exit 1`); !errors.Is(err, ErrRecognitionUnavailable) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := run(`echo 'noise only'; exit 1`); !errors.Is(err, ErrFailed) {
		t.Fatalf("crash: %v", err)
	}
	if _, err := New(Config{transcribeCommand: []string{"true"}}).Transcribe(context.Background(), nil); !errors.Is(err, ErrFailed) {
		t.Fatalf("empty recording: %v", err)
	}
	if New(Config{}).CanTranscribe() {
		t.Fatal("an unconfigured engine cannot transcribe")
	}
}
