package web

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/narration"
)

// A stand-in for the Pocket runtime: an "interpreter" that ignores the worker
// script it is given and answers the protocol with silence.
func fakeNarrationRuntime(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	worker := filepath.Join(dir, "worker.py")
	script := `
import sys, json, base64
print(json.dumps({"ready": True, "sample_rate": 8000}), flush=True)
for line in sys.stdin:
    r = json.loads(line)
    if r.get("op") == "encode":
        open(r["state"], "wb").write(b"state")
        print(json.dumps({"id": r["id"], "ok": True}), flush=True)
        continue
    pcm = bytes(1600)
    if r.get("stream"):
        for _ in range(2):
            print(json.dumps({"id": r["id"], "more": True, "sample_rate": 8000, "pcm": base64.b64encode(pcm).decode()}), flush=True)
        print(json.dumps({"id": r["id"], "ok": True, "sample_rate": 8000}), flush=True)
        continue
    print(json.dumps({"id": r["id"], "ok": True, "sample_rate": 8000, "pcm": base64.b64encode(pcm).decode()}), flush=True)
`
	if err := os.WriteFile(worker, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(dir, "python")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nexec python3 -u "+worker+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return wrapper
}

func narrationServer(t *testing.T, modules map[string]bool, config narration.Config) *multiServer {
	t.Helper()
	handler, err := NewMulti(MultiConfig{AllowOpen: true, Modules: modules, Narration: config, NarrationVoices: filepath.Join(t.TempDir(), "voices"), ReadingState: filepath.Join(t.TempDir(), "reading.json")},
		nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler.(*managedHandler).Handler.(*multiServer)
}

func TestNarrationRoutesAreGoneWhenTheModuleIsOff(t *testing.T) {
	m := narrationServer(t, map[string]bool{"narration": false}, narration.Config{})
	for _, path := range []string{"/api/narration/status", "/api/narration/speak"} {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s with the module off: %d", path, w.Code)
		}
	}
	if m.narration != nil {
		t.Fatal("no engine may exist while the module is off")
	}
}

func TestNarrationWithoutARuntimeSaysSo(t *testing.T) {
	m := narrationServer(t, map[string]bool{"narration": true}, narration.Config{})
	status := httptest.NewRecorder()
	m.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/narration/status", nil))
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"available":false`) {
		t.Fatalf("status: %d %s", status.Code, status.Body.String())
	}
	speak := httptest.NewRecorder()
	m.ServeHTTP(speak, httptest.NewRequest(http.MethodPost, "/api/narration/speak", strings.NewReader(`{"text":"Hello."}`)))
	if speak.Code != http.StatusServiceUnavailable {
		t.Fatalf("speak without a runtime: %d", speak.Code)
	}
}

func TestNarrationSpeaksThroughTheRuntime(t *testing.T) {
	m := narrationServer(t, map[string]bool{"narration": true}, narration.Config{Python: fakeNarrationRuntime(t)})
	t.Cleanup(m.narration.Close)
	post := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/narration/speak", strings.NewReader(body)))
		return w
	}
	ok := post(`{"text":"The lantern swung.","voice":"cosette"}`)
	if ok.Code != http.StatusOK || ok.Header().Get("Content-Type") != "audio/wav" || !strings.HasPrefix(ok.Body.String(), "RIFF") {
		t.Fatalf("speak: %d %q", ok.Code, ok.Header().Get("Content-Type"))
	}
	if ok.Header().Get("X-Audio-Seconds") != "0.10" {
		t.Fatalf("duration header %q", ok.Header().Get("X-Audio-Seconds"))
	}
	for body, want := range map[string]int{
		`{"text":"Hi.","voice":"../../etc/passwd"}`: http.StatusBadRequest,
		`{"text":""}`: http.StatusBadRequest,
		`not json`:    http.StatusBadRequest,
		`{"text":"` + strings.Repeat("a", narration.MaxTextBytes+1) + `"}`: http.StatusRequestEntityTooLarge,
	} {
		if got := post(body).Code; got != want {
			t.Errorf("%.30q: got %d, want %d", body, got, want)
		}
	}
	status := httptest.NewRecorder()
	m.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/narration/status", nil))
	if !strings.Contains(status.Body.String(), `"available":true`) || !strings.Contains(status.Body.String(), `"cosette"`) {
		t.Fatalf("status: %s", status.Body.String())
	}
}

// fakeFFmpeg answers every decode with a ten second, clearly audible recording.
func fakeFFmpeg(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	samples := 24000 * 10
	pcm := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		value := int16(3000)
		if i%2 == 1 {
			value = -3000
		}
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(value))
	}
	source := filepath.Join(dir, "recording.wav")
	if err := os.WriteFile(source, narration.Audio{SampleRate: 24000, PCM: pcm}.WAV(), 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nfor last; do :; done\ncp '" + source + "' \"$last\"\n"
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func upload(t *testing.T, name, language string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	_ = form.WriteField("name", name)
	_ = form.WriteField("language", language)
	file, _ := form.CreateFormFile("file", "sample.webm")
	_, _ = file.Write([]byte("recorded audio"))
	_ = form.Close()
	return body, form.FormDataContentType()
}

func TestSavedVoicesCanBeMadeUsedRenamedAndRemoved(t *testing.T) {
	fakeFFmpeg(t)
	m := narrationServer(t, map[string]bool{"narration": true}, narration.Config{Python: fakeNarrationRuntime(t)})
	t.Cleanup(m.narration.Close)
	do := func(method, path string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
		if body == nil {
			body = &bytes.Buffer{}
		}
		request := httptest.NewRequest(method, path, body)
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		w := httptest.NewRecorder()
		m.ServeHTTP(w, request)
		return w
	}

	body, contentType := upload(t, "Amelia Tyler", "dutch")
	created := do(http.MethodPost, "/api/narration/voices", body, contentType)
	var voice narrationVoice
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &voice) != nil || voice.Name != "Amelia Tyler" || voice.Language != "dutch" || !voice.Custom {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}

	status := do(http.MethodGet, "/api/narration/status", nil, "").Body.String()
	if !strings.Contains(status, voice.ID) || !strings.Contains(status, `"daan"`) || strings.Index(status, voice.ID) > strings.Index(status, `"id":"alba"`) {
		t.Fatalf("saved voices should come first in the list: %s", status)
	}

	speak := do(http.MethodPost, "/api/narration/speak", bytes.NewBufferString(`{"text":"Hallo daar.","voice":"`+voice.ID+`"}`), "")
	if speak.Code != http.StatusOK || !strings.HasPrefix(speak.Body.String(), "RIFF") {
		t.Fatalf("speaking with a saved voice: %d", speak.Code)
	}
	if sample := do(http.MethodGet, "/api/narration/voices/"+voice.ID+"/sample", nil, ""); sample.Code != http.StatusOK || sample.Header().Get("Content-Type") != "audio/wav" {
		t.Fatalf("sample: %d", sample.Code)
	}

	if renamed := do(http.MethodPatch, "/api/narration/voices/"+voice.ID, bytes.NewBufferString(`{"name":"Amelia"}`), ""); renamed.Code != http.StatusNoContent {
		t.Fatalf("rename: %d", renamed.Code)
	}
	if !strings.Contains(do(http.MethodGet, "/api/narration/status", nil, "").Body.String(), `"Amelia"`) {
		t.Fatal("the new name is not listed")
	}
	if bad := do(http.MethodPatch, "/api/narration/voices/"+voice.ID, bytes.NewBufferString(`{"name":"  "}`), ""); bad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("an empty name: %d", bad.Code)
	}

	if gone := do(http.MethodDelete, "/api/narration/voices/"+voice.ID, nil, ""); gone.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", gone.Code)
	}
	for _, check := range []*httptest.ResponseRecorder{
		do(http.MethodPost, "/api/narration/speak", bytes.NewBufferString(`{"text":"Hi.","voice":"`+voice.ID+`"}`), ""),
	} {
		if check.Code != http.StatusBadRequest {
			t.Fatalf("a deleted voice must not speak: %d", check.Code)
		}
	}
	if again := do(http.MethodDelete, "/api/narration/voices/"+voice.ID, nil, ""); again.Code != http.StatusNotFound {
		t.Fatalf("deleting twice: %d", again.Code)
	}
	for _, path := range []string{"/api/narration/voices/../../etc/passwd/sample", "/api/narration/voices/zzzz/sample"} {
		if got := do(http.MethodGet, path, nil, "").Code; got != http.StatusNotFound {
			t.Errorf("%s: %d", path, got)
		}
	}
}

func TestCloningRefusesAnUploadWithoutAName(t *testing.T) {
	fakeFFmpeg(t)
	m := narrationServer(t, map[string]bool{"narration": true}, narration.Config{Python: fakeNarrationRuntime(t)})
	t.Cleanup(m.narration.Close)
	body, contentType := upload(t, "   ", "english")
	request := httptest.NewRequest(http.MethodPost, "/api/narration/voices", body)
	request.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, request)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestWarmAnswersAtOnceAndLoadsTheVoiceInTheBackground(t *testing.T) {
	m := narrationServer(t, map[string]bool{"narration": true}, narration.Config{Python: fakeNarrationRuntime(t)})
	t.Cleanup(m.narration.Close)
	warm := func(body string) int {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/narration/warm", strings.NewReader(body)))
		return w.Code
	}
	if code := warm(`{"voice":"cosette"}`); code != http.StatusAccepted {
		t.Fatalf("warm: %d", code)
	}
	for i := 0; i < 100 && !m.narration.Loaded(); i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if !m.narration.Loaded() {
		t.Fatal("the worker should have been started by the warm-up")
	}
	if code := warm(`{"voice":"../../etc/passwd"}`); code != http.StatusAccepted {
		t.Fatalf("an unknown voice is ignored, not an error: %d", code)
	}
}

func TestNarrationStreamsRawAudioAsItIsMade(t *testing.T) {
	m := narrationServer(t, map[string]bool{"narration": true}, narration.Config{Python: fakeNarrationRuntime(t)})
	t.Cleanup(m.narration.Close)
	post := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/narration/stream", strings.NewReader(body)))
		return w
	}
	ok := post(`{"text":"The lantern swung.","voice":"cosette"}`)
	if ok.Code != http.StatusOK || ok.Header().Get("X-Sample-Rate") != "8000" || ok.Header().Get("Content-Type") != "application/octet-stream" || ok.Body.Len() != 3200 {
		t.Fatalf("stream: %d %q rate=%q bytes=%d", ok.Code, ok.Header().Get("Content-Type"), ok.Header().Get("X-Sample-Rate"), ok.Body.Len())
	}
	if post(`{"text":"Hi.","voice":"../../etc/passwd"}`).Code != http.StatusBadRequest || post(`{"text":""}`).Code != http.StatusBadRequest {
		t.Fatal("bad requests are refused before any audio")
	}
}
