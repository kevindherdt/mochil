package web

import (
	"io"
	"log/slog"
	"mochil/internal/changelog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAndroidAppDownloadServesConfiguredAPK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mochil.apk")
	if err := os.WriteFile(path, []byte("signed-apk"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, AndroidAPK: path}, []Library{{ID: "books", Name: "Books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/api/android/app.apk", nil))
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/vnd.android.package-archive" || response.Header().Get("X-Mochil-App-Version") != androidAppVersion {
			t.Fatalf("%s response=%d headers=%v", method, response.Code, response.Header())
		}
		if method == http.MethodGet && response.Body.String() != "signed-apk" {
			t.Fatalf("download body=%q", response.Body.String())
		}
	}
}

func TestAndroidAppDownloadIsUnavailableWithoutConfiguredFile(t *testing.T) {
	server := &multiServer{}
	response := httptest.NewRecorder()
	server.androidAppDownload(response, httptest.NewRequest(http.MethodGet, "/api/android/app.apk", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d", response.Code)
	}
}

// A cached "you are up to date" is indistinguishable from a broken update check, so the answer must
// never be stored by anything between the server and the phone.
func TestAndroidAppInfoIsNeverCached(t *testing.T) {
	server := &multiServer{cfg: MultiConfig{AndroidAPK: "/configured/app.apk"}}
	response := httptest.NewRecorder()
	server.androidAppInfo(response, httptest.NewRequest(http.MethodGet, "/api/android/v1/app", nil))
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestAndroidAppInfo(t *testing.T) {
	server := &multiServer{cfg: MultiConfig{AndroidAPK: "/configured/app.apk"}}
	response := httptest.NewRecorder()
	server.androidAppInfo(response, httptest.NewRequest(http.MethodGet, "/api/android/v1/app", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"version_code":`+strconv.Itoa(androidAppVersionCode)) || !strings.Contains(response.Body.String(), `"download_url":"/api/android/v1/app.apk"`) || !strings.Contains(response.Body.String(), `"release_notes"`) {
		t.Fatalf("response=%d body=%s", response.Code, response.Body.String())
	}
	entry, ok := changelog.For("android", androidAppVersion)
	if !ok {
		t.Fatalf("the changelog has no entry for %s", androidAppVersion)
	}
	for _, note := range append(append(append([]string{}, entry.New...), entry.Fixed...), entry.Changes...) {
		if !strings.Contains(response.Body.String(), strings.ReplaceAll(strings.ReplaceAll(note, `"`, `\"`), "<", `\u003c`)) {
			t.Fatalf("release notes are missing %q: %s", note, response.Body.String())
		}
	}
}
