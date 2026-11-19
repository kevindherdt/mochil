package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/audio"
)

func TestAudiobookSeriesSummariesAndPagedGroup(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"Author/Series 10/Book 2/second.m4b",
		"Author/Series 10/Book 1/first.m4b",
		"Author/Series 2/Book 1/other.m4b",
		"standalone.m4b",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := audio.NewStore("audiobooks", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "audiobooks", Name: "Audiobooks", Kind: "audiobooks", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := handler.(interface{ Close() }); ok {
		defer closer.Close()
	}
	summaryResponse := httptest.NewRecorder()
	handler.ServeHTTP(summaryResponse, httptest.NewRequest(http.MethodGet, "/api/audiobook-series?library=audiobooks&sort=series", nil))
	var summaries []audiobookSeriesSummary
	if err := json.Unmarshal(summaryResponse.Body.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if summaryResponse.Code != http.StatusOK || len(summaries) != 3 || summaries[0].Name != "Series 2" || summaries[1].Name != "Series 10" || summaries[1].Count != 2 {
		t.Fatalf("unexpected summaries: status=%d %#v", summaryResponse.Code, summaries)
	}
	groupResponse := httptest.NewRecorder()
	handler.ServeHTTP(groupResponse, httptest.NewRequest(http.MethodGet, "/api/audio?library=audiobooks&group=Series+10&sort=series&limit=1", nil))
	var books []audioView
	if err := json.Unmarshal(groupResponse.Body.Bytes(), &books); err != nil {
		t.Fatal(err)
	}
	if groupResponse.Code != http.StatusOK || groupResponse.Header().Get("X-Total-Count") != "2" || groupResponse.Header().Get("X-Next-Cursor") == "" || len(books) != 1 || books[0].SeriesIndex != 1 {
		t.Fatalf("unexpected group page: status=%d headers=%#v books=%#v", groupResponse.Code, groupResponse.Header(), books)
	}
}
