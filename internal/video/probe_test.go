package video

import (
	"os"
	"path/filepath"
	"testing"
)

type countingProber struct{ calls int }

func (p *countingProber) Probe(string) (ProbeResult, error) {
	p.calls++
	return ProbeResult{Duration: 90, Video: []VideoStream{{Index: 0, Codec: "h264"}}, Audio: []AudioStream{{Index: 1, Codec: "aac", Language: "eng"}}, Subtitles: []SubtitleStream{{Index: 2, Codec: "subrip", Language: "eng", Text: true}}}, nil
}

func TestProbeMetadataPersistsAndIsReusedByRevision(t *testing.T) {
	root, cache := t.TempDir(), filepath.Join(t.TempDir(), "videos.json")
	if err := os.WriteFile(filepath.Join(root, "film.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstProbe := &countingProber{}
	first := NewStore("movies", root, cache)
	first.SetProber(firstProbe)
	if _, err := first.Scan(); err != nil {
		t.Fatal(err)
	}
	if firstProbe.calls != 1 || first.Items()[0].Duration != 90 || len(first.Items()[0].Subtitles) != 1 {
		t.Fatalf("first probe = %d %#v", firstProbe.calls, first.Items())
	}
	secondProbe := &countingProber{}
	second := NewStore("movies", root, cache)
	second.SetProber(secondProbe)
	stats, err := second.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if secondProbe.calls != 0 || stats.Reused != 1 || second.Items()[0].Audio[0].Language != "eng" {
		t.Fatalf("cached probe = calls %d stats %#v items %#v", secondProbe.calls, stats, second.Items())
	}
}
