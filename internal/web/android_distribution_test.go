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

// The download screen and the in-app update check both answer from what the server was built as, and the
// file people install is built by Gradle from the same VERSION. Those used to be three copies of the same
// number, and all three drifted: the app was labelled 0.1.20 for forty releases, the server advertised
// 444, and a phone was told it was up to date while a newer APK sat behind the download link.
//
// There is one source now. This pins both ends: Gradle must derive its labels from VERSION, and the number
// the server advertises must be the one the release carries.
func TestAndroidAppInfoMatchesTheAppThatIsActuallyBuilt(t *testing.T) {
	gradlePath := filepath.Join("..", "..", "android", "app", "build.gradle.kts")
	gradle, err := os.ReadFile(gradlePath)
	if err != nil {
		t.Skipf("android build file not present (%v); the public export may exclude it", err)
	}
	build := string(gradle)
	if !regexp.MustCompile(`versionName\s*=\s*mochilVersion`).MatchString(build) ||
		!regexp.MustCompile(`versionCode\s*=\s*mochilVersionCode`).MatchString(build) {
		t.Fatal("android/app/build.gradle.kts must take versionName and versionCode from the VERSION file, not literals")
	}
	if regexp.MustCompile(`versionCode\s*=\s*\d`).MatchString(build) || regexp.MustCompile(`versionName\s*=\s*"`).MatchString(build) {
		t.Fatal("android/app/build.gradle.kts still hard-codes a version; that is the drift this test exists for")
	}

	raw, err := os.ReadFile(filepath.Join("..", "..", "VERSION"))
	if err != nil {
		t.Fatalf("VERSION is missing: %v", err)
	}
	release := strings.TrimSpace(string(raw))
	if got := androidVersionCode(release); got != androidAppVersionCode {
		t.Errorf("the server advertises version_code %d for %s; the app builds %d", androidAppVersionCode, release, got)
	}
	if androidAppVersionCode <= 444 {
		t.Errorf("version_code %d would be refused as a downgrade over the releases that were all 444", androidAppVersionCode)
	}
	if androidAppVersion != release {
		t.Errorf("the server advertises %s; the release is %s", androidAppVersion, release)
	}
}
