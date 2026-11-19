package narration

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed transcribe.py
var transcribeScript string

// MaxTranscribeBytes bounds one spoken turn (about two minutes of 16 kHz speech, far more than the 60 seconds allowed).
const MaxTranscribeBytes = 12 << 20

var (
	// ErrNoSpeech means the recording held nothing that could be understood.
	ErrNoSpeech = errors.New("no speech was heard")
	// ErrRecognitionUnavailable means speech recognition is not installed.
	ErrRecognitionUnavailable = errors.New("speech recognition is not set up")
)

const transcriptMark = "NARRATION_TRANSCRIPT:"

// CanTranscribe reports whether speech recognition looks installed: the voice environment exists and holds the model.
func (e *Engine) CanTranscribe() bool {
	if len(e.cfg.transcribeCommand) > 0 {
		return true
	}
	if !e.Available() || e.cfg.Cache == "" {
		return false
	}
	matches, _ := filepath.Glob(filepath.Join(e.cfg.Cache, "whisper", "models--Systran--faster-whisper-base.en", "snapshots", "*"))
	return len(matches) > 0
}

// Transcribe turns a short recording (any common audio container) into text. Each call runs a short-lived worker, so
// nothing stays in memory between turns.
func (e *Engine) Transcribe(ctx context.Context, recording []byte) (string, error) {
	if !e.CanTranscribe() {
		return "", ErrRecognitionUnavailable
	}
	if len(recording) == 0 || len(recording) > MaxTranscribeBytes {
		return "", fmt.Errorf("%w: the recording is empty or too long", ErrFailed)
	}
	file, err := os.CreateTemp("", "recording-*.audio")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailed, err)
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(recording)
	if closeErr := file.Close(); writeErr != nil || closeErr != nil {
		return "", fmt.Errorf("%w: could not store the recording", ErrFailed)
	}
	threads := e.cfg.Threads
	if threads < 1 || threads > 6 {
		threads = 4
	}
	input, _ := json.Marshal(map[string]any{"input": file.Name(), "cache": e.cfg.Cache, "threads": threads})
	args := e.cfg.transcribeCommand
	if len(args) == 0 {
		args = []string{e.cfg.Python, "-c", transcribeScript}
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = bytes.NewReader(input)
	var output bytes.Buffer
	cmd.Stdout = &output
	_ = cmd.Run() // a failing worker still prints why; the printed line decides
	for _, line := range strings.Split(output.String(), "\n") {
		payload, found := strings.CutPrefix(strings.TrimSpace(line), transcriptMark)
		if !found {
			continue
		}
		var result struct {
			OK   bool   `json:"ok"`
			Text string `json:"text"`
			Code string `json:"code"`
		}
		if json.Unmarshal([]byte(payload), &result) != nil {
			break
		}
		switch {
		case result.OK && strings.TrimSpace(result.Text) != "":
			return strings.TrimSpace(result.Text), nil
		case result.Code == "no_speech" || result.Code == "invalid_audio":
			return "", ErrNoSpeech
		case result.Code == "unavailable":
			return "", ErrRecognitionUnavailable
		case result.Code == "audio_too_long":
			return "", fmt.Errorf("%w: the recording is too long", ErrFailed)
		}
		break
	}
	if ctx.Err() != nil {
		return "", fmt.Errorf("%w: speech recognition took too long", ErrFailed)
	}
	return "", fmt.Errorf("%w: speech recognition failed", ErrFailed)
}
