package narration

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	// MaxSampleBytes bounds an uploaded recording.
	MaxSampleBytes = 12 << 20
	sampleRate     = 24000
	minSampleSecs  = 3
	maxSampleSecs  = 30
)

var (
	ErrBadSample    = errors.New("that recording could not be read")
	ErrSampleLength = errors.New("use between 3 and 30 seconds of speech")
	ErrSampleQuiet  = errors.New("that recording is too quiet")
)

// Clone makes a saved voice from a recording of someone speaking. The audio is
// decoded and levelled with ffmpeg, then the speech model prepares the voice.
// Nothing is kept unless every step succeeds.
func (v *Voices) Clone(ctx context.Context, engine *Engine, recording io.Reader, name, language string) (Voice, error) {
	name, ok := cleanName(name)
	if !ok {
		return Voice{}, ErrBadName
	}
	if !ValidLanguage(language) {
		return Voice{}, errors.New("language must be english or dutch")
	}
	if len(v.List()) >= MaxVoices {
		return Voice{}, ErrTooManyVoices
	}
	if err := os.MkdirAll(v.dir, 0o750); err != nil {
		return Voice{}, err
	}
	work, err := os.MkdirTemp(v.dir, ".new-")
	if err != nil {
		return Voice{}, err
	}
	defer os.RemoveAll(work)

	upload := filepath.Join(work, "upload")
	if err := writeLimited(upload, recording, MaxSampleBytes); err != nil {
		return Voice{}, err
	}
	sample := filepath.Join(work, sampleFile)
	if err := ffmpeg(ctx, engine.ffmpegPath(), "-i", upload, "-vn", "-ac", "1", "-ar", fmt.Sprint(sampleRate), "-t", fmt.Sprint(maxSampleSecs+1), "-c:a", "pcm_s16le", sample); err != nil {
		return Voice{}, ErrBadSample
	}
	seconds, loudness, err := inspectWAV(sample)
	if err != nil {
		return Voice{}, ErrBadSample
	}
	if seconds < minSampleSecs || seconds > maxSampleSecs {
		return Voice{}, ErrSampleLength
	}
	if loudness < 0.0005 {
		return Voice{}, ErrSampleQuiet
	}
	reference := filepath.Join(work, "reference.wav")
	// Even loudness gives the same clone whether the recording was quiet or loud.
	if err := ffmpeg(ctx, engine.ffmpegPath(), "-i", sample, "-af", "loudnorm=I=-20:TP=-2:LRA=11", "-ac", "1", "-ar", fmt.Sprint(sampleRate), "-c:a", "pcm_s16le", reference); err != nil {
		return Voice{}, ErrBadSample
	}
	if err := engine.Encode(ctx, reference, filepath.Join(work, stateFile), language); err != nil {
		return Voice{}, err
	}
	if _, err := os.Stat(filepath.Join(work, stateFile)); err != nil {
		return Voice{}, fmt.Errorf("%w: no voice was produced", ErrFailed)
	}
	if err := os.Remove(upload); err != nil {
		return Voice{}, err
	}

	voice := Voice{ID: newVoiceID(), Name: name, Language: language, Seconds: seconds, CreatedAt: time.Now().UTC()}
	v.mu.Lock()
	defer v.mu.Unlock()
	final := filepath.Join(v.dir, voice.ID)
	if err := os.Rename(work, final); err != nil {
		return Voice{}, err
	}
	if err := v.writeVoice(voice); err != nil {
		_ = os.RemoveAll(final)
		return Voice{}, err
	}
	return voice, nil
}

func (e *Engine) ffmpegPath() string {
	if e.cfg.ffmpeg != "" {
		return e.cfg.ffmpeg
	}
	return "ffmpeg"
}

func writeLimited(path string, source io.Reader, limit int64) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	written, err := io.Copy(file, io.LimitReader(source, limit+1))
	if err != nil {
		return err
	}
	if written > limit {
		return ErrBadSample
	}
	return nil
}

// ffmpeg decodes untrusted audio, so it may only read the file it is given, in
// the container formats a recording can plausibly be in.
func ffmpeg(ctx context.Context, binary string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	// The whitelist options apply to the input that follows them, so they come before -i.
	full := append([]string{"-nostdin", "-v", "error", "-y", "-protocol_whitelist", "file", "-format_whitelist", "wav,mp3,flac,ogg,mov,matroska,webm,aac,aiff"}, args...)
	return exec.CommandContext(ctx, binary, full...).Run()
}

// inspectWAV reads a 16-bit mono WAV and returns its length in seconds and RMS loudness (0 to 1).
func inspectWAV(path string) (seconds, rms float64, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	data := bytes.Index(raw, []byte("data"))
	if len(raw) < 44 || string(raw[:4]) != "RIFF" || data < 0 || data+8 > len(raw) {
		return 0, 0, errors.New("not a WAV file")
	}
	pcm := raw[data+8:]
	if declared := int(binary.LittleEndian.Uint32(raw[data+4:])); declared < len(pcm) {
		pcm = pcm[:declared]
	}
	samples := len(pcm) / 2
	if samples == 0 {
		return 0, 0, errors.New("empty recording")
	}
	var sum float64
	for i := 0; i < samples; i++ {
		value := float64(int16(binary.LittleEndian.Uint16(pcm[i*2:]))) / 32768
		sum += value * value
	}
	return float64(samples) / sampleRate, math.Sqrt(sum / float64(samples)), nil
}
