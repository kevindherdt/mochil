package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/audio"
)

func TestAudioCompatibilityHLSIsAdvertisedAndServed(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg unavailable")
	}
	root := t.TempDir()
	source := filepath.Join(root, "tone.wav")
	if output, err := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000", "-t", "0.5", source).CombinedOutput(); err != nil {
		t.Fatalf("generate: %v: %s", err, output)
	}
	store := audio.NewStore("music", root, filepath.Join(t.TempDir(), "cache.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	hls, err := audio.NewHLSStore(filepath.Join(t.TempDir(), "hls"), ffmpeg, 16<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, AudioHLS: hls}, []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/audio?library=music", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("catalogue = %d: %s", response.Code, response.Body.String())
	}
	var items []audioView
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !strings.Contains(items[0].CompatibilityURL, "/api/audio-hls/music/") {
		t.Fatalf("items = %#v", items)
	}
	playlist := httptest.NewRecorder()
	handler.ServeHTTP(playlist, httptest.NewRequest(http.MethodGet, items[0].CompatibilityURL, nil))
	if playlist.Code != http.StatusOK || !strings.Contains(playlist.Body.String(), "/seg00000.m4s") {
		t.Fatalf("playlist = %d: %s", playlist.Code, playlist.Body.String())
	}
	if !strings.Contains(playlist.Body.String(), "/high/seg00000.m4s") {
		t.Fatalf("legacy playlist did not resolve to the high profile: %s", playlist.Body.String())
	}
	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, strings.Replace(items[0].CompatibilityURL, "/index.m3u8", "/ultra/index.m3u8", 1), nil))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid quality status = %d", invalid.Code)
	}
	segmentURL := strings.Split(strings.Split(playlist.Body.String(), "seg00000.m4s")[0], "\n")
	segment := segmentURL[len(segmentURL)-1] + "seg00000.m4s"
	media := httptest.NewRecorder()
	handler.ServeHTTP(media, httptest.NewRequest(http.MethodGet, segment, nil))
	if media.Code != http.StatusOK || media.Header().Get("Content-Type") != "audio/iso.segment" || media.Body.Len() == 0 {
		t.Fatalf("segment = %d type=%q bytes=%d body=%s", media.Code, media.Header().Get("Content-Type"), media.Body.Len(), media.Body.String())
	}
}

// TestAudioHLSDistantSegmentIsPreparedOnDemand guards the seek path on a long
// track. The audio playlist lists fragmented-MP4 segments (seg00810.m4s), but
// the handler used to match segment names against the video pattern
// (seg00810.ts), so EnsureSegment was never called for audio: a request for a
// segment the first encoder run had not reached waited out the engine's 20 s
// timeout and failed with 503 instead of starting an encoder at that point of
// the track. A short track hides this because it is fully encoded in seconds.
func TestAudioHLSDistantSegmentIsPreparedOnDemand(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg unavailable")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe unavailable")
	}
	root := t.TempDir()
	source := filepath.Join(root, "long.wav")
	// 30 minutes of cheap audio (about 14 MB) = 900 two-second segments.
	if output, err := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000", "-ac", "1", "-c:a", "pcm_u8", "-t", "1800", source).CombinedOutput(); err != nil {
		t.Fatalf("generate: %v: %s", err, output)
	}
	store := audio.NewStore("music", root, filepath.Join(t.TempDir(), "cache.json"))
	prober, err := audio.NewFFProber(ffprobe, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store.SetProber(prober) // without a probe the item has no duration and the playlist lists a single segment
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	hls, err := audio.NewHLSStore(filepath.Join(t.TempDir(), "hls"), ffmpeg, 64<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, AudioHLS: hls}, []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/audio?library=music", nil))
	var items []audioView
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil || len(items) != 1 {
		t.Fatalf("catalogue: %v %s", err, response.Body.String())
	}
	playlist := httptest.NewRecorder()
	handler.ServeHTTP(playlist, httptest.NewRequest(http.MethodGet, items[0].CompatibilityURL, nil))
	if playlist.Code != http.StatusOK || !strings.Contains(playlist.Body.String(), "seg00810.m4s") {
		t.Fatalf("playlist = %d: %.300s", playlist.Code, playlist.Body.String())
	}
	prefix := strings.Split(strings.Split(playlist.Body.String(), "seg00000.m4s")[0], "\n")
	distant := prefix[len(prefix)-1] + "seg00810.m4s"
	// A seek to 90% of the track: nothing has encoded that far yet.
	started := time.Now()
	media := httptest.NewRecorder()
	handler.ServeHTTP(media, httptest.NewRequest(http.MethodGet, distant, nil))
	if media.Code != http.StatusOK || media.Body.Len() == 0 {
		t.Fatalf("distant segment = %d after %v (%d bytes): %s", media.Code, time.Since(started).Round(time.Millisecond), media.Body.Len(), media.Body.String())
	}
}
