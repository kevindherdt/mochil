package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/audio"
)

type deliveryTestAudioProber struct{}

func (deliveryTestAudioProber) Probe(string) (audio.Metadata, error) {
	return audio.Metadata{Title: "Test track"}, nil
}

func TestDirectAudioDeliverySupportsRangesAndDownloads(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "track.mp3")
	if err := os.WriteFile(source, []byte("0123456789"), 0o640); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("music", root, filepath.Join(t.TempDir(), "audio.json"))
	store.SetProber(deliveryTestAudioProber{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	items := store.Items()
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	path := "/audio/music/" + items[0].ID + "/track.mp3"

	ranged := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Range", "bytes=2-5")
	handler.ServeHTTP(ranged, request)
	if ranged.Code != http.StatusPartialContent || ranged.Body.String() != "2345" {
		t.Fatalf("range = %d %q", ranged.Code, ranged.Body.String())
	}
	if ranged.Header().Get("Accept-Ranges") != "bytes" || !strings.HasPrefix(ranged.Header().Get("Content-Range"), "bytes 2-5/") {
		t.Fatalf("range headers = %#v", ranged.Header())
	}

	download := httptest.NewRecorder()
	handler.ServeHTTP(download, httptest.NewRequest(http.MethodHead, path+"?download=1", nil))
	if download.Code != http.StatusOK || !strings.Contains(download.Header().Get("Content-Disposition"), "attachment") || download.Body.Len() != 0 {
		t.Fatalf("download = %d disposition=%q bytes=%d", download.Code, download.Header().Get("Content-Disposition"), download.Body.Len())
	}

	rejected := httptest.NewRecorder()
	handler.ServeHTTP(rejected, httptest.NewRequest(http.MethodPost, path, nil))
	if rejected.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d, want %d", rejected.Code, http.StatusMethodNotAllowed)
	}
}

func TestAudioArtworkRejectsUnsupportedMethodsBeforeWork(t *testing.T) {
	response := httptest.NewRecorder()
	(&multiServer{}).audioArtworkHTTP(response, httptest.NewRequest(http.MethodPost, "/api/audio-artwork/music/item", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestMissingArtworkAnswerIsRememberedByTheBrowser(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "track.mp3"), []byte("0123456789"), 0o640); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("music", root, filepath.Join(t.TempDir(), "audio.json"))
	store.SetProber(deliveryTestAudioProber{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	artwork, err := audio.NewArtworkStore(t.TempDir(), "false", 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, AudioArtwork: artwork}, []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	id := store.Items()[0].ID

	withRevision := httptest.NewRecorder()
	handler.ServeHTTP(withRevision, httptest.NewRequest(http.MethodGet, "/api/audio-artwork/music/"+id+"?size=card&v=abc", nil))
	if withRevision.Code != http.StatusNotFound || !strings.Contains(withRevision.Header().Get("Cache-Control"), "max-age=3600") {
		t.Fatalf("with revision: %d cache=%q", withRevision.Code, withRevision.Header().Get("Cache-Control"))
	}
	without := httptest.NewRecorder()
	handler.ServeHTTP(without, httptest.NewRequest(http.MethodGet, "/api/audio-artwork/music/"+id, nil))
	if without.Code != http.StatusNotFound || strings.Contains(without.Header().Get("Cache-Control"), "max-age") {
		t.Fatalf("without revision: %d cache=%q", without.Code, without.Header().Get("Cache-Control"))
	}
}
