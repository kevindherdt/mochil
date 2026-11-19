package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/audio"
)

func TestAudioCursorPagesAndExpiresWithCatalogueSnapshot(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Alpha.mp3", "Bravo.mp3", "Charlie.mp3"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, AudioState: filepath.Join(t.TempDir(), "progress.json")}, []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := handler.(interface{ Close() }); ok {
		defer closer.Close()
	}
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/audio?library=music&sort=title&limit=1", nil))
	cursor := first.Header().Get("X-Next-Cursor")
	if first.Code != http.StatusOK || cursor == "" || !strings.Contains(first.Body.String(), `"title":"Alpha"`) {
		t.Fatalf("first page = %d %#v %s", first.Code, first.Header(), first.Body.String())
	}
	decodedCursor, err := decodeAudioCursor(cursor)
	if err != nil || decodedCursor.ProgressVersion != 0 {
		t.Fatalf("unfiltered shelf cursor unexpectedly depends on progress: %#v, %v", decodedCursor, err)
	}
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/audio?library=music&sort=title&limit=1&cursor="+url.QueryEscape(cursor), nil))
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"title":"Bravo"`) {
		t.Fatalf("second page = %d %s", second.Code, second.Body.String())
	}
	wrongView := httptest.NewRecorder()
	handler.ServeHTTP(wrongView, httptest.NewRequest(http.MethodGet, "/api/audio?library=music&sort=oldest&cursor="+url.QueryEscape(cursor), nil))
	if wrongView.Code != http.StatusBadRequest {
		t.Fatalf("cross-view cursor = %d", wrongView.Code)
	}
	if err := os.WriteFile(filepath.Join(root, "Delta.mp3"), []byte("delta"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	expired := httptest.NewRecorder()
	handler.ServeHTTP(expired, httptest.NewRequest(http.MethodGet, "/api/audio?library=music&sort=title&cursor="+url.QueryEscape(cursor), nil))
	if expired.Code != http.StatusConflict || !strings.Contains(expired.Body.String(), "reload") {
		t.Fatalf("expired cursor = %d %s", expired.Code, expired.Body.String())
	}
}

func TestAudioCursorCodecRejectsMalformedValues(t *testing.T) {
	cursor := audioCursor{AudioVersion: 4, PodcastVersion: 2, ProgressVersion: 7, View: "view", ItemID: "item"}
	encoded := encodeAudioCursor(cursor)
	decoded, err := decodeAudioCursor(encoded)
	if err != nil || decoded != cursor {
		t.Fatalf("round trip = %#v, %v", decoded, err)
	}
	for _, value := range []string{"%%%", strings.Repeat("a", 513), encodeAudioCursor(audioCursor{View: "view", ItemID: "item"})} {
		if _, err := decodeAudioCursor(value); err == nil {
			t.Fatalf("accepted malformed cursor %q", value)
		}
	}
}

func BenchmarkAudioCursorCodec(b *testing.B) {
	cursor := audioCursor{AudioVersion: 42, PodcastVersion: 9, ProgressVersion: 81, View: "0123456789abcdef01234567", ItemID: "0123456789abcdef0123456789abcdef"}
	for range b.N {
		encoded := encodeAudioCursor(cursor)
		if _, err := decodeAudioCursor(encoded); err != nil {
			b.Fatal(err)
		}
	}
}
