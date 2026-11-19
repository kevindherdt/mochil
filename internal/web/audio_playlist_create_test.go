package web

import (
	"mochil/internal/audio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlaylistCreateCannotEraseExistingTracks(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	if err := m.audioPlaylists.Put("reader", "Evening", []audio.QueueEntry{{LibraryID: "music", ItemID: "one"}}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Evening", " evening ", "EVENING"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/audio-playlists", strings.NewReader(`{"name":"`+name+`"}`))
		m.audioPlaylistCollection(response, request, "reader")
		if response.Code != http.StatusConflict {
			t.Fatalf("duplicate create status = %d", response.Code)
		}
		list, found := m.audioPlaylists.Get("reader", "Evening")
		if !found || list.Name != "Evening" || len(list.Entries) != 1 {
			t.Fatalf("playlist changed: %#v", list)
		}
	}
}
