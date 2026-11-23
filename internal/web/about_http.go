package web

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"mochil/internal/version"
)

// servedSince is when this process started serving, used for the uptime line. It is process-wide because a version
// answer is a fact about the binary, not about one library.
var servedSince = time.Now()

// aboutInfo is the one place the running build describes itself. Everything here is safe to hand out unauthenticated:
// a version number, a commit, a build and start time. It deliberately carries nothing about libraries, paths or
// accounts, so /about can answer "which build is this?" before anyone signs in, the way /healthz answers "is it up?".
type aboutInfo struct {
	Product   string `json:"product"`
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	ModTime   string `json:"binary_time,omitempty"`
	Started   string `json:"started"`
	Uptime    string `json:"uptime"`
	GoVersion string `json:"go,omitempty"`
	PID       int    `json:"pid"`
}

// buildAbout assembles the answer. The commit and build time come from the linker when a release supplies them and
// from the module's build info otherwise, so a locally built binary still names the commit it came from instead of
// claiming to be 0.0.0-dev with no history.
func buildAbout() aboutInfo {
	info := aboutInfo{
		Product: "Mochil",
		Version: version.Version,
		Commit:  version.Commit,
		Started: servedSince.UTC().Format(time.RFC3339),
		Uptime:  time.Since(servedSince).Round(time.Second).String(),
		PID:     os.Getpid(),
	}
	if build, ok := debug.ReadBuildInfo(); ok {
		info.GoVersion = build.GoVersion
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "" && len(setting.Value) >= 7 {
					info.Commit = setting.Value[:7]
				}
			case "vcs.time":
				if info.BuildTime == "" {
					if stamp, err := time.Parse(time.RFC3339, setting.Value); err == nil {
						info.BuildTime = stamp.UTC().Format(time.RFC3339)
					}
				}
			}
		}
	}
	if exe, err := os.Executable(); err == nil {
		if stat, err := os.Stat(exe); err == nil {
			info.ModTime = stat.ModTime().UTC().Format(time.RFC3339)
		}
	}
	return info
}

// versionAPI answers the About panel. It is never cached: a stale version is worse than no version, because it is
// exactly the answer someone is reading to work out whether their browser is showing them the new build.
func (m *multiServer) versionAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(buildAbout())
}

// aboutText is the same facts as plain text, for a terminal: `curl -s https://host/about` answers "which build am I
// looking at?" in one line without signing in, a session or a token.
func (m *multiServer) aboutText(w http.ResponseWriter, r *http.Request) {
	info := buildAbout()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	lines := "Mochil " + info.Version
	if info.Commit != "" {
		lines += " (" + info.Commit + ")"
	}
	lines += "\n"
	if info.BuildTime != "" {
		lines += "built  " + info.BuildTime + "\n"
	}
	lines += "started " + info.Started + "\n"
	lines += "uptime " + info.Uptime + "\n"
	_, _ = w.Write([]byte(lines))
}
