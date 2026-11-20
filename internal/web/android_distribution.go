package web

import (
	"fmt"
	"strconv"
	"strings"

	"mochil/internal/changelog"
	"mochil/internal/version"
	"net/http"
	"os"
	"path/filepath"
)

// What the download screen offers, and what "is there a newer version?" is answered from.
//
// This was a third place that knew the app's version, and it drifted the way a third copy does: the
// server advertised 0.1.20 while the APK was rebuilt every release, so an installed phone was told it
// was up to date and never took an update. The release is now the only source: the app is labelled
// with VERSION by Gradle, and the server answers with the version it was built as.
var (
	androidAppVersion     = androidVersionName()
	androidAppVersionCode = androidVersionCode(androidAppVersion)
)

// A binary built from a working tree has no release number of its own, so a dev server reads the VERSION
// file beside it rather than advertising "dev"; the constant covers the one case with neither -- a binary
// copied somewhere without the repository -- and is the last release the app shipped as.
const androidDevVersion = "0.1.62"

func androidVersionName() string {
	if stamped := strings.TrimSpace(version.Version); stamped != "" && stamped != "0.0.0-dev" {
		return stamped
	}
	for _, path := range []string{"VERSION", filepath.Join("..", "VERSION"), filepath.Join("..", "..", "VERSION")} {
		if raw, err := os.ReadFile(path); err == nil {
			if name := strings.TrimSpace(string(raw)); androidVersionCode(name) != 0 {
				return name
			}
		}
	}
	return androidDevVersion
}

// androidVersionCode is the rule android/app/build.gradle.kts applies, kept in one shape: MAJOR.MINOR.PATCH
// becomes (major*10000 + minor*100 + patch) * 100, which clears the 444 that every release up to 0.1.61
// carried, because an upgrade needs a code strictly above the installed one.
func androidVersionCode(name string) int {
	parts := strings.Split(name, ".")
	if len(parts) != 3 {
		return 0
	}
	numbers := make([]int, 3)
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 || number > 99 {
			return 0
		}
		numbers[index] = number
	}
	return (numbers[0]*10000 + numbers[1]*100 + numbers[2]) * 100
}

func (m *multiServer) androidAppInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.cfg.AndroidAPK == "" {
		http.Error(w, "The Android app download is not configured", http.StatusNotFound)
		return
	}
	// "Is there a newer version?" must never be answered from a cache: a stale answer tells the app it
	// is up to date long after a release, which is indistinguishable from a broken update check.
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"version_name":  androidAppVersion,
		"version_code":  androidAppVersionCode,
		"download_url":  "/api/android/v1/app.apk",
		"release_notes": androidReleaseNotes(),
	})
}

// androidReleaseNotes is what this release changed, read from the changelog so the update screen and the changelog agree.
func androidReleaseNotes() map[string][]string {
	entry, ok := changelog.For("android", androidAppVersion)
	if !ok {
		// Not every release changes the app: the notes that matter are the newest the app has, not the
		// ones dated for a server-only version number.
		if latest, found := changelog.Latest("android"); found {
			entry = latest
		} else {
			return map[string][]string{}
		}
	}
	return map[string][]string{"new": entry.New, "fixes": entry.Fixed, "performance": entry.Performance, "changes": entry.Changes}
}

// apiChangelog lists releases newest first: ?component=android|server|web and ?limit=N narrow it.
func (m *multiServer) apiChangelog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 200
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": changelog.List(r.URL.Query().Get("component"), limit)})
}

func (m *multiServer) androidAppDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.cfg.AndroidAPK == "" {
		http.Error(w, "The Android app download is not configured", http.StatusNotFound)
		return
	}
	file, err := os.Open(m.cfg.AndroidAPK)
	if err != nil {
		http.Error(w, "The Android app is temporarily unavailable", http.StatusNotFound)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.Error(w, "The Android app is temporarily unavailable", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	w.Header().Set("Content-Disposition", `attachment; filename="mochil-android.apk"`)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Mochil-App-Version", androidAppVersion)
	w.Header().Set("ETag", fmt.Sprintf(`W/"%x-%x"`, info.Size(), info.ModTime().Unix()))
	http.ServeContent(w, r, filepath.Base(m.cfg.AndroidAPK), info.ModTime(), file)
}
