package web

import (
	"io"
	"log/slog"
	"mochil/internal/changelog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
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

// The download screen and the in-app update check both answer from the constants in
// android_distribution.go, while the file people actually install is built from
// android/app/build.gradle.kts. Those drifted once: the server kept advertising 0.1.11
// after 0.1.13 had shipped, so a phone already on 0.1.13 was offered nothing while a
// fresh install from the same server got a newer file than the one advertised.
// Bumping the Gradle version is now a gate failure until this matches.
func TestAndroidAppInfoMatchesTheAppThatIsActuallyBuilt(t *testing.T) {
	path := filepath.Join("..", "..", "android", "app", "build.gradle.kts")
	gradle, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("android build file not present (%v); the public export may exclude it", err)
	}
	code := regexp.MustCompile(`versionCode\s*=\s*(\d+)`).FindSubmatch(gradle)
	name := regexp.MustCompile(`versionName\s*=\s*"([^"]+)"`).FindSubmatch(gradle)
	if code == nil || name == nil {
		t.Fatalf("could not read versionCode/versionName from %s", path)
	}
	wantCode, err := strconv.Atoi(string(code[1]))
	if err != nil {
		t.Fatalf("versionCode in %s is not a number: %q", path, code[1])
	}
	if wantCode != androidAppVersionCode {
		t.Errorf("versionCode: build.gradle.kts says %d, android_distribution.go advertises %d", wantCode, androidAppVersionCode)
	}
	if wantName := string(name[1]); wantName != androidAppVersion {
		t.Errorf("versionName: build.gradle.kts says %q, android_distribution.go advertises %q", wantName, androidAppVersion)
	}
}
