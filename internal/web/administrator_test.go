package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"mochil/internal/accounts"
)

func TestAdministratorOnlyPolicySeparatesGlobalAndPersonalMutations(t *testing.T) {
	global := [][2]string{
		{"/api/settings", http.MethodGet},
		{"/api/restart", http.MethodPost},
		{"/api/metadata/issues", http.MethodGet},
		{"/api/metadata/jobs/one", http.MethodDelete},
		{"/api/publication-metadata/books/one", http.MethodPut},
		{"/api/video-metadata/movies/one", http.MethodPut},
		{"/api/video-show-metadata/tv/show", http.MethodDelete},
		{"/api/podcast-feeds", http.MethodPost},
		{"/api/podcast-downloads/one", http.MethodDelete},
		{"/api/audio-analysis", http.MethodPost},
		{"/api/video-preferences", http.MethodPut},
		{"/api/source-connection", http.MethodPost},
	}
	for _, request := range global {
		if !administratorOnly(request[0], request[1]) {
			t.Errorf("global operation was not gated: %s %s", request[1], request[0])
		}
	}
	personal := [][2]string{
		{"/api/reading/books/one", http.MethodPut},
		{"/api/audio-progress", http.MethodPut},
		{"/api/video-progress", http.MethodPut},
		{"/api/wiki", http.MethodPost},
		{"/api/collections", http.MethodPost},
		{"/api/annotations/books/one", http.MethodPost},
		{"/api/external-library", http.MethodPost},
		{"/api/podcast-feeds", http.MethodGet},
	}
	for _, request := range personal {
		if administratorOnly(request[0], request[1]) {
			t.Errorf("personal/read operation was gated: %s %s", request[1], request[0])
		}
	}
}

func TestOrdinaryAccountCannotReachGlobalControls(t *testing.T) {
	directory := t.TempDir()
	store, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create("reader", "reader password", accounts.RoleUser); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{Username: "owner", Password: "owner password", ReadingState: filepath.Join(directory, "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	loginRequest := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"reader password"}})
	login := httptest.NewRecorder()
	handler.ServeHTTP(login, loginRequest)
	cookie := login.Result().Cookies()[0]
	for _, target := range []struct{ method, path string }{{http.MethodGet, "/api/settings"}, {http.MethodGet, "/api/metadata/issues"}, {http.MethodPost, "/api/podcast-feeds"}, {http.MethodPost, "/api/restart"}} {
		request := httptest.NewRequest(target.method, target.path, nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Errorf("%s %s status=%d body=%s", target.method, target.path, response.Code, response.Body.String())
		}
	}
	personal := httptest.NewRequest(http.MethodGet, "/api/reading", nil)
	personal.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, personal)
	if response.Code != http.StatusOK {
		t.Fatalf("personal reading route status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAdministratorOnlyCoversDiagnosticsAndSharedVoiceChanges(t *testing.T) {
	for _, request := range [][2]string{
		{"/api/diagnostics", http.MethodGet},
		{"/api/narration/voices/one", http.MethodPatch},
		{"/api/narration/voices/one", http.MethodDelete},
	} {
		if !administratorOnly(request[0], request[1]) {
			t.Errorf("not gated: %s %s", request[1], request[0])
		}
	}
	for _, request := range [][2]string{
		{"/api/narration/voices/one/sample", http.MethodGet},
		{"/api/narration/voices", http.MethodGet},
	} {
		if administratorOnly(request[0], request[1]) {
			t.Errorf("gated although ordinary users need it: %s %s", request[1], request[0])
		}
	}
}

func TestSharedVaultAndVoiceCloningAreAdministratorWrites(t *testing.T) {
	for _, request := range [][2]string{
		{"/api/narration/voices", http.MethodPost},
		{"/api/vaults/notes/pages", http.MethodPost},
		{"/api/vaults/notes/pages/a.md", http.MethodPut},
	} {
		if !administratorOnly(request[0], request[1]) {
			t.Errorf("%s %s should need an administrator", request[1], request[0])
		}
	}
	for _, request := range [][2]string{
		{"/api/wiki", http.MethodPost}, // the wiki is each person's own notes
		{"/api/wiki/pages/x", http.MethodPut},
		{"/api/vaults/notes/pages/a.md", http.MethodGet},
		{"/api/narration/voices", http.MethodGet},
		{"/api/narration/voices/someone/sample", http.MethodGet},
	} {
		if administratorOnly(request[0], request[1]) {
			t.Errorf("%s %s is reading and must stay open", request[1], request[0])
		}
	}
}

func TestOnlyAdministratorsSeeTheVaultAsEditable(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/documents", nil)
	if !(&multiServer{}).mayEditVaults(r) {
		t.Fatal("without accounts the single owner may edit")
	}
	server := &multiServer{accounts: &accounts.Store{}}
	reader := r.WithContext(contextWithIdentity(r.Context(), accounts.Identity{ID: "r", Username: "r", Role: accounts.RoleUser}))
	owner := r.WithContext(contextWithIdentity(r.Context(), accounts.Identity{ID: "o", Username: "o", Role: accounts.RoleAdministrator}))
	if server.mayEditVaults(reader) || server.mayEditVaults(r) {
		t.Fatal("an ordinary or unknown account must not see edit controls")
	}
	if !server.mayEditVaults(owner) {
		t.Fatal("an administrator must")
	}
}

func TestNarrationStatusTellsWhoMayManageVoices(t *testing.T) {
	status := func(server *multiServer, identity *accounts.Identity) bool {
		r := httptest.NewRequest(http.MethodGet, "/api/narration/status", nil)
		if identity != nil {
			r = r.WithContext(contextWithIdentity(r.Context(), *identity))
		}
		w := httptest.NewRecorder()
		server.narrationAPI(w, r)
		var body struct {
			Manage bool `json:"manage"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Manage
	}
	server := &multiServer{accounts: &accounts.Store{}}
	if status(server, &accounts.Identity{ID: "r", Role: accounts.RoleUser}) {
		t.Fatal("an ordinary account must not be offered voice management")
	}
	if !status(server, &accounts.Identity{ID: "o", Role: accounts.RoleAdministrator}) {
		t.Fatal("an administrator must")
	}
}
