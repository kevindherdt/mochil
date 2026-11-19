package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mochil/internal/appconfig"
)

func TestHermesChatRelaysTheStreamWithTheKeyAndNamespacedSession(t *testing.T) {
	var gotAuth, gotSession, gotBody string
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			_, _ = io.WriteString(w, `{"status":"ok"}`)
			return
		}
		gotAuth, gotSession = r.Header.Get("Authorization"), r.Header.Get("X-Hermes-Session-Id")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"po\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"ng\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Username: "Kev", Hermes: appconfig.Hermes{URL: hermes.URL + "/", Key: "secret"}}}

	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"ping"}`)))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"po"`) || !strings.Contains(w.Body.String(), "[DONE]") {
		t.Fatalf("relay: %d %q", w.Code, w.Body.String())
	}
	if gotAuth != "Bearer secret" || gotSession != m.hermesSessionPrefix(httptest.NewRequest(http.MethodPost, "/api/hermes/chat", nil))+"c1234567890abcdef" {
		t.Fatalf("headers: %q %q", gotAuth, gotSession)
	}
	if !strings.Contains(gotBody, `"stream":true`) || !strings.Contains(gotBody, `"content":"ping"`) || strings.Contains(gotBody, "secret") {
		t.Fatalf("body: %s", gotBody)
	}

	status := httptest.NewRecorder()
	m.apiHermes(status, httptest.NewRequest(http.MethodGet, "/api/hermes/status", nil))
	if !strings.Contains(status.Body.String(), `"configured":true`) || !strings.Contains(status.Body.String(), `"reachable":true`) || strings.Contains(status.Body.String(), "secret") {
		t.Fatalf("status: %s", status.Body.String())
	}
}

func TestHermesChatRejectsBadInputAndMissingSetup(t *testing.T) {
	m := &multiServer{cfg: MultiConfig{Hermes: appconfig.Hermes{URL: "http://127.0.0.1:1", Key: "k"}}}
	for _, body := range []string{`{"session":"short","message":"hi"}`, `{"session":"c1234567890abcdef","message":"  "}`, `{"session":"../../etc/passwd-x","message":"hi"}`, `{"nope":1}`} {
		w := httptest.NewRecorder()
		m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(body)))
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d", body, w.Code)
		}
	}
	down := httptest.NewRecorder()
	m.apiHermes(down, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"hi"}`)))
	if down.Code != http.StatusBadGateway {
		t.Errorf("unreachable: got %d", down.Code)
	}
	unset := &multiServer{}
	w := httptest.NewRecorder()
	unset.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"hi"}`)))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("unset: got %d", w.Code)
	}
	if moduleForPath("/api/hermes/chat") != "hermes" || moduleForPath("/api/android/v1/hermes/chat") != "hermes" {
		t.Error("hermes routes must belong to the hermes module")
	}
}

func TestHermesIsAdministratorOnly(t *testing.T) {
	for _, path := range []string{"/hermes", "/api/hermes/chat", "/api/hermes/status", "/api/hermes/chats/x"} {
		if !administratorOnly(path, http.MethodGet) {
			t.Errorf("%s must be administrator only", path)
		}
	}
}

