package web

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/audio"
)

type albumDownloadProber map[string]audio.Metadata

func (p albumDownloadProber) Probe(path string) (audio.Metadata, error) {
	return p[filepath.Base(path)], nil
}

func TestAudioGroupDownloadUsesOpaqueAlbumIdentity(t *testing.T) {
	root := t.TempDir()
	for relative, body := range map[string]string{"Artist A/a1.flac": "artist a track one", "Artist A/a2.flac": "artist a track two", "Artist B/b1.flac": "artist b track"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, relative)), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, relative), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := audio.NewStore("music", root, "")
	store.SetProber(albumDownloadProber{
		"a1.flac": {Title: "First", Artist: "Artist A", Album: "Shared", Track: 1},
		"a2.flac": {Title: "Second", Artist: "Artist A", Album: "Shared", Track: 2},
		"b1.flac": {Title: "Other", Artist: "Artist B", Album: "Shared", Track: 1},
	})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	m := &multiServer{libraries: []Library{{ID: "music", Kind: "music", Audio: store}}}
	var groupID string
	for _, item := range store.Items() {
		if item.Artist == "Artist A" {
			groupID = audioGroupItemID("music", "albums", item)
			break
		}
	}
	response := httptest.NewRecorder()
	m.audioGroupDownload(response, httptest.NewRequest(http.MethodGet, "/api/audio-group-download?library=music&group_id="+groupID, nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("download=%d type=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 2 || !strings.Contains(archive.File[0].Name, "a1.flac") || !strings.Contains(archive.File[1].Name, "a2.flac") {
		t.Fatalf("archive entries=%v", archive.File)
	}
	missing := httptest.NewRecorder()
	m.audioGroupDownload(missing, httptest.NewRequest(http.MethodGet, "/api/audio-group-download?library=music&group_id="+strings.Repeat("0", 32), nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown group=%d", missing.Code)
	}
}
