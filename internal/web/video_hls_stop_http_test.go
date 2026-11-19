package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mochil/internal/video"
)

type hlsStopProbe struct{}

func (hlsStopProbe) Probe(string) (video.ProbeResult, error) {
	return video.ProbeResult{Duration: 125, Video: []video.VideoStream{{Index: 0}}, Audio: []video.AudioStream{{Index: 0}}}, nil
}

// Before this fix, closing the video player client-side had no way to tell
// the server to stop transcoding: the ffmpeg encoder kept running (burning
// CPU and disk) until the passive 75-second idle timeout eventually noticed
// nobody was fetching segments. DELETE on the playlist URL lets the client's
// explicit "I stopped watching" signal reach the encoder immediately.
func TestVideoHLSDeleteStopsRunningEncoderImmediately(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "movie.mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, filepath.Join(t.TempDir(), "cache.json"))
	store.SetProber(hlsStopProbe{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "encoder.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	hls, err := video.NewHLSStore(filepath.Join(t.TempDir(), "hls"), script, 1<<20, 2, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer hls.Close()
	handler, err := NewMulti(MultiConfig{AllowOpen: true, HLS: hls}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	playlistPath := "/api/video-hls/movies/" + item.ID + "/0/high/index.m3u8"
	opened := httptest.NewRecorder()
	handler.ServeHTTP(opened, httptest.NewRequest(http.MethodGet, playlistPath, nil))
	if opened.Code != http.StatusOK {
		t.Fatalf("start playlist = %d %s", opened.Code, opened.Body.String())
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if states := hls.Jobs(); len(states) > 0 && states[0].Status == "running" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	stopped := httptest.NewRecorder()
	handler.ServeHTTP(stopped, httptest.NewRequest(http.MethodDelete, playlistPath, nil))
	if stopped.Code != http.StatusNoContent {
		t.Fatalf("stop response = %d %s", stopped.Code, stopped.Body.String())
	}
	deadline = time.Now().Add(2 * time.Second)
	var finalStatus string
	for time.Now().Before(deadline) {
		for _, state := range hls.Jobs() {
			finalStatus = state.Status
		}
		if finalStatus == "canceled" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if finalStatus != "canceled" {
		t.Fatalf("job status = %q, want canceled", finalStatus)
	}

	// A second stop (already stopped, or an unknown/never-started session)
	// is a harmless no-op, not an error: the client may race a close against
	// an encoder that already finished or never started.
	again := httptest.NewRecorder()
	handler.ServeHTTP(again, httptest.NewRequest(http.MethodDelete, playlistPath, nil))
	if again.Code != http.StatusNoContent {
		t.Fatalf("second stop response = %d %s", again.Code, again.Body.String())
	}
	unknown := httptest.NewRecorder()
	handler.ServeHTTP(unknown, httptest.NewRequest(http.MethodDelete, "/api/video-hls/movies/does-not-exist/0/high/index.m3u8", nil))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown item stop response = %d %s", unknown.Code, unknown.Body.String())
	}
}

// navigator.sendBeacon (the only reliable signal on tab close / navigating
// away, since it survives page unload where a plain fetch would not) can
// only send POST, never DELETE — the server must accept both as the same
// stop action.
func TestVideoHLSPostAlsoStopsForSendBeacon(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "movie.mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, filepath.Join(t.TempDir(), "cache.json"))
	store.SetProber(hlsStopProbe{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "encoder.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	hls, err := video.NewHLSStore(filepath.Join(t.TempDir(), "hls"), script, 1<<20, 2, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer hls.Close()
	handler, err := NewMulti(MultiConfig{AllowOpen: true, HLS: hls}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	playlistPath := "/api/video-hls/movies/" + item.ID + "/0/high/index.m3u8"
	opened := httptest.NewRecorder()
	handler.ServeHTTP(opened, httptest.NewRequest(http.MethodGet, playlistPath, nil))
	if opened.Code != http.StatusOK {
		t.Fatalf("start playlist = %d %s", opened.Code, opened.Body.String())
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if states := hls.Jobs(); len(states) > 0 && states[0].Status == "running" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	stopped := httptest.NewRecorder()
	handler.ServeHTTP(stopped, httptest.NewRequest(http.MethodPost, playlistPath, nil))
	if stopped.Code != http.StatusNoContent {
		t.Fatalf("beacon stop response = %d %s", stopped.Code, stopped.Body.String())
	}
	deadline = time.Now().Add(2 * time.Second)
	var finalStatus string
	for time.Now().Before(deadline) {
		for _, state := range hls.Jobs() {
			finalStatus = state.Status
		}
		if finalStatus == "canceled" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if finalStatus != "canceled" {
		t.Fatalf("job status = %q, want canceled", finalStatus)
	}
}
