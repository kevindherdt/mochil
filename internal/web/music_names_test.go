package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/audio"
)

func TestMusicNamesSuggestsFromTagsAndNeverTouchesAFile(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{"dl/a1.flac": "one", "dl/a2.flac": "two", "dl/02 - Second.flac": "blocker"}
	for relative, body := range files {
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
	})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	m := &multiServer{libraries: []Library{{ID: "music", Kind: "music", SourcePath: root, Audio: store}}}
	var groupID string
	for _, item := range store.Items() {
		if strings.HasPrefix(item.Path, "dl/") {
			groupID = audioGroupItemID("music", "albums", item)
		}
	}
	response := httptest.NewRecorder()
	m.hermesMusicNames(response, httptest.NewRequest(http.MethodGet, "/api/hermes/music-names?library=music&group_id="+groupID, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("%d %s", response.Code, response.Body)
	}
	var plan struct {
		Tracks  []struct{ From, To, Status, Problem string }
		Change  int
		Problem int
	}
	_ = json.Unmarshal(response.Body.Bytes(), &plan)
	if plan.Change != 1 || plan.Problem != 1 || plan.Tracks[0].To != "dl/01 - First.flac" || !strings.Contains(plan.Tracks[1].Problem, "already exists") {
		t.Fatalf("plan = %s", response.Body)
	}
	for relative, body := range files {
		if got, err := os.ReadFile(filepath.Join(root, relative)); err != nil || string(got) != body {
			t.Fatalf("%s was changed or removed", relative)
		}
	}
	missing := httptest.NewRecorder()
	m.hermesMusicNames(missing, httptest.NewRequest(http.MethodGet, "/api/hermes/music-names?library=music&group_id="+strings.Repeat("0", 32), nil))
	post := httptest.NewRecorder()
	m.hermesMusicNames(post, httptest.NewRequest(http.MethodPost, "/api/hermes/music-names?library=music&group_id="+groupID, nil))
	if missing.Code != http.StatusNotFound || post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("missing=%d post=%d", missing.Code, post.Code)
	}
}

func renameFixture(t *testing.T) (*multiServer, string, string, string) {
	t.Helper()
	root, state := t.TempDir(), t.TempDir()
	for relative, body := range map[string]string{"dl/a1.flac": "one", "dl/a2.flac": "two", "dl/cover.jpg": "art"} {
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
	})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	queue, _ := audio.OpenQueueStore(filepath.Join(state, "q.json"))
	favourites, _ := audio.OpenQueueStore(filepath.Join(state, "f.json"))
	lists, _ := audio.OpenPlaylistStore(filepath.Join(state, "p.json"))
	progress, _ := audio.OpenProgressStore(filepath.Join(state, "g.json"))
	m := &multiServer{libraries: []Library{{ID: "music", Kind: "music", SourcePath: root, Audio: store}}, audioQueue: queue, audioFavorites: favourites, audioPlaylists: lists, audioProgress: progress}
	m.cfg.AudioState = filepath.Join(state, "audio-state.json")
	var groupID, firstID string
	for _, item := range store.Items() {
		groupID = audioGroupItemID("music", "albums", item)
		if item.Track == 1 {
			firstID = item.ID
		}
	}
	return m, root, groupID, firstID
}

func postJSON(m *multiServer, target, body string) (int, map[string]any) {
	response := httptest.NewRecorder()
	m.hermesMusicNames(response, httptest.NewRequest(http.MethodPost, target, strings.NewReader(body)))
	var decoded map[string]any
	_ = json.Unmarshal(response.Body.Bytes(), &decoded)
	return response.Code, decoded
}

