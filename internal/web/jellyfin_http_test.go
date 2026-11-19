package web

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/accounts"
	"mochil/internal/jellyfin"
	"mochil/internal/video"
)

func jellyfinLogin(t *testing.T, handler http.Handler, username, password string) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{"Username": username, "Pw": password})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/Users/AuthenticateByName", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Jellyfin test client")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("Jellyfin login=%d %s", response.Code, response.Body.String())
	}
	var auth struct{ AccessToken string }
	if json.Unmarshal(response.Body.Bytes(), &auth) != nil || auth.AccessToken == "" {
		t.Fatal("missing Jellyfin token")
	}
	return auth.AccessToken
}

func TestJellyfinLoginBrowseAndRangeStream(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Film.mp4"), []byte("0123456789"), 0o640); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{PublicURL: "https://media.example", Username: "reader", Password: "secret", VideoState: filepath.Join(t.TempDir(), "video-progress.json")}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	token := jellyfinLogin(t, handler, "reader", "secret")
	items := httptest.NewRequest(http.MethodGet, "/Items?ParentId="+jellyfin.ID("library", "movies"), nil)
	items.Header.Set("X-Emby-Token", token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, items)
	if response.Code != http.StatusOK {
		t.Fatalf("items=%d %s", response.Code, response.Body.String())
	}
	item := store.Items()[0]
	stream := httptest.NewRequest(http.MethodGet, "/Videos/"+jellyfin.ID("video", item.ID)+"/stream", nil)
	stream.Header.Set("X-Emby-Token", token)
	stream.Header.Set("Range", "bytes=2-5")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, stream)
	if response.Code != http.StatusPartialContent || response.Body.String() != "2345" {
		t.Fatalf("stream=%d %q", response.Code, response.Body.String())
	}
}

func TestJellyfinRejectsMochilBasicPasswordAsToken(t *testing.T) {
	handler, err := NewMulti(MultiConfig{PublicURL: "https://media.example", Username: "reader", Password: "secret"}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: video.NewStore("movies", t.TempDir(), "")}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/Items", nil)
	request.Header.Set("X-Emby-Token", "secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestJellyfinTokensBindProgressToStableAccounts(t *testing.T) {
	directory := t.TempDir()
	accountsPath := filepath.Join(directory, "accounts.json")
	accountStore, err := accounts.Open(accountsPath, "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := accountStore.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(directory, "videos")
	if err = os.MkdirAll(root, 0o750); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "Film.mp4"), []byte("video"), 0o640); err != nil {
		t.Fatal(err)
	}
	videoStore := video.NewStore("movies", root, "")
	if _, err = videoStore.Scan(); err != nil {
		t.Fatal(err)
	}
	progressPath := filepath.Join(directory, "video-progress.json")
	handler, err := NewMulti(MultiConfig{PublicURL: "https://media.example", Username: "owner", Password: "owner password", AccountsState: accountsPath, ReadingState: filepath.Join(directory, "reading.json"), VideoState: progressPath}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: videoStore}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ownerToken := jellyfinLogin(t, handler, "owner", "owner password")
	readerToken := jellyfinLogin(t, handler, "reader", "reader password")
	if ownerToken == readerToken {
		t.Fatal("different accounts received the same Jellyfin token")
	}
	itemID := jellyfin.ID("video", videoStore.Items()[0].ID)
	for _, token := range []string{ownerToken, readerToken} {
		request := httptest.NewRequest(http.MethodPost, "/Sessions/Playing/Progress", bytes.NewBufferString(`{"ItemId":"`+itemID+`","PositionTicks":0}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Emby-Token", token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("progress=%d %s", response.Code, response.Body.String())
		}
	}
	var saved struct {
		Accounts map[string]map[string]video.Progress `json:"accounts"`
	}
	data, err := os.ReadFile(progressPath)
	if err != nil || json.Unmarshal(data, &saved) != nil {
		t.Fatalf("read progress state: %v", err)
	}
	if len(saved.Accounts["owner"]) != 1 || len(saved.Accounts[reader.ID]) != 1 || len(saved.Accounts) != 2 {
		t.Fatalf("progress accounts = %#v", saved.Accounts)
	}
}

func TestJellyfinTokenStopsWhenAccountIsDisabled(t *testing.T) {
	directory := t.TempDir()
	accountStore, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := accountStore.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	devices, err := newAndroidDeviceStore(filepath.Join(directory, "jellyfin-devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := devices.issue(reader.ID, "Living room")
	if err != nil {
		t.Fatal(err)
	}
	server := &multiServer{accounts: accountStore, jellyfinDevices: devices}
	request := httptest.NewRequest(http.MethodGet, "/Items", nil)
	request.Header.Set("X-Emby-Token", token)
	if identity, ok := server.jellyfinIdentity(request); !ok || identity.ID != reader.ID {
		t.Fatalf("identity=%+v valid=%v", identity, ok)
	}
	if err = accountStore.SetDisabled(reader.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := server.jellyfinIdentity(request); ok {
		t.Fatal("disabled account retained Jellyfin access")
	}
}

func TestJellyfinTVLibraryExposesSeriesAndSeasons(t *testing.T) {
	root := t.TempDir()
	episode := filepath.Join(root, "Example Show", "Season 01", "Example.Show.S01E02.mp4")
	if err := os.MkdirAll(filepath.Dir(episode), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(episode, []byte("video"), 0o640); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("tv", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{PublicURL: "https://media.example", Username: "reader", Password: "secret"}, []Library{{ID: "tv", Name: "TV", Kind: "tv", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	token := jellyfinLogin(t, handler, "reader", "secret")
	request := httptest.NewRequest(http.MethodGet, "/Items?ParentId="+jellyfin.ID("library", "tv"), nil)
	request.Header.Set("X-Emby-Token", token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("series=%d %s", response.Code, response.Body.String())
	}
	var result struct{ Items []map[string]any }
	if json.Unmarshal(response.Body.Bytes(), &result) != nil || len(result.Items) != 1 || result.Items[0]["Type"] != "Series" {
		t.Fatalf("items=%s", response.Body.String())
	}
	seriesID, _ := result.Items[0]["Id"].(string)
	request = httptest.NewRequest(http.MethodGet, "/Shows/"+seriesID+"/Seasons", nil)
	request.Header.Set("X-Emby-Token", token)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"Type":"Season"`)) {
		t.Fatalf("seasons=%d %s", response.Code, response.Body.String())
	}
}
