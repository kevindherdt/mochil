package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRewriteHLSPlaylistUsesAuthenticatedRouteBase(t *testing.T) {
	input := []byte("#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:6,\nseg00000.m4s\n#EXT-X-ENDLIST\n")
	got := string(RewriteHLSPlaylist(input, "/api/video-hls/anime/item/1/"))
	if !strings.Contains(got, `URI="/api/video-hls/anime/item/1/init.mp4"`) || !strings.Contains(got, "/api/video-hls/anime/item/1/seg00000.m4s") {
		t.Fatalf("rewritten playlist = %q", got)
	}
}

func TestRewriteHLSPlaylistVersionsCachedResources(t *testing.T) {
	input := []byte("#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:2,\nseg00003.m4s\n")
	got := string(RewriteHLSPlaylist(input, "/api/video-hls/movies/item/1/", "cache-key"))
	if !strings.Contains(got, `URI="/api/video-hls/movies/item/1/init.mp4?v=cache-key"`) || !strings.Contains(got, "/api/video-hls/movies/item/1/seg00003.m4s?v=cache-key") {
		t.Fatalf("versioned playlist = %q", got)
	}
}

func TestHLSSeekWaitsForPreviousEncoderToExit(t *testing.T) {
	flock, err := exec.LookPath("flock")
	if err != nil {
		t.Skip("flock is not installed")
	}
	root := t.TempDir()
	script := filepath.Join(root, "encoder.sh")
	lock := filepath.Join(root, "encoder.lock")
	overlap := filepath.Join(root, "overlap")
	body := fmt.Sprintf("#!/bin/sh\nexec 9>%q\nif ! %q -n 9; then echo overlap >>%q; exit 1; fi\nsleep 0.5\nexit 1\n", lock, flock, overlap)
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), script, 1<<20, 2, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	store.readyAfter = 2 * time.Second
	item := Item{LibraryID: "movies", ID: "seek", Revision: "one", Duration: 120, Audio: []AudioStream{{Index: 1}}}
	file, _ := os.Open(source)
	if _, err := store.Start(file, item, 1, HLSQualityHigh); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if states := store.Jobs(); len(states) > 0 && states[0].Status == "running" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	file, _ = os.Open(source)
	if err := store.EnsureSegment(file, item, 1, HLSQualityHigh, 20); err != nil {
		t.Fatal(err)
	}
	for _, state := range store.Jobs() {
		waitContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = store.pool.Wait(waitContext, state.Key)
		cancel()
	}
	if data, err := os.ReadFile(overlap); err == nil {
		t.Fatalf("encoders overlapped: %s", data)
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestHLSSeekPreservesManifestTimeline(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "encoder.sh")
	arguments := filepath.Join(root, "arguments")
	body := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" >%q\nexit 1\n", arguments)
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), script, 1<<20, 1, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "anime", ID: "resume", Revision: "one", Duration: 120, Audio: []AudioStream{{Index: 1}}}
	file, _ := os.Open(source)
	if _, err := store.start(file, item, 1, HLSQualityHigh, 48, true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if data, readErr := os.ReadFile(arguments); readErr == nil {
			encoded := string(data)
			if !strings.Contains(encoded, "-output_ts_offset\n96\n") {
				t.Fatalf("seeked HLS arguments do not preserve the 96-second timeline: %s", encoded)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("encoder arguments were not captured")
}

func TestHLSStartClearsIncompleteCacheBeforeEncoding(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "encoder.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), script, 1<<20, 1, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "anime", ID: "interrupted", Revision: "one", Duration: 120, Audio: []AudioStream{{Index: 1}}}
	directory := filepath.Join(store.root, store.Key(item, 1, HLSQualityHigh))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(directory, "seg00400.ts")
	if err := os.WriteFile(stale, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, _ := os.Open(source)
	if _, err := store.Start(file, item, 1, HLSQualityHigh); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale partial segment survived a fresh conversion: %v", err)
	}
}

func TestHLSCacheKeySeparatesQualityAndAudio(t *testing.T) {
	first, err := NewHLSStore(t.TempDir(), "true", 1<<20, 1, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	original, err := NewHLSStore(t.TempDir(), "true", 1<<20, 1, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer original.Close()
	item := Item{LibraryID: "tv", ID: "episode", Revision: "one"}
	if first.Key(item, 1, HLSQualityHigh) == first.Key(item, 2, HLSQualityHigh) || first.Key(item, 1, HLSQualityMaximum) == original.Key(item, 1, HLSQualityMaximum) {
		t.Fatal("audio and quality variants must have separate cache keys")
	}
	if first.Key(item, 1, HLSQualityHigh) == first.Key(item, 1, HLSQualityLow) {
		t.Fatal("named quality profiles must have separate cache keys")
	}
	if profile := first.profile(HLSQualityLow); profile.height != 480 || profile.videoBitrate != "1500k" || profile.audioBitrate != "80k" {
		t.Fatalf("low profile = %+v", profile)
	}
}

func TestHLSStoreConvertsShortVideo(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source.mp4")
	generate := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "color=c=black:s=320x180:r=10", "-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo", "-t", "0.5", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", source)
	if output, err := generate.CombinedOutput(); err != nil {
		t.Skipf("test media generation unavailable: %v: %s", err, output)
	}
	file, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), ffmpeg, 64<<20, 1, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "tv", ID: "episode", Revision: "one", Duration: .5, Video: []VideoStream{{Index: 0}}, Audio: []AudioStream{{Index: 1}}}
	key, err := store.Start(file, item, 1, HLSQualityLow)
	if err != nil {
		t.Fatal(err)
	}
	playlist := string(store.Playlist(item))
	if !strings.Contains(playlist, "#EXT-X-PLAYLIST-TYPE:VOD") || !strings.Contains(playlist, "#EXT-X-ENDLIST") {
		t.Fatalf("synthetic playlist = %q", playlist)
	}
	segment, mediaType, err := store.Open(key, "seg00000.ts")
	if err != nil {
		t.Fatal(err)
	}
	defer segment.Close()
	if mediaType != "video/mp2t" {
		t.Fatalf("segment type = %q", mediaType)
	}
}

func TestHLSPlaylistCoversFullDuration(t *testing.T) {
	store := &HLSStore{}
	playlist := string(store.Playlist(Item{Duration: 13}))
	if strings.Count(playlist, "#EXTINF:") != 7 || !strings.Contains(playlist, "#EXTINF:1.000000") {
		t.Fatalf("playlist does not cover duration: %q", playlist)
	}
}

func TestHLSFailureUsesRetryCooldown(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("not-video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), "false", 1<<20, 1, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "movies", ID: "broken", Revision: "one", Duration: 10, Audio: []AudioStream{{Index: 1}}}
	file, _ := os.Open(source)
	key, err := store.Start(file, item, 1, HLSQualityHigh)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Open(key, "seg00000.ts"); err == nil {
		t.Fatal("failed converter unexpectedly produced a segment")
	}
	file, _ = os.Open(source)
	if _, err := store.Start(file, item, 1, HLSQualityHigh); err == nil || !strings.Contains(err.Error(), "retry") {
		t.Fatalf("immediate retry was not cooled down: %v", err)
	}
}

func TestHLSSegmentPastTheEndStartsNoEncoder(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "encoder.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), script, 1<<20, 2, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "movies", ID: "short", Revision: "one", Duration: 120, Audio: []AudioStream{{Index: 1}}}
	file, _ := os.Open(source)
	if err := store.EnsureSegment(file, item, 1, HLSQualityHigh, 99999); err == nil {
		t.Fatal("a segment far past the end must be refused")
	}
	if len(store.Jobs()) != 0 {
		t.Fatalf("an encoder was started: %v", store.Jobs())
	}
}
