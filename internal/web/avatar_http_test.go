package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/appconfig"
)

func avatarFixture(t *testing.T) (*multiServer, string) {
	t.Helper()
	root := t.TempDir()
	write := func(name, body string) {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Ice Girl/Pack/Ice.model3.json", `{"Version":3}`)
	write("Ice Girl/Pack/Ice.moc3", "MOC3")
	write("Ice Girl/Pack/脸红.exp3.json", `{}`)
	write("Ice Girl/Pack/Wave.motion3.json", `{}`)
	write("Ice Girl/Pack/notes.txt", "private")
	write("Other/Other.model3.json", `{}`)
	write("_runtime/live2dcubismcore.min.js", "//core")
	write("_runtime/secret.json", "{}")
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "leak.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "Other", "link")); err != nil {
		t.Fatal(err)
	}
	return &multiServer{cfg: MultiConfig{Avatar: appconfig.Avatar{Directory: root}, AvatarState: filepath.Join(t.TempDir(), "avatar.json")}}, root
}

func TestAvatarModelsAreFoundAndSelectionIsSaved(t *testing.T) {
	m, _ := avatarFixture(t)
	w := httptest.NewRecorder()
	m.apiAvatar(w, httptest.NewRequest(http.MethodGet, "/api/avatar/models", nil))
	var listed struct {
		Models   []avatarModel `json:"models"`
		Selected string        `json:"selected"`
		Core     bool          `json:"core"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listed); err != nil || len(listed.Models) != 2 || !listed.Core {
		t.Fatalf("%v %s", err, w.Body.String())
	}
	first := listed.Models[0]
	if first.ID != "Ice Girl/Pack" || first.Model != "Ice.model3.json" || len(first.Expressions) != 1 || first.Expressions[0].Name != "脸红" || len(first.Motions) != 1 {
		t.Fatalf("model: %+v", first)
	}
	if listed.Selected != "Ice Girl/Pack" {
		t.Fatalf("the first model is the default, got %q", listed.Selected)
	}
	put := func(body string) int {
		w := httptest.NewRecorder()
		m.apiAvatar(w, httptest.NewRequest(http.MethodPut, "/api/avatar/select", strings.NewReader(body)))
		return w.Code
	}
	if put(`{"model":"nope"}`) != http.StatusBadRequest || put(`{"model":"Other"}`) != http.StatusOK {
		t.Fatal("only known models can be selected")
	}
	w = httptest.NewRecorder()
	m.apiAvatar(w, httptest.NewRequest(http.MethodGet, "/api/avatar/models", nil))
	if !strings.Contains(w.Body.String(), `"selected":"Other"`) {
		t.Fatalf("selection was not kept: %s", w.Body.String())
	}
}

func TestAvatarFilesAreServedSafely(t *testing.T) {
	m, _ := avatarFixture(t)
	get := func(target string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		m.apiAvatar(w, httptest.NewRequest(http.MethodGet, target, nil))
		return w
	}
	if w := get("/api/avatar/files/Ice%20Girl/Pack/Ice.moc3"); w.Code != 200 || w.Body.String() != "MOC3" {
		t.Fatalf("model file: %d", w.Code)
	}
	if w := get("/api/avatar/core"); w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
		t.Fatalf("core: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	for _, bad := range []string{
		"/api/avatar/files/Ice%20Girl/Pack/notes.txt",     // not a model file type
		"/api/avatar/files/../../etc/passwd",              // outside
		"/api/avatar/files/_runtime/secret.json",          // runtime folder is only for the core
		"/api/avatar/files/Other/link/leak.json",          // a link that leaves the folder
		"/api/avatar/files/Ice%20Girl/Pack",               // a folder
		"/api/avatar/files/%2e%2e/%2e%2e/etc/passwd.json", // encoded traversal
	} {
		if w := get(bad); w.Code != http.StatusNotFound {
			t.Fatalf("%s should be refused, got %d", bad, w.Code)
		}
	}
}

func TestAvatarIsAdministratorOnlyAndAModule(t *testing.T) {
	for _, path := range []string{"/api/avatar/models", "/api/avatar/files/x/y.json", "/api/avatar/select", "/api/avatar/core"} {
		if !administratorOnly(path, http.MethodGet) {
			t.Fatalf("%s must be administrators only", path)
		}
	}
	if moduleForPath("/api/avatar/models") != "avatar" {
		t.Fatal("avatar routes belong to the avatar module")
	}
}

func TestAvatarKeepsAVoicePerModel(t *testing.T) {
	m, _ := avatarFixture(t)
	put := func(body string) int {
		w := httptest.NewRecorder()
		m.apiAvatar(w, httptest.NewRequest(http.MethodPut, "/api/avatar/select", strings.NewReader(body)))
		return w.Code
	}
	if put(`{"model":"Other","voice":"alba"}`) != http.StatusOK || put(`{"model":"Other","voice":"no-such-voice"}`) != http.StatusBadRequest {
		t.Fatal("a model's voice must be a known voice")
	}
	// Choosing another model keeps the voices already set.
	if put(`{"model":"Ice Girl/Pack"}`) != http.StatusOK {
		t.Fatal("select")
	}
	w := httptest.NewRecorder()
	m.apiAvatar(w, httptest.NewRequest(http.MethodGet, "/api/avatar/models", nil))
	if !strings.Contains(w.Body.String(), `"voices":{"Other":"alba"}`) || !strings.Contains(w.Body.String(), `"selected":"Ice Girl/Pack"`) {
		t.Fatalf("%s", w.Body.String())
	}
	if put(`{"model":"Other","voice":""}`) != http.StatusOK {
		t.Fatal("clear")
	}
	w = httptest.NewRecorder()
	m.apiAvatar(w, httptest.NewRequest(http.MethodGet, "/api/avatar/models", nil))
	if strings.Contains(w.Body.String(), `"Other":"alba"`) {
		t.Fatalf("the voice was not cleared: %s", w.Body.String())
	}
}
