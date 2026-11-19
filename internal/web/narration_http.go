package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mochil/internal/narration"
)

// narrationPresets are the voices that ship with Pocket TTS.
var narrationPresets = []narrationVoice{
	{ID: "alba", Name: "Alba", Language: "english"}, {ID: "cosette", Name: "Cosette", Language: "english"},
	{ID: "fantine", Name: "Fantine", Language: "english"}, {ID: "eponine", Name: "Éponine", Language: "english"},
	{ID: "azelma", Name: "Azelma", Language: "english"}, {ID: "jean", Name: "Jean", Language: "english"},
	{ID: "marius", Name: "Marius", Language: "english"}, {ID: "javert", Name: "Javert", Language: "english"},
	{ID: "daan", Name: "Daan", Language: "dutch"},
}

const (
	defaultNarrationVoice = "alba"
	// A reader asks for a few sentences ahead; more than this waiting means a stuck client.
	narrationQueueLimit = 8
)

var narrationQueue = make(chan struct{}, narrationQueueLimit)

type narrationStatus struct {
	Enabled   bool `json:"enabled"`
	Available bool             `json:"available"`
	Loaded    bool             `json:"loaded"`
	Voice     string           `json:"voice"`
	Voices    []narrationVoice `json:"voices"`
	// Listen is true when the server can also recognise speech (the Hermes chat's voice input).
	Listen bool `json:"listen"`
	// Manage is true when this person may make, rename or delete voices and set the default (administrators).
	Manage bool `json:"manage"`
}

type narrationVoice struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Language string  `json:"language"`
	Custom   bool    `json:"custom,omitempty"`
	Seconds  float64 `json:"seconds,omitempty"`
}

// narrationVoiceList is the saved voices first, then the built-in ones.
func (m *multiServer) narrationVoiceList() []narrationVoice {
	out := make([]narrationVoice, 0, len(narrationPresets)+8)
	for _, voice := range m.narrationVoices.List() {
		out = append(out, narrationVoice{ID: voice.ID, Name: voice.Name, Language: voice.Language, Custom: true, Seconds: voice.Seconds})
	}
	return append(out, narrationPresets...)
}

func (m *multiServer) narrationAPI(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/narration/")
	switch {
	case rest == "status" && r.Method == http.MethodGet:
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, narrationStatus{
			Enabled:   m.narrationEnabled(),
			Available: m.narration != nil && m.narration.Available(),
			Loaded:    m.narration != nil && m.narration.Loaded(),
			Voice:     m.narrationDefaultVoice(), Voices: m.narrationVoiceList(), Listen: m.narration != nil && m.narration.CanTranscribe(),
			Manage: m.isAdministrator(r),
		})
	case rest == "stream" && r.Method == http.MethodPost:
		m.narrationStream(w, r)
	case rest == "transcribe" && r.Method == http.MethodPost:
		m.narrationTranscribe(w, r)
	case rest == "default-voice" && r.Method == http.MethodPut:
		m.narrationSetDefault(w, r)
	case rest == "speak" && r.Method == http.MethodPost:
		m.narrationSpeak(w, r)
	case strings.HasPrefix(rest, "book/"):
		m.narrationBook(w, r, rest)
	case rest == "warm" && r.Method == http.MethodPost:
		m.narrationWarm(w, r)
	case rest == "voices" && r.Method == http.MethodPost:
		m.narrationClone(w, r)
	case strings.HasPrefix(rest, "voices/"):
		m.narrationVoice(w, r, strings.TrimPrefix(rest, "voices/"))
	default:
		http.NotFound(w, r)
	}
}

// narrationVoiceRef turns the voice a client asked for into what the model loads.
func (m *multiServer) narrationVoiceRef(id string) (key, language string, ok bool) {
	if id == "" {
		id = defaultNarrationVoice
	}
	for _, preset := range narrationPresets {
		if preset.ID == id {
			return preset.ID, preset.Language, true
		}
	}
	if voice, found := m.narrationVoices.Get(id); found {
		if path, has := m.narrationVoices.StatePath(id); has {
			return path, voice.Language, true
		}
	}
	return "", "", false
}

func (m *multiServer) narrationSpeak(w http.ResponseWriter, r *http.Request) {
	if m.narration == nil || !m.narration.Available() {
		http.Error(w, "Narration is not set up on this server.", http.StatusServiceUnavailable)
		return
	}
	var request struct {
		Text  string `json:"text"`
		Voice string `json:"voice"`
	}
	limitRequestBody(w, r, 16<<10, smallBodyBudget)
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Text == "" {
		http.Error(w, "Send the text to speak.", http.StatusBadRequest)
		return
	}
	key, language, ok := m.narrationVoiceRef(request.Voice)
	if !ok {
		http.Error(w, "Unknown voice.", http.StatusBadRequest)
		return
	}
	if !m.narrationAcquire(w) {
		return
	}
	defer m.narrationRelease()
	audio, err := m.narration.Speak(r.Context(), request.Text, key, language)
	if !m.narrationFailed(w, r, err) {
		return
	}
	wav := audio.WAV()
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Length", strconv.Itoa(len(wav)))
	w.Header().Set("X-Audio-Seconds", strconv.FormatFloat(audio.Seconds(), 'f', 2, 64))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(wav)
}

