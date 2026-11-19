package web

import (
	"log/slog"
	"net/http"
	"regexp"
	"sort"
)

var androidDiagnosticName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

type androidDiagnosticEvent struct {
	At         int64  `json:"at"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	DurationMS int64  `json:"duration_ms,omitempty"`
	Outcome    string `json:"outcome"`
	Detail     string `json:"detail,omitempty"`
}

type androidDiagnosticReport struct {
	AppVersion string                   `json:"app_version"`
	Android    string                   `json:"android"`
	Device     string                   `json:"device"`
	Events     []androidDiagnosticEvent `json:"events"`
}

func (m *multiServer) androidDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var report androidDiagnosticReport
	if decodeJSONRequest(w, r, 64<<10, &report, false) != nil || !validAndroidDiagnosticReport(report) {
		http.Error(w, "Invalid diagnostics report", http.StatusBadRequest)
		return
	}
	logger := slog.Default()
	if m.root != nil && m.root.log != nil {
		logger = m.root.log
	}
	for _, event := range report.Events {
		logger.Info("android diagnostic",
			"app_version", report.AppVersion,
			"android", report.Android,
			"device", report.Device,
			"at", event.At,
			"kind", event.Kind,
			"name", event.Name,
			"duration_ms", event.DurationMS,
			"outcome", event.Outcome,
			"detail", event.Detail,
		)
	}
	if m.androidDiagnosticStore != nil {
		if err := m.androidDiagnosticStore.Append(report); err != nil {
			logger.Warn("persist android diagnostics", "error", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func validAndroidDiagnosticReport(report androidDiagnosticReport) bool {
	if len(report.Events) < 1 || len(report.Events) > 200 || len(report.AppVersion) > 32 || len(report.Android) > 32 || len(report.Device) > 120 {
		return false
	}
	allowedOutcome := map[string]bool{"ok": true, "error": true, "cancelled": true, "offline": true, "timeout": true}
	previous := int64(0)
	for _, event := range report.Events {
		if event.At < 1 || event.At < previous || !androidDiagnosticName.MatchString(event.Kind) || !androidDiagnosticName.MatchString(event.Name) || !allowedOutcome[event.Outcome] || event.DurationMS < 0 || event.DurationMS > 3_600_000 || event.Detail != "" {
			return false
		}
		previous = event.At
	}
	return sort.SliceIsSorted(report.Events, func(i, j int) bool { return report.Events[i].At <= report.Events[j].At })
}
