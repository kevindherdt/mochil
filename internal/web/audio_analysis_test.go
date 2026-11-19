package web

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mochil/internal/audio"
)

type webAnalysisStub struct{}

func (webAnalysisStub) Analyze(_ context.Context, file *os.File) (audio.Analysis, error) {
	file.Close()
	return audio.Analysis{GainDB: 2, Peak: .8}, nil
}

func TestAudioAnalysisEndpointQueuesAndPersistsMissingResults(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "track.flac"), []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("music", root, filepath.Join(root, "cache.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	queue := audio.NewAnalysisQueue(webAnalysisStub{}, 1)
	handler, err := NewMulti(MultiConfig{AllowOpen: true, AudioAnalysis: queue}, []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/audio-analysis?library=music", nil))
	var before struct {
		Missing int `json:"missing"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &before) != nil || before.Missing != 1 {
		t.Fatalf("before = %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/audio-analysis?library=music", nil))
	if response.Code != http.StatusAccepted {
		t.Fatalf("post = %d %s", response.Code, response.Body.String())
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(store.MissingAnalysis()) == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(store.MissingAnalysis()) != 0 {
		t.Fatal("analysis did not finish")
	}
	item := store.Items()[0]
	if item.ReplayTrack != 2 || item.PeakTrack != .8 || item.AnalysisRevision != item.Revision {
		t.Fatalf("item = %#v", item)
	}
}