// narrationStream speaks one sentence as a stream of raw 16-bit little-endian mono audio, written as the model makes it
// (the sample rate is in X-Sample-Rate). It is the same speech as narrationSpeak, but the first sound arrives long
// before the sentence is finished.
func (m *multiServer) narrationStream(w http.ResponseWriter, r *http.Request) {
	if m.narration == nil || !m.narration.Available() {
		http.Error(w, "Narration is not set up on this server.", http.StatusServiceUnavailable)
		return
	}
	var request struct {
		Text  string `json:"text"`
		Voice string `json:"voice"`
	}
	limitRequestBody(w, r, 16<<10, smallBodyBudget)
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Text == "" {
		http.Error(w, "Send the text to speak.", http.StatusBadRequest)
		return
	}
	key, language, ok := m.narrationVoiceRef(request.Voice)
	if !ok {
		http.Error(w, "Unknown voice.", http.StatusBadRequest)
		return
	}
	if !m.narrationAcquire(w) {
		return
	}
	defer m.narrationRelease()
	flusher, _ := w.(http.Flusher)
	started := false
	err := m.narration.SpeakStream(r.Context(), request.Text, key, language, func(piece narration.Audio) error {
		if !started {
			started = true
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("X-Sample-Rate", strconv.Itoa(piece.SampleRate))
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Accel-Buffering", "no")
		}
		if _, err := w.Write(piece.PCM); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	})
	if err != nil && !started {
		m.narrationFailed(w, r, err)
	}
}

// narrationWarm loads the model and voice ahead of the first sentence. It answers
// at once and carries on in the background; a busy or missing engine is not an error.
func (m *multiServer) narrationWarm(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Voice string `json:"voice"`
	}
	limitRequestBody(w, r, 1<<10, smallBodyBudget)
	_ = json.NewDecoder(r.Body).Decode(&request)
	w.WriteHeader(http.StatusAccepted)
	if m.narration == nil || !m.narration.Available() {
		return
	}
	key, language, ok := m.narrationVoiceRef(request.Voice)
	if !ok {
		return
	}
	select {
	case narrationQueue <- struct{}{}:
	default:
		return
	}
	go func() {
		defer m.narrationRelease()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		contained("narration warm", func() { _ = m.narration.Warm(ctx, key, language) })
	}()
}

func (m *multiServer) narrationAcquire(w http.ResponseWriter) bool {
	select {
	case narrationQueue <- struct{}{}:
		return true
	default:
		http.Error(w, "Narration is busy.", http.StatusTooManyRequests)
		return false
	}
}

func (m *multiServer) narrationRelease() { <-narrationQueue }

// narrationFailed answers for a failed call and reports whether the caller should carry on.
func (m *multiServer) narrationFailed(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, narration.ErrTooLong):
		http.Error(w, "That is too long to read in one go.", http.StatusRequestEntityTooLarge)
	case errors.Is(err, narration.ErrUnavailable):
		http.Error(w, "Narration is not set up on this server.", http.StatusServiceUnavailable)
	case errors.Is(err, narration.ErrCloningUnavailable):
		http.Error(w, "This server does not have the voice-cloning model. See the narration setup notes.", http.StatusFailedDependency)
	case errors.Is(err, narration.ErrBadSample), errors.Is(err, narration.ErrSampleLength), errors.Is(err, narration.ErrSampleQuiet),
		errors.Is(err, narration.ErrBadName), errors.Is(err, narration.ErrTooManyVoices):
		http.Error(w, err.Error()+".", http.StatusUnprocessableEntity)
	case r.Context().Err() != nil:
	default:
		m.root.log.Warn("narration failed", "error", err)
		http.Error(w, "Narration failed.", http.StatusBadGateway)
	}
	return false
}

// narrationClone makes a saved voice from an uploaded or recorded sample.
func (m *multiServer) narrationClone(w http.ResponseWriter, r *http.Request) {
	if m.narration == nil || !m.narration.Available() || m.narrationVoices == nil {
		http.Error(w, "Narration is not set up on this server.", http.StatusServiceUnavailable)
		return
	}
	limitRequestBody(w, r, narration.MaxSampleBytes+(64<<10), uploadBodyBudget)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "Send the recording and a name.", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Send the recording and a name.", http.StatusBadRequest)
		return
	}
	defer file.Close()
	language := r.FormValue("language")
	if language == "" {
		language = "english"
	}
	if !m.narrationAcquire(w) {
		return
	}
	defer m.narrationRelease()
	voice, err := m.narrationVoices.Clone(r.Context(), m.narration, file, r.FormValue("name"), language)
	if !m.narrationFailed(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, narrationVoice{ID: voice.ID, Name: voice.Name, Language: voice.Language, Custom: true, Seconds: voice.Seconds})
}

// narrationVoice handles one saved voice: rename, delete, or listen to its recording.
func (m *multiServer) narrationVoice(w http.ResponseWriter, r *http.Request, rest string) {
	id, action, _ := strings.Cut(rest, "/")
	switch {
	case action == "sample" && r.Method == http.MethodGet:
		path, ok := m.narrationVoices.SamplePath(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.ServeFile(w, r, path)
	case action == "" && r.Method == http.MethodPatch:
		var request struct {
			Name string `json:"name"`
		}
		limitRequestBody(w, r, 4<<10, smallBodyBudget)
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			http.Error(w, "Send a name.", http.StatusBadRequest)
			return
		}
		m.narrationChange(w, m.narrationVoices.Rename(id, request.Name))
	case action == "" && r.Method == http.MethodDelete:
		m.narrationChange(w, m.narrationVoices.Delete(id))
	default:
		http.NotFound(w, r)
	}
}

func (m *multiServer) narrationChange(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, narration.ErrNoSuchVoice):
		http.Error(w, "No such voice.", http.StatusNotFound)
	case errors.Is(err, narration.ErrBadName):
		http.Error(w, err.Error()+".", http.StatusUnprocessableEntity)
	default:
		m.root.log.Warn("narration voice change failed", "error", err)
		http.Error(w, "That did not work.", http.StatusInternalServerError)
	}
}
