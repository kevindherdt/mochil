package web

import (
	"errors"
	"io"
	"net/http"

	"mochil/internal/narration"
)

// One recording is recognised at a time; speech recognition and speech synthesis share the same few CPU cores.
var transcribeGate = make(chan struct{}, 1)

// narrationTranscribe turns a short recording (the request body) into text. Administrators only: it is the voice
// input of the Hermes chat.
func (m *multiServer) narrationTranscribe(w http.ResponseWriter, r *http.Request) {
	if m.narration == nil || !m.narration.CanTranscribe() {
		http.Error(w, "Speech recognition is not set up on this server.", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, narration.MaxTranscribeBytes)
	recording, err := io.ReadAll(r.Body)
	if err != nil || len(recording) == 0 {
		http.Error(w, "Send a recording.", http.StatusBadRequest)
		return
	}
	select {
	case transcribeGate <- struct{}{}:
		defer func() { <-transcribeGate }()
	case <-r.Context().Done():
		return
	}
	text, err := m.narration.Transcribe(r.Context(), recording)
	w.Header().Set("Cache-Control", "no-store")
	switch {
	case errors.Is(err, narration.ErrNoSpeech):
		writeJSON(w, http.StatusOK, map[string]any{"text": "", "no_speech": true})
	case errors.Is(err, narration.ErrRecognitionUnavailable):
		http.Error(w, "Speech recognition is not set up on this server.", http.StatusServiceUnavailable)
	case err != nil:
		http.Error(w, "Could not understand the recording.", http.StatusBadGateway)
	default:
		writeJSON(w, http.StatusOK, map[string]any{"text": text})
	}
}