func TestHermesChatsAreListedOpenedRenamedAndDeletedPerAccount(t *testing.T) {
	var renamed, deleted, system string
	prefix := (&multiServer{cfg: MultiConfig{Username: "Kev"}}).hermesSessionPrefix(httptest.NewRequest(http.MethodGet, "/", nil))
	other := (&multiServer{cfg: MultiConfig{Username: "Other"}}).hermesSessionPrefix(httptest.NewRequest(http.MethodGet, "/", nil))
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/sessions" && r.Method == http.MethodGet:
			_, _ = fmt.Fprintf(w, `{"data":[
				{"id":"%sc1234567890abcdef","title":"Good morning","message_count":4,"last_active":10},
				{"id":"%sc2234567890abcdef","title":"","preview":"What should I read","message_count":2,"last_active":20},
				{"id":"%scempty0000000000","title":"Empty","message_count":0,"last_active":30},
				{"id":"%sc1234567890abcdef","title":"Someone else","message_count":3,"last_active":40},
				{"id":"20261103_discord","title":"Discord","message_count":3,"last_active":50}],"has_more":false}`, prefix, prefix, prefix, other)
		case strings.HasSuffix(r.URL.Path, "/messages"):
			_, _ = io.WriteString(w, `{"data":[{"role":"user","content":"hi"},{"role":"assistant","content":"","tool_calls":[{}]},{"role":"tool","content":"x"},{"role":"assistant","content":"hello"}]}`)
		case r.Method == http.MethodPatch:
			b, _ := io.ReadAll(r.Body)
			renamed = r.URL.Path + " " + string(b)
			_, _ = io.WriteString(w, `{}`)
		case r.Method == http.MethodDelete:
			deleted = r.URL.Path
			_, _ = io.WriteString(w, `{}`)
		case r.URL.Path == "/v1/chat/completions":
			b, _ := io.ReadAll(r.Body)
			system = string(b)
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Username: "Kev", Hermes: appconfig.Hermes{URL: hermes.URL, Key: "k", Name: "Mia", Persona: "You are {name}, warm."}}}

	list := httptest.NewRecorder()
	m.apiHermes(list, httptest.NewRequest(http.MethodGet, "/api/hermes/chats", nil))
	body := list.Body.String()
	if !strings.Contains(body, `"c2234567890abcdef"`) || strings.Index(body, "c2234567890abcdef") > strings.Index(body, "c1234567890abcdef") {
		t.Fatalf("not newest first: %s", body)
	}
	for _, bad := range []string{"Someone else", "Discord", "Empty"} {
		if strings.Contains(body, bad) {
			t.Fatalf("%q must not be listed: %s", bad, body)
		}
	}
	if !strings.Contains(body, "What should I read") {
		t.Fatalf("a chat without a title uses its first words: %s", body)
	}

	open := httptest.NewRecorder()
	m.apiHermes(open, httptest.NewRequest(http.MethodGet, "/api/hermes/chats/c1234567890abcdef", nil))
	if strings.Contains(open.Body.String(), "tool") || !strings.Contains(open.Body.String(), `"hello"`) || strings.Count(open.Body.String(), `"role"`) != 2 {
		t.Fatalf("messages: %s", open.Body.String())
	}

	rename := httptest.NewRecorder()
	m.apiHermes(rename, httptest.NewRequest(http.MethodPatch, "/api/hermes/chats/c1234567890abcdef", strings.NewReader(`{"title":"Plans"}`)))
	del := httptest.NewRecorder()
	m.apiHermes(del, httptest.NewRequest(http.MethodDelete, "/api/hermes/chats/c1234567890abcdef", nil))
	if !strings.Contains(renamed, prefix+"c1234567890abcdef") || !strings.Contains(renamed, `"title":"Plans"`) || deleted != "/api/sessions/"+prefix+"c1234567890abcdef" {
		t.Fatalf("rename %q delete %q", renamed, deleted)
	}
	bad := httptest.NewRecorder()
	m.apiHermes(bad, httptest.NewRequest(http.MethodDelete, "/api/hermes/chats/..%2F..%2Fetc", nil))
	if bad.Code != http.StatusNotFound {
		t.Fatalf("a bad chat id must be refused: %d", bad.Code)
	}

	chat := httptest.NewRecorder()
	m.apiHermes(chat, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"hi"}`)))
	if !strings.Contains(system, `"role":"system"`) || !strings.Contains(system, "You are Mia, warm.") {
		t.Fatalf("persona with the assistant's name was not sent: %s", system)
	}
}

func TestHermesConnectionTestUsesTheSavedKeyWhenBlank(t *testing.T) {
	var seen string
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		if r.URL.Path == "/v1/models" && seen != "Bearer saved" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Hermes: appconfig.Hermes{URL: "http://127.0.0.1:1", Key: "saved"}}}
	test := func(body string) string {
		w := httptest.NewRecorder()
		m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/test", strings.NewReader(body)))
		return w.Body.String()
	}
	if got := test(`{"url":"` + hermes.URL + `","key":""}`); !strings.Contains(got, `"ok":true`) {
		t.Fatalf("saved key: %s", got)
	}
	if got := test(`{"url":"` + hermes.URL + `","key":"wrong"}`); !strings.Contains(got, "refused the key") {
		t.Fatalf("wrong key: %s", got)
	}
	if got := test(`{"url":"http://127.0.0.1:1","key":"x"}`); !strings.Contains(got, "did not answer") {
		t.Fatalf("unreachable: %s", got)
	}
}

func TestHermesSpokenRepliesAskForSpokenStyle(t *testing.T) {
	var body string
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Hermes: appconfig.Hermes{URL: hermes.URL, Key: "k", Persona: "You are warm."}}}
	send := func(payload string) {
		m.apiHermes(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(payload)))
	}
	send(`{"session":"c1234567890abcdef","message":"hi","spoken":true}`)
	if !strings.Contains(body, "read aloud") || !strings.Contains(body, "You are warm.") {
		t.Fatalf("spoken style missing: %s", body)
	}
	send(`{"session":"c1234567890abcdef","message":"hi"}`)
	if strings.Contains(body, "read aloud") {
		t.Fatalf("typed chats must not get the spoken style: %s", body)
	}
}

func TestNarrationDefaultVoiceIsSavedAndOnlyKnownVoicesAreAccepted(t *testing.T) {
	path := t.TempDir() + "/default-voice.json"
	m := &multiServer{cfg: MultiConfig{NarrationDefaultVoiceState: path}, narrationVoices: nil}
	if got := m.narrationDefaultVoice(); got != defaultNarrationVoice {
		t.Fatalf("default before saving: %q", got)
	}
	put := func(body string) int {
		w := httptest.NewRecorder()
		m.narrationSetDefault(w, httptest.NewRequest(http.MethodPut, "/api/narration/default-voice", strings.NewReader(body)))
		return w.Code
	}
	if put(`{"voice":"nobody-here"}`) != http.StatusBadRequest || put(`{"voice":""}`) != http.StatusBadRequest {
		t.Fatal("an unknown or empty voice must be refused")
	}
	if put(`{"voice":"marius"}`) != http.StatusOK || m.narrationDefaultVoice() != "marius" {
		t.Fatalf("a built-in voice should be saved, got %q", m.narrationDefaultVoice())
	}
	if !administratorOnly("/api/narration/default-voice", http.MethodPut) {
		t.Fatal("choosing the default voice is for administrators")
	}
}

func TestHermesInterruptedReplyIsToldToTheModelAndTranscribeNeedsSetup(t *testing.T) {
	var body string
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Hermes: appconfig.Hermes{URL: hermes.URL, Key: "k"}}}
	m.apiHermes(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"hi","interrupted":true}`)))
	if !strings.Contains(body, "spoke over your previous reply") {
		t.Fatalf("the interruption was not passed on: %s", body)
	}
	w := httptest.NewRecorder()
	m.narrationTranscribe(w, httptest.NewRequest(http.MethodPost, "/api/narration/transcribe", strings.NewReader("audio")))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("without narration set up: %d", w.Code)
	}
	if !administratorOnly("/api/narration/transcribe", http.MethodPost) {
		t.Fatal("voice input is for administrators")
	}
}

