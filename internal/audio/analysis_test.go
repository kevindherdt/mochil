package audio

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseAnalysisUsesFinalSummary(t *testing.T) {
	result, err := parseAnalysis("I: -70.0 LUFS\nPeak: -9.0 dBFS\nSummary:\n  I: -20.5 LUFS\n  Peak: -1.0 dBFS\n")
	if err != nil {
		t.Fatal(err)
	}
	if result.GainDB != 2.5 || math.Abs(result.Peak-math.Pow(10, -1.0/20)) > 1e-9 {
		t.Fatalf("analysis = %#v", result)
	}
}

func TestApplyAnalysisPreservesEmbeddedValuesAndRejectsStaleResults(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "cache.json")
	store := NewStore("music", root, cache)
	now := time.Now()
	first, _ := Identify("music", "one.flac", 10, now)
	second, _ := Identify("music", "two.flac", 10, now)
	second.ReplayTrack = -3
	second.PeakTrack = .8
	store.publish([]Item{first, second})
	updated, err := store.ApplyAnalysis(map[string]Analysis{first.ID: {Revision: first.Revision, GainDB: 2, Peak: .9}, second.ID: {Revision: second.Revision, GainDB: 4, Peak: .95}})
	if err != nil {
		t.Fatal(err)
	}
	if updated != 1 {
		t.Fatalf("updated = %d", updated)
	}
	got, _ := store.Item(first.ID)
	if got.ReplayTrack != 2 || got.PeakTrack != .9 {
		t.Fatalf("first = %#v", got)
	}
	got, _ = store.Item(second.ID)
	if got.ReplayTrack != -3 || got.PeakTrack != .8 {
		t.Fatalf("embedded values changed: %#v", got)
	}
	if changed, err := store.ApplyAnalysis(map[string]Analysis{first.ID: {Revision: "stale", GainDB: 8, Peak: 1}}); err != nil || changed != 0 {
		t.Fatalf("stale changed=%d err=%v", changed, err)
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatal(err)
	}
}
