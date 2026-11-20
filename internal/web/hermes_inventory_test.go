package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mochil/internal/appconfig"
)

func TestHermesInventoryAddressFollowsTheHermesHost(t *testing.T) {
	if got, ok := hermesInventoryAddress("", "http://192.0.2.10:8642/p/companion"); !ok || got != "http://192.0.2.10:8643/inventory" {
		t.Fatalf("derived: %q %v", got, ok)
	}
	if got, _ := hermesInventoryAddress("http://helper:9000/inventory", "http://x"); got != "http://helper:9000/inventory" {
		t.Fatalf("configured: %q", got)
	}
	if _, ok := hermesInventoryAddress("file:///etc/passwd", "http://x"); ok {
		t.Fatal("only http and https")
	}
}

func TestHermesInventoryRelaysTheHelperAndSurvivesItBeingDown(t *testing.T) {
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"plugins":[{"name":"homeassistant"}],"skills":[],"mcp":[]}`))
	}))
	m := &multiServer{cfg: MultiConfig{Hermes: appconfig.Hermes{URL: "http://127.0.0.1:1/", InventoryURL: helper.URL + "/inventory"}}}
	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodGet, "/api/hermes/inventory", nil))
	if !strings.Contains(w.Body.String(), `"available":true`) || !strings.Contains(w.Body.String(), "homeassistant") {
		t.Fatalf("relay: %s", w.Body.String())
	}
	helper.Close()
	w = httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodGet, "/api/hermes/inventory", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"available":false`) {
		t.Fatalf("down: %d %s", w.Code, w.Body.String())
	}
}

func TestHermesInstallSendsTheKeyAndNeverFromABadRequest(t *testing.T) {
	var gotAuth, gotBody string
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 512)
		n, _ := r.Body.Read(b)
		gotAuth, gotBody = r.Header.Get("Authorization"), string(b[:n])
		_, _ = w.Write([]byte(`{"ok":true,"message":"Installed"}`))
	}))
	defer helper.Close()
	m := &multiServer{cfg: MultiConfig{Hermes: appconfig.Hermes{URL: "http://127.0.0.1:1/", Key: "secret", InventoryURL: helper.URL + "/inventory"}}}
	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/install", strings.NewReader(`{"kind":"plugin","name":"langfuse"}`)))
	if !strings.Contains(w.Body.String(), `"ok":true`) || gotAuth != "Bearer secret" || !strings.Contains(gotBody, "langfuse") {
		t.Fatalf("install: %s auth=%q body=%q", w.Body.String(), gotAuth, gotBody)
	}
	for _, bad := range []string{`{"kind":"plugin"}`, `{"kind":"git","name":"x"}`, `{}`} {
		w = httptest.NewRecorder()
		m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/install", strings.NewReader(bad)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", bad, w.Code)
		}
	}
	w = httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodGet, "/api/hermes/catalog?kind=other", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("catalog kind: %d", w.Code)
	}
}
