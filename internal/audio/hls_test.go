package audio

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAudioHLSKeyTracksRevision(t *testing.T) {
	store, err := NewHLSStore(t.TempDir(), "true", 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "music", ID: "track", Revision: "one"}
	first := store.Key(item, HLSQualityHigh)
	item.Revision = "two"
	if first == store.Key(item, HLSQualityHigh) {
		t.Fatal("revision did not change key")
	}
}

func TestAudioHLSQualityProfilesHaveIndependentCacheKeys(t *testing.T) {
	store, err := NewHLSStore(t.TempDir(), "true", 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "music", ID: "track", Revision: "one"}
	if store.Key(item, HLSQualityHigh) == store.Key(item, HLSQualityMedium) || store.Key(item, HLSQualityMedium) == store.Key(item, HLSQualityLow) {
		t.Fatal("audio quality profiles shared a cache key")
	}
	if _, ok := ParseHLSQuality("ultra"); ok {
		t.Fatal("unknown audio quality was accepted")
	}
}

func TestAudioHLSConvertsShortTrack(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg unavailable")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source.wav")
	generate := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000", "-t", "0.5", source)
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate: %v: %s", err, output)
	}
	info, _ := os.Stat(source)
	item, ok := Identify("music", "source.wav", info.Size(), time.Now())
	if !ok {
		t.Fatal("source not identified")
	}
	item.Duration = .5
	store, err := NewHLSStore(filepath.Join(root, "hls"), ffmpeg, 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	file, _ := os.Open(source)
	key, err := store.Start(file, item, HLSQualityHigh)
	if err != nil {
		t.Fatal(err)
	}
	file, mediaType, err := store.Open(key, "seg00000.m4s")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if mediaType != "audio/iso.segment" {
		t.Fatalf("media type = %q", mediaType)
	}
	manifest := string(store.Playlist(item))
	if !strings.Contains(manifest, "seg00000.m4s") || !strings.Contains(manifest, "#EXT-X-ENDLIST") {
		t.Fatalf("manifest = %q", manifest)
	}
}