func TestApplyRenamesCarriesHistoryToTheNewIdsAndUndoBringsItAllBack(t *testing.T) {
	m, root, groupID, firstID := renameFixture(t)
	newID, _ := audio.Identify("music", "dl/01 - First.flac", 3, time.Now())
	_ = m.audioPlaylists.Put("kevin", "Mix", []audio.QueueEntry{{LibraryID: "music", ItemID: firstID}})
	_ = m.audioFavorites.Put("kevin", []audio.QueueEntry{{LibraryID: "music", ItemID: firstID}})
	_ = m.audioProgress.Put("kevin", audio.Progress{LibraryID: "music", ItemID: firstID, Position: 30, Duration: 100})

	body := `{"library":"music","group_id":"` + groupID + `","names":[{"id":"` + firstID + `","to":"dl/01 - First.flac"}]}`
	code, reply := postJSON(m, "/api/hermes/music-names/apply", body)
	if code != http.StatusOK || reply["renamed"] != float64(1) {
		t.Fatalf("apply = %d %v", code, reply)
	}
	for relative, want := range map[string]bool{"dl/01 - First.flac": true, "dl/a1.flac": false, "dl/a2.flac": true, "dl/cover.jpg": true} {
		if _, err := os.Stat(filepath.Join(root, relative)); (err == nil) != want {
			t.Fatalf("%s exists = %v", relative, err == nil)
		}
	}
	if list, _ := m.audioPlaylists.Get("kevin", "Mix"); list.Entries[0].ItemID != newID.ID {
		t.Fatalf("playlist = %+v", list)
	}
	if got := m.audioFavorites.Get("kevin"); got[0].ItemID != newID.ID {
		t.Fatalf("favourites = %+v", got)
	}
	if p, ok := m.audioProgress.Get("kevin", "music", newID.ID); !ok || p.Position != 30 {
		t.Fatalf("progress = %+v %v", p, ok)
	}

	last := httptest.NewRecorder()
	m.hermesMusicNames(last, httptest.NewRequest(http.MethodGet, "/api/hermes/music-names/last?library=music", nil))
	var info map[string]any
	_ = json.Unmarshal(last.Body.Bytes(), &info)
	if info["ledger"] == "" || info["files"] != float64(1) {
		t.Fatalf("last = %s", last.Body)
	}
	if code, reply = postJSON(m, "/api/hermes/music-names/undo", `{"library":"music","ledger":"`+info["ledger"].(string)+`"}`); code != http.StatusOK {
		t.Fatalf("undo = %d %v", code, reply)
	}
	if _, err := os.Stat(filepath.Join(root, "dl/a1.flac")); err != nil {
		t.Fatal("the file must be back")
	}
	if list, _ := m.audioPlaylists.Get("kevin", "Mix"); list.Entries[0].ItemID != firstID {
		t.Fatalf("playlist after undo = %+v", list)
	}
	if p, ok := m.audioProgress.Get("kevin", "music", firstID); !ok || p.Position != 30 {
		t.Fatalf("progress after undo = %+v %v", p, ok)
	}
}

func TestApplyRefusesNamesThatAreNotSafeAndTracksFromOtherAlbums(t *testing.T) {
	m, root, groupID, firstID := renameFixture(t)
	for _, to := range []string{"other/01.flac", "../x.flac", "dl/a2.flac", "dl/01 - First.mp3", "dl/a:b.flac"} {
		code, _ := postJSON(m, "/api/hermes/music-names/apply", `{"library":"music","group_id":"`+groupID+`","names":[{"id":"`+firstID+`","to":"`+to+`"}]}`)
		if code == http.StatusOK {
			t.Fatalf("%s must be refused", to)
		}
	}
	if code, _ := postJSON(m, "/api/hermes/music-names/apply", `{"library":"music","group_id":"`+groupID+`","names":[{"id":"not-in-album","to":"dl/x.flac"}]}`); code != http.StatusBadRequest {
		t.Fatalf("a foreign track = %d", code)
	}
	for _, name := range []string{"dl/a1.flac", "dl/a2.flac", "dl/cover.jpg"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("%s was changed", name)
		}
	}
}

func TestTidyWorksWithoutAModelAndAModelNeverInventsWords(t *testing.T) {
	m, _, groupID, _ := renameFixture(t)
	plan := func() (int, string) {
		response := httptest.NewRecorder()
		m.hermesMusicNames(response, httptest.NewRequest(http.MethodGet, "/api/hermes/music-names?library=music&tidy=1&group_id="+groupID, nil))
		return response.Code, response.Body.String()
	}
	if code, body := plan(); code != http.StatusOK || !strings.Contains(body, `"tidied":true`) {
		t.Fatalf("the fixed rules need no model: %d %s", code, body)
	}
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply := `{"tracks":[{"n":1,"title":"First (Remastered 2009)","artist":"Artist A","album":"Shared"},{"n":2,"title":"Second","artist":"Artist A","album":"Shared"}]}`
		payload, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}}}})
		_, _ = w.Write(payload)
	}))
	defer helper.Close()
	m.cfg.Hermes.HelperURL, m.cfg.Hermes.HelperModel = helper.URL, "m"
	code, body := plan()
	if code != http.StatusOK || strings.Contains(body, "Remastered") {
		t.Fatalf("an added word must be ignored: %d %s", code, body)
	}
	m.cfg.Hermes.HelperURL = "http://127.0.0.1:1"
	if code, body := plan(); code != http.StatusOK || !strings.Contains(body, "only the fixed cleaning rules") {
		t.Fatalf("a model that is down must not break tidying: %d %s", code, body)
	}
}
