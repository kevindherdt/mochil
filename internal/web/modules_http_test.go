package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModuleRoutesMapToTheirModule(t *testing.T) {
	for path, want := range map[string]string{
		"/videos/a/b": "video", "/api/video-hls/x": "video", "/api/videos": "video", "/api/metadata/video-quality": "video",
		"/audio/a/b": "audio", "/api/audio-queue": "audio", "/api/audiobook-series": "audio", "/podcast-media/x": "audio", "/api/podcast-feeds": "audio",
		"/documents/a": "documents", "/api/wiki": "documents", "/api/vaults/x": "documents",
		"/api/android/v1/videos": "video", "/api/android/v1/audio-groups": "audio", "/api/android/v1/podcast-feeds": "audio", "/api/android/v1/documents": "documents",
		"/api/android/v1/books": "", "/api/android/v1/home": "", "/api/android/v1/reading": "", "/api/android/v1/libraries": "",
		"/api/books": "", "/api/series": "", "/read/a/b": "", "/covers/a/b": "", "/": "", "/api/reading": "",
	} {
		if got := moduleForPath(path); got != want {
			t.Errorf("moduleForPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestSwitchedOffModulesAreNotThere(t *testing.T) {
	m := &multiServer{cfg: MultiConfig{Modules: map[string]bool{"video": false, "audio": false, "documents": true}}}
	if m.moduleOn("video") || !m.moduleOn("documents") {
		t.Fatal("module state not read")
	}
	if !(&multiServer{}).moduleOn("video") {
		t.Fatal("a nil module map means everything is on")
	}
	gone := httptest.NewRecorder()
	m.moduleOffResponse(gone, httptest.NewRequest(http.MethodGet, "/api/video-hls/a", nil))
	if gone.Code != http.StatusNotFound {
		t.Fatalf("got %d", gone.Code)
	}
	for _, path := range []string{"/api/video-progress", "/api/audio-progress", "/api/android/v1/video-progress", "/api/android/v1/audio-progress"} {
		list := httptest.NewRecorder()
		m.moduleOffResponse(list, httptest.NewRequest(http.MethodGet, path, nil))
		if list.Code != http.StatusOK || strings.TrimSpace(list.Body.String()) != "[]" {
			t.Fatalf("%s: %d %q", path, list.Code, list.Body.String())
		}
	}
	view := httptest.NewRecorder()
	m.apiModules(view, httptest.NewRequest(http.MethodGet, "/api/modules", nil))
	if view.Code != http.StatusOK || !strings.Contains(view.Body.String(), `"id":"video","name":"Video"`) || !strings.Contains(view.Body.String(), `"enabled":true`) {
		t.Fatalf("modules view: %s", view.Body.String())
	}
}

func TestAndroidModuleOwnsItsRoutesAndDefaultsOn(t *testing.T) {
	for path, want := range map[string]bool{
		"/api/android/v1/login": true, "/api/android/v1/books": true, "/pair": true, "/api/android/app.apk": true,
		"/api/books": false, "/read/a/b": false, "/api/koreader/x": false,
	} {
		if androidRoute(path) != want {
			t.Errorf("androidRoute(%q) = %v, want %v", path, !want, want)
		}
	}
	m := &multiServer{cfg: MultiConfig{Modules: map[string]bool{"android": false}}}
	for _, path := range []string{"/api/android/v1/login", "/pair", "/api/android/app.apk"} {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s with the module off: %d", path, w.Code)
		}
	}
}

func TestModulesViewExposesHardcoverGroups(t *testing.T) {
	m := &multiServer{cfg: MultiConfig{Modules: map[string]bool{}}}
	view := httptest.NewRecorder()
	m.apiModules(view, httptest.NewRequest(http.MethodGet, "/api/modules", nil))
	if view.Code != http.StatusOK {
		t.Fatalf("modules view = %d", view.Code)
	}
	var modules []map[string]any
	if err := json.Unmarshal(view.Body.Bytes(), &modules); err != nil {
		t.Fatalf("modules view is not JSON: %v", err)
	}
	byID := map[string]map[string]any{}
	for _, module := range modules {
		id, _ := module["id"].(string)
		byID[id] = module
	}
	lead, ok := byID["hardcover"]
	if !ok {
		t.Fatal("hardcover module missing from the view")
	}
	if _, grouped := lead["group"]; grouped {
		t.Fatalf("the hardcover card must not name a group of its own: %v", lead)
	}
	if option, _ := lead["option"].(bool); option {
		t.Fatalf("the hardcover card is not an option on another card: %v", lead)
	}
	for _, id := range []string{"hardcover_progress", "hardcover_ratings"} {
		module, ok := byID[id]
		if !ok {
			t.Fatalf("%s missing from the view", id)
		}
		if module["group"] != "hardcover" {
			t.Fatalf("%s group = %v, want hardcover", id, module["group"])
		}
		if option, _ := module["option"].(bool); !option {
			t.Fatalf("%s option = %v, want true", id, module["option"])
		}
		if _, enabled := module["enabled"]; !enabled {
			t.Fatalf("%s must still report its own switch: %v", id, module)
		}
	}
	for _, module := range modules {
		if module["group"] == nil {
			continue
		}
		if _, ok := byID[module["group"].(string)]; !ok {
			t.Fatalf("module %v names a group that is not in the view", module["id"])
		}
	}
	if _, grouped := byID["video"]["group"]; grouped {
		t.Fatalf("ungrouped modules must not carry an empty group: %v", byID["video"])
	}
}
