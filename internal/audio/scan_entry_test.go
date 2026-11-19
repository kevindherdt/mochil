package audio

import (
	"errors"
	"io/fs"
	"testing"
)

type countingAudioEntry struct {
	name  string
	calls int
	err   error
}

func (e *countingAudioEntry) Name() string               { return e.name }
func (e *countingAudioEntry) IsDir() bool                { return false }
func (e *countingAudioEntry) Type() fs.FileMode          { return 0 }
func (e *countingAudioEntry) Info() (fs.FileInfo, error) { e.calls++; return nil, e.err }

func TestAudioScanSkipsMetadataForNonMedia(t *testing.T) {
	for _, name := range []string{"cover.jpg", "lyrics.lrc", "notes.txt", "._Song.FLAC", "metadata.json", "audio.mp3.tmp"} {
		entry := &countingAudioEntry{name: name, err: errors.New("should not stat")}
		info, err := audioScanInfo(entry)
		if info != nil || err != nil || entry.calls != 0 {
			t.Fatalf("%s: info=%v err=%v calls=%d", name, info, err, entry.calls)
		}
	}
}

func TestAudioScanStillStatsAllSupportedFormats(t *testing.T) {
	for extension := range formats {
		failure := errors.New("stat failed")
		entry := &countingAudioEntry{name: "Song" + extension, err: failure}
		_, err := audioScanInfo(entry)
		if !errors.Is(err, failure) || entry.calls != 1 {
			t.Fatalf("%s: err=%v calls=%d", extension, err, entry.calls)
		}
	}
	entry := &countingAudioEntry{name: "Song.FLAC"}
	_, _ = audioScanInfo(entry)
	if entry.calls != 1 {
		t.Fatal("uppercase format skipped")
	}
}
