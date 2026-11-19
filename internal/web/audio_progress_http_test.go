package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"mochil/internal/audio"
)

func TestAudioProgressCanBeReadForConflictSafeClients(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Track.mp3"), []byte("track"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	handler, err := NewMulti(
		MultiConfig{AudioState: filepath.Join(t.TempDir(), "audio-progress.json")},
		[]Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/audio-progress/music/" + item.ID
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, path, nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing progress=%d %s", missing.Code, missing.Body.String())
	}
	put := httptest.NewRecorder()
	handler.ServeHTTP(put, httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"position_seconds":42,"duration_seconds":120,"expected_revision":0}`)))
	if put.Code != http.StatusOK {
		t.Fatalf("put=%d %s", put.Code, put.Body.String())
	}
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, path, nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `"position_seconds":42`) || !strings.Contains(get.Body.String(), `"revision":1`) {
		t.Fatalf("get=%d %s", get.Code, get.Body.String())
	}
	equivalent := httptest.NewRecorder()
	handler.ServeHTTP(equivalent, httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"position_seconds":42.2,"duration_seconds":120,"expected_revision":0,"device_id":"stale-phone"}`)))
	if equivalent.Code != http.StatusOK || !strings.Contains(equivalent.Body.String(), `"position_seconds":42`) || !strings.Contains(equivalent.Body.String(), `"revision":1`) || strings.Contains(equivalent.Body.String(), `stale-phone`) {
		t.Fatalf("equivalent stale progress=%d %s", equivalent.Code, equivalent.Body.String())
	}
	passive := httptest.NewRecorder()
	handler.ServeHTTP(passive, httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"position_seconds":10,"duration_seconds":120,"expected_revision":1}`)))
	if passive.Code != http.StatusConflict {
		t.Fatalf("passive rewind=%d %s", passive.Code, passive.Body.String())
	}
	explicit := httptest.NewRecorder()
	handler.ServeHTTP(explicit, httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"position_seconds":10,"duration_seconds":120,"expected_revision":1,"explicit_seek":true}`)))
	if explicit.Code != http.StatusOK || !strings.Contains(explicit.Body.String(), `"position_seconds":10`) || !strings.Contains(explicit.Body.String(), `"revision":2`) {
		t.Fatalf("explicit rewind=%d %s", explicit.Code, explicit.Body.String())
	}
}

func TestAudioStatusPatchDoesNotReplaceNewerPlaybackPosition(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Episode.mp3"), []byte("track"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("podcasts", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	handler, err := NewMulti(MultiConfig{AllowOpen: true, AudioState: filepath.Join(t.TempDir(), "progress.json")}, []Library{{ID: "podcasts", Name: "Podcasts", Kind: "podcasts", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/audio-progress/podcasts/" + item.ID
	put := httptest.NewRecorder()
	handler.ServeHTTP(put, httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"position_seconds":125,"duration_seconds":600,"device_id":"phone"}`)))
	patch := httptest.NewRecorder()
	handler.ServeHTTP(patch, httptest.NewRequest(http.MethodPatch, path, strings.NewReader(`{"dismissed":true}`)))
	if patch.Code != http.StatusOK || !strings.Contains(patch.Body.String(), `"position_seconds":125`) || !strings.Contains(patch.Body.String(), `"device_id":"phone"`) || !strings.Contains(patch.Body.String(), `"dismissed":true`) || !strings.Contains(patch.Body.String(), `"revision":2`) {
		t.Fatalf("patch=%d %s", patch.Code, patch.Body.String())
	}
	continued := httptest.NewRecorder()
	handler.ServeHTTP(continued, httptest.NewRequest(http.MethodGet, "/api/audio-progress?continue=1&limit=5", nil))
	if continued.Code != http.StatusOK || strings.TrimSpace(continued.Body.String()) != "[]" {
		t.Fatalf("dismissed continue list=%d %s", continued.Code, continued.Body.String())
	}
}

func TestAudioProgressListsOnlyLatestMusicSession(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"First.mp3", "Second.mp3"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("track"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, AudioState: filepath.Join(t.TempDir(), "audio-progress.json")}, []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	items := store.Items()
	for index, item := range items {
		request := httptest.NewRequest(http.MethodPut, "/api/audio-progress/music/"+item.ID, strings.NewReader(`{"position_seconds":`+strconv.Itoa(index+1)+`,"duration_seconds":120}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("put=%d %s", response.Code, response.Body.String())
		}
		time.Sleep(1100 * time.Millisecond)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/audio-progress", nil))
	var listed []audioView
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &listed) != nil || len(listed) != 1 {
		t.Fatalf("list=%d %s", response.Code, response.Body.String())
	}
	if listed[0].ID != items[len(items)-1].ID {
		t.Fatalf("latest item=%q want %q", listed[0].ID, items[len(items)-1].ID)
	}
	continued := httptest.NewRecorder()
	handler.ServeHTTP(continued, httptest.NewRequest(http.MethodGet, "/api/audio-progress?continue=1&limit=5", nil))
	if continued.Code != http.StatusOK || strings.TrimSpace(continued.Body.String()) != "[]" {
		t.Fatalf("music must not enter long-form continue list=%d %s", continued.Code, continued.Body.String())
	}

	history := httptest.NewRecorder()
	handler.ServeHTTP(history, httptest.NewRequest(http.MethodGet, "/api/audio-progress?music_library=music&limit=8", nil))
	listed = nil
	if history.Code != http.StatusOK || json.Unmarshal(history.Body.Bytes(), &listed) != nil || len(listed) != 2 {
		t.Fatalf("history=%d %s", history.Code, history.Body.String())
	}
	if listed[0].ID != items[len(items)-1].ID || listed[1].ID != items[0].ID {
		t.Fatalf("history order=%q,%q", listed[0].ID, listed[1].ID)
	}
}

func TestAudioProgressMusicHistoryRejectsNonMusicLibrary(t *testing.T) {
	handler, err := NewMulti(
		MultiConfig{AudioState: filepath.Join(t.TempDir(), "audio-progress.json")},
		[]Library{{ID: "podcasts", Name: "Podcasts", Kind: "podcasts", Audio: audio.NewStore("podcasts", t.TempDir(), "")}},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/audio-progress?music_library=podcasts", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
