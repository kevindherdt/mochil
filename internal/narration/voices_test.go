package narration

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wavBytes makes a mono 24 kHz recording of a quiet or loud tone.
func wavBytes(seconds float64, amplitude int16) []byte {
	samples := int(seconds * sampleRate)
	pcm := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		value := amplitude
		if i%2 == 1 {
			value = -amplitude
		}
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(value))
	}
	return Audio{SampleRate: sampleRate, PCM: pcm}.WAV()
}

// fakeFFmpeg stands in for ffmpeg: whatever it is asked to decode, it writes the
// given recording to its last argument.
func fakeFFmpeg(t *testing.T, recording []byte) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "recording.wav")
	if err := os.WriteFile(source, recording, 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "ffmpeg")
	body := "#!/bin/sh\nfor last; do :; done\ncp '" + source + "' \"$last\"\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return script
}

func cloneEngine(t *testing.T, recording []byte) *Engine {
	e := newFake(t, 60)
	e.cfg.ffmpeg = fakeFFmpeg(t, recording)
	return e
}

func TestCloneMakesAUsableVoice(t *testing.T) {
	voices := NewVoices(filepath.Join(t.TempDir(), "voices"))
	engine := cloneEngine(t, wavBytes(10, 3000))
	voice, err := voices.Clone(context.Background(), engine, strings.NewReader("recorded audio"), "  Amelia   Tyler ", "dutch")
	if err != nil {
		t.Fatal(err)
	}
	if voice.Name != "Amelia Tyler" || voice.Language != "dutch" || voice.Seconds < 9.9 || !voiceID.MatchString(voice.ID) {
		t.Fatalf("unexpected voice: %+v", voice)
	}
	state, ok := voices.StatePath(voice.ID)
	if !ok {
		t.Fatal("the voice has no state")
	}
	if raw, _ := os.ReadFile(state); string(raw) != "state:dutch" {
		t.Fatalf("the model was not asked for the right language: %q", raw)
	}
	if _, ok := voices.SamplePath(voice.ID); !ok {
		t.Fatal("the original recording should be kept for listening back")
	}
	if list := voices.List(); len(list) != 1 || list[0].ID != voice.ID {
		t.Fatalf("list: %+v", list)
	}
	entries, _ := os.ReadDir(voices.dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			t.Fatalf("a working folder was left behind: %s", entry.Name())
		}
	}
}

func TestCloneRefusesBadRecordingsAndKeepsNothing(t *testing.T) {
	cases := map[string]struct {
		recording []byte
		name      string
		want      error
	}{
		"too short": {wavBytes(1, 3000), "A", ErrSampleLength},
		"too long":  {wavBytes(31, 3000), "A", ErrSampleLength},
		"silent":    {wavBytes(10, 1), "A", ErrSampleQuiet},
		"no name":   {wavBytes(10, 3000), "   ", ErrBadName},
		"not audio": {[]byte("not a wav"), "A", ErrBadSample},
	}
	for label, c := range cases {
		voices := NewVoices(filepath.Join(t.TempDir(), "voices"))
		_, err := voices.Clone(context.Background(), cloneEngine(t, c.recording), strings.NewReader("x"), c.name, "english")
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", label, err, c.want)
		}
		if len(voices.List()) != 0 {
			t.Errorf("%s: a voice was saved", label)
		}
		if entries, _ := os.ReadDir(voices.dir); len(entries) != 0 {
			t.Errorf("%s: left %d files behind", label, len(entries))
		}
	}
}

func TestCloneReportsWhenTheModelCannotClone(t *testing.T) {
	voices := NewVoices(t.TempDir())
	engine := cloneEngine(t, wavBytes(10, 3000))
	// The stand-in worker refuses to clone when this marker sits beside it.
	if err := os.WriteFile(filepath.Join(filepath.Dir(engine.cfg.command[2]), "noclone"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := voices.Clone(context.Background(), engine, strings.NewReader("x"), "A", "english"); !errors.Is(err, ErrCloningUnavailable) {
		t.Fatalf("got %v", err)
	}
	if len(voices.List()) != 0 {
		t.Fatal("nothing should be saved")
	}
}

func TestRenameDeleteAndIDsAreChecked(t *testing.T) {
	voices := NewVoices(t.TempDir())
	voice, err := voices.Clone(context.Background(), cloneEngine(t, wavBytes(10, 3000)), bytes.NewReader([]byte("x")), "First", "english")
	if err != nil {
		t.Fatal(err)
	}
	if err := voices.Rename(voice.ID, " Second  one "); err != nil {
		t.Fatal(err)
	}
	if got, _ := voices.Get(voice.ID); got.Name != "Second one" {
		t.Fatalf("rename: %+v", got)
	}
	if !errors.Is(voices.Rename(voice.ID, strings.Repeat("x", 81)), ErrBadName) {
		t.Fatal("an overlong name must be refused")
	}
	for _, id := range []string{"../../etc", "", "ABCDEF", strings.Repeat("g", 32)} {
		if _, ok := voices.StatePath(id); ok {
			t.Errorf("%q must not resolve", id)
		}
		if !errors.Is(voices.Delete(id), ErrNoSuchVoice) {
			t.Errorf("deleting %q must fail", id)
		}
	}
	if err := voices.Delete(voice.ID); err != nil {
		t.Fatal(err)
	}
	if len(voices.List()) != 0 {
		t.Fatal("the voice should be gone")
	}
}

func TestVoicesFromMochiAreReadAsTheyAre(t *testing.T) {
	dir := t.TempDir()
	id := strings.Repeat("ab", 16)
	if err := os.MkdirAll(filepath.Join(dir, id), 0o750); err != nil {
		t.Fatal(err)
	}
	// Mochi's voice.json has more fields, and older voices have no language.
	_ = os.WriteFile(filepath.Join(dir, id, voiceFile), []byte(`{"id":"`+id+`","name":"Amelia Tyler - BG","engine":"pocket","seconds":17.97,"created_at":"2026-10-22T12:13:20.959940315Z","loudness_normalized":true,"source_voice_id":"x"}`), 0o600)
	_ = os.WriteFile(filepath.Join(dir, id, stateFile), []byte("state"), 0o600)
	// A folder without a prepared state is not a usable voice.
	_ = os.MkdirAll(filepath.Join(dir, strings.Repeat("cd", 16)), 0o750)
	list := NewVoices(dir).List()
	if len(list) != 1 || list[0].Name != "Amelia Tyler - BG" || list[0].Language != "english" {
		t.Fatalf("list: %+v", list)
	}
}

func TestWithoutAVoiceFolderChangesFindNoVoice(t *testing.T) {
	var none *Voices
	if err := none.Rename("0123456789abcdef0123456789abcdef", "x"); !errors.Is(err, ErrNoSuchVoice) {
		t.Fatalf("rename: %v", err)
	}
	if err := none.Delete("0123456789abcdef0123456789abcdef"); !errors.Is(err, ErrNoSuchVoice) {
		t.Fatalf("delete: %v", err)
	}
}
