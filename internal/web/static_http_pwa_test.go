package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInstallableAppFilesAreServedFromTheRoot(t *testing.T) {
	m := &multiServer{}
	worker := httptest.NewRecorder()
	m.pwaFile(worker, httptest.NewRequest(http.MethodGet, "/sw.js", nil))
	if worker.Code != http.StatusOK || worker.Header().Get("Service-Worker-Allowed") != "/" || !strings.Contains(worker.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("service worker: %d %v", worker.Code, worker.Header())
	}
	if cache := worker.Header().Get("Cache-Control"); !strings.Contains(cache, "no-cache") {
		t.Fatalf("the worker must revalidate on every load, got %q", cache)
	}
	manifest := httptest.NewRecorder()
	m.pwaFile(manifest, httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil))
	if manifest.Code != http.StatusOK || manifest.Header().Get("Content-Type") != "application/manifest+json" || !strings.Contains(manifest.Body.String(), `"display": "standalone"`) {
		t.Fatalf("manifest: %d %v", manifest.Code, manifest.Header())
	}
	for _, icon := range []string{"icon-48.png", "icon-128.png", "icon-256.png", "icon-192.png", "icon-512.png", "icon-maskable-512.png", "apple-touch-icon.png", "favicon-32.png"} {
		response := httptest.NewRecorder()
		m.assets(response, httptest.NewRequest(http.MethodGet, "/assets/icons/"+icon, nil))
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("%s: %d %v", icon, response.Code, response.Header())
		}
	}
}

func TestReadiumPreviewPageGetsOnlyTheScriptAndBaseRelaxations(t *testing.T) {
	m := &multiServer{}
	classic := httptest.NewRecorder()
	m.reader(classic, httptest.NewRequest(http.MethodGet, "/reader/books/abc?engine=classic", nil))
	policy := classic.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "script-src 'self';") || !strings.Contains(policy, "base-uri 'none'") {
		t.Fatalf("the classic reader keeps the strict policy, got %q", policy)
	}
	preview := httptest.NewRecorder()
	m.reader(preview, httptest.NewRequest(http.MethodGet, "/reader/books/abc?engine=readium", nil))
	relaxed := preview.Header().Get("Content-Security-Policy")
	if preview.Code != http.StatusOK || !strings.Contains(relaxed, "script-src 'self' blob:;") || !strings.Contains(relaxed, "base-uri 'self'") {
		t.Fatalf("readium preview: %d %q", preview.Code, relaxed)
	}
	if strings.Contains(relaxed, "unsafe-eval") || !strings.Contains(relaxed, "connect-src 'self';") {
		t.Fatalf("nothing else may loosen: %q", relaxed)
	}
}