func TestHermesChatSaysItIsStillWaitingWhileHermesIsSlow(t *testing.T) {
	quick, every := hermesQuickAnswer, hermesKeepAliveEvery
	hermesQuickAnswer, hermesKeepAliveEvery = 30*time.Millisecond, 40*time.Millisecond
	defer func() { hermesQuickAnswer, hermesKeepAliveEvery = quick, every }()
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // a slow first word
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Username: "Kev", Hermes: appconfig.Hermes{URL: hermes.URL + "/", Key: "secret"}}}
	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"ping"}`)))
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "event: hermes.waiting") || !strings.Contains(body, `"hi"`) || !strings.Contains(body, "[DONE]") {
		t.Fatalf("slow relay: %d %q", w.Code, body)
	}
	if strings.Index(body, "hermes.waiting") > strings.Index(body, `"hi"`) {
		t.Fatalf("the waiting signal must come before the answer: %q", body)
	}
}

func TestHermesChatReportsAFailureAfterTheStreamHasOpened(t *testing.T) {
	quick := hermesQuickAnswer
	hermesQuickAnswer = 20 * time.Millisecond
	defer func() { hermesQuickAnswer = quick }()
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Username: "Kev", Hermes: appconfig.Hermes{URL: hermes.URL + "/", Key: "secret"}}}
	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"ping"}`)))
	if !strings.Contains(w.Body.String(), "event: hermes.error") || !strings.Contains(w.Body.String(), "refused the request (500)") || strings.Contains(w.Body.String(), "boom") {
		t.Fatalf("late failure: %d %q", w.Code, w.Body.String())
	}
}

