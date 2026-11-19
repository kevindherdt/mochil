package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mochil/internal/appconfig"
	"mochil/internal/audio"
)

func TestPlayFromLibraryCanBeSwitchedOff(t *testing.T) {
	store := audio.NewStore("music", t.TempDir(), "")
	called := false
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Username: "Kev", Hermes: appconfig.Hermes{URL: hermes.URL + "/", Key: "k", Features: map[string]bool{"library_play": false}}}, libraries: []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}}
	if cards, reply := m.hermesPlayLookup("kev", "play lilac", m.hermesSettings()); len(cards) != 0 || reply != "" {
		t.Fatalf("the ability is off but a lookup happened: %v %q", cards, reply)
	}
	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"play lilac"}`)))
	if !called {
		t.Fatal("with the ability off, the message must go to Hermes")
	}
}

func TestHermesFeaturesDefaultOnAndHonourTheLegacySwitch(t *testing.T) {
	var h appconfig.Hermes
	for _, name := range appconfig.HermesFeatures {
		if !h.Feature(name) {
			t.Fatalf("%s must default to on", name)
		}
	}
	if (appconfig.Hermes{NoLibraryPlay: true}).Feature("library_play") {
		t.Fatal("the older switch must still turn library play off")
	}
	if (appconfig.Hermes{Features: map[string]bool{"home": false}}).Feature("home") || !(appconfig.Hermes{Features: map[string]bool{"home": false}}).Feature("radio") {
		t.Fatal("features switch one at a time")
	}
}
