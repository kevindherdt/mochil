package narration

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var voiceID = regexp.MustCompile(`^[0-9a-f]{32}$`)

const (
	// MaxVoices keeps the folder from growing without bound.
	MaxVoices       = 200
	maxVoiceName    = 80
	stateFile       = "state.safetensors"
	sampleFile      = "sample.wav"
	voiceFile       = "voice.json"
	languageEnglish = "english"
	languageDutch   = "dutch"
)

var (
	ErrNoSuchVoice   = errors.New("no such voice")
	ErrTooManyVoices = errors.New("too many saved voices")
	ErrBadName       = errors.New("give the voice a name of up to 80 characters")
)

// Voice is a saved voice made from a recording.
type Voice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Language  string    `json:"language"`
	Seconds   float64   `json:"seconds,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Voices is a folder of saved voices, one subfolder each:
// voice.json, sample.wav (the recording) and state.safetensors (the prepared voice).
// The layout matches the one Mochi writes, so its voices can be copied in as they are.
type Voices struct {
	dir string
	mu  sync.Mutex
}

func NewVoices(dir string) *Voices { return &Voices{dir: dir} }

func ValidLanguage(language string) bool {
	return language == languageEnglish || language == languageDutch
}

func (v *Voices) path(id string) (string, bool) {
	if v == nil || !voiceID.MatchString(id) {
		return "", false
	}
	return filepath.Join(v.dir, id), true
}

func (v *Voices) read(id string) (Voice, bool) {
	dir, ok := v.path(id)
	if !ok {
		return Voice{}, false
	}
	raw, err := os.ReadFile(filepath.Join(dir, voiceFile))
	if err != nil {
		return Voice{}, false
	}
	var voice Voice
	if json.Unmarshal(raw, &voice) != nil || voice.ID != id {
		return Voice{}, false
	}
	if _, err := os.Stat(filepath.Join(dir, stateFile)); err != nil {
		return Voice{}, false
	}
	if !ValidLanguage(voice.Language) {
		voice.Language = languageEnglish // older voices did not record one
	}
	return voice, true
}

// List returns every usable saved voice, by name.
func (v *Voices) List() []Voice {
	if v == nil {
		return nil
	}
	entries, _ := os.ReadDir(v.dir)
	voices := make([]Voice, 0, len(entries))
	for _, entry := range entries {
		if voice, ok := v.read(entry.Name()); entry.IsDir() && ok {
			voices = append(voices, voice)
		}
	}
	sort.Slice(voices, func(i, j int) bool {
		a, b := strings.ToLower(voices[i].Name), strings.ToLower(voices[j].Name)
		if a != b {
			return a < b
		}
		return voices[i].ID < voices[j].ID
	})
	return voices
}

// Get returns one saved voice.
func (v *Voices) Get(id string) (Voice, bool) { return v.read(id) }

// StatePath is the prepared voice file the speech model loads.
func (v *Voices) StatePath(id string) (string, bool) {
	if _, ok := v.read(id); !ok {
		return "", false
	}
	dir, _ := v.path(id)
	return filepath.Join(dir, stateFile), true
}

// SamplePath is the original recording, for listening back.
func (v *Voices) SamplePath(id string) (string, bool) {
	if _, ok := v.read(id); !ok {
		return "", false
	}
	dir, _ := v.path(id)
	path := filepath.Join(dir, sampleFile)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

func cleanName(name string) (string, bool) {
	name = strings.Join(strings.Fields(name), " ")
	return name, name != "" && len([]rune(name)) <= maxVoiceName
}

func (v *Voices) Rename(id, name string) error {
	name, ok := cleanName(name)
	if !ok {
		return ErrBadName
	}
	if v == nil {
		return ErrNoSuchVoice
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	voice, found := v.read(id)
	if !found {
		return ErrNoSuchVoice
	}
	voice.Name = name
	return v.writeVoice(voice)
}

func (v *Voices) Delete(id string) error {
	if v == nil {
		return ErrNoSuchVoice
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, found := v.read(id); !found {
		return ErrNoSuchVoice
	}
	dir, _ := v.path(id)
	return os.RemoveAll(dir)
}

func (v *Voices) writeVoice(voice Voice) error {
	dir, _ := v.path(voice.ID)
	raw, err := json.Marshal(voice)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".voice-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, voiceFile))
}

func newVoiceID() string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	return hex.EncodeToString(raw[:])
}