func memoryTestServer(t *testing.T, gotBody *string) (*multiServer, *httptest.Server) {
	t.Helper()
	hindsight := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/default/banks/lyana/memories/list":
			_, _ = io.WriteString(w, `{"total":3,"items":[`+
				`{"id":"a","text":"Kevin lives in Belgium | Where: Belgium | Involving: Kevin","fact_type":"world","mentioned_at":"2026-11-03T21:33:59+00:00"},`+
				`{"id":"b","text":"Kevin lives in Belgium.","fact_type":"world","date":"2026-11-03T21:30:00+00:00"},`+
				`{"id":"c","text":"Kevin is building Mochil.","fact_type":"experience"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/default/banks/lyana/memories":
			b, _ := io.ReadAll(r.Body)
			*gotBody = string(b)
			_, _ = io.WriteString(w, `{"success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(hindsight.Close)
	return &multiServer{cfg: MultiConfig{Username: "Kev", Hermes: appconfig.Hermes{URL: "http://hermes.invalid/", Key: "k", MemoryURL: hindsight.URL, MemoryBank: "lyana"}}}, hindsight
}

func TestHermesMemoriesListIsCleanedAndDeduplicated(t *testing.T) {
	var sent string
	m, _ := memoryTestServer(t, &sent)
	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodGet, "/api/hermes/memories", nil))
	var got struct {
		Enabled bool           `json:"enabled"`
		Total   int            `json:"total"`
		Items   []hermesMemory `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK {
		t.Fatalf("memories: %d %q %v", w.Code, w.Body.String(), err)
	}
	if !got.Enabled || got.Total != 3 || len(got.Items) != 2 {
		t.Fatalf("expected 2 distinct memories, got %+v", got)
	}
	if got.Items[0].Text != "Kevin lives in Belgium" || got.Items[0].Date == "" || strings.Contains(w.Body.String(), "Involving") {
		t.Fatalf("detail should be dropped: %+v", got.Items)
	}
}

func TestHermesRememberSavesTheTextAsAMemoryAndRejectsBadInput(t *testing.T) {
	var sent string
	m, _ := memoryTestServer(t, &sent)
	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/remember", strings.NewReader(`{"text":"  My sister is called Anna  "}`)))
	if w.Code != http.StatusOK || !strings.Contains(sent, `"content":"My sister is called Anna"`) || !strings.Contains(sent, `"async":true`) {
		t.Fatalf("remember: %d %q sent=%s", w.Code, w.Body.String(), sent)
	}
	for _, body := range []string{`{"text":"   "}`, `{"text":"` + strings.Repeat("x", 4001) + `"}`, `{"text":"a","extra":1}`} {
		bad := httptest.NewRecorder()
		m.apiHermes(bad, httptest.NewRequest(http.MethodPost, "/api/hermes/remember", strings.NewReader(body)))
		if bad.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %.30q, got %d", body, bad.Code)
		}
	}
}

func TestHermesMemoryIsOffUntilABankIsSetAndNeverLeaksTheAddress(t *testing.T) {
	m := &multiServer{cfg: MultiConfig{Username: "Kev", Hermes: appconfig.Hermes{URL: "http://hermes.invalid/", Key: "k"}}}
	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodGet, "/api/hermes/memories", nil))
	if !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Fatalf("memory should be off: %s", w.Body.String())
	}
	off := httptest.NewRecorder()
	m.apiHermes(off, httptest.NewRequest(http.MethodPost, "/api/hermes/remember", strings.NewReader(`{"text":"x"}`)))
	if off.Code != http.StatusServiceUnavailable {
		t.Fatalf("remember without memory: %d", off.Code)
	}
	var sent string
	m2, hindsight := memoryTestServer(t, &sent)
	hindsight.Close()
	down := httptest.NewRecorder()
	m2.apiHermes(down, httptest.NewRequest(http.MethodGet, "/api/hermes/memories", nil))
	if down.Code != http.StatusBadGateway || strings.Contains(down.Body.String(), "127.0.0.1") {
		t.Fatalf("unreachable: %d %s", down.Code, down.Body.String())
	}
}
