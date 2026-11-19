package audio

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type analysisStub struct{ calls int }

func (s *analysisStub) Analyze(_ context.Context, file *os.File) (Analysis, error) {
	s.calls++
	file.Close()
	return Analysis{GainDB: 1.5, Peak: .75}, nil
}

func TestAnalysisQueueAnalyzesOnlyMissingItems(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one.flac"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "two.flac"), []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("music", root, filepath.Join(root, "cache.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	items := store.Items()
	items[0].ReplayTrack = -2
	items[0].PeakTrack = .9
	store.publish(items)
	analyzer := &analysisStub{}
	queue := NewAnalysisQueue(analyzer, 1)
	defer queue.Close()
	key, err := queue.Submit(store)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state, _ := queue.pool.State(key)
		if state.Status == "ready" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if analyzer.calls != 1 {
		t.Fatalf("calls = %d", analyzer.calls)
	}
	state, _ := queue.pool.State(key)
	if state.Status != "ready" || state.ProgressCurrent != 1 || state.ProgressTotal != 1 {
		t.Fatalf("state = %#v", state)
	}
}
