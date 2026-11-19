package web

import (
	"fmt"
	"strconv"

	"mochil/internal/changelog"
	"net/http"
	"os"
	"path/filepath"
)

const (
	androidAppVersion     = "0.1.11"
	androidAppVersionCode = 435
)

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
		return map[string][]string{}
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
