package web

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

var androidCrashSymbol = regexp.MustCompile(`^[A-Za-z0-9_.$<>-]{1,240}$`)

type androidCrashFrame struct {
	Class  string `json:"class"`
	Method string `json:"method"`
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
}

type androidCrashReport struct {
	Exception  string              `json:"exception"`
	AppVersion string              `json:"app_version"`
	Android    string              `json:"android"`
	Device     string              `json:"device"`
	Frames     []androidCrashFrame `json:"frames"`
}

func (m *multiServer) androidCrashReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var report androidCrashReport
	if decodeJSONRequest(w, r, 64<<10, &report, false) != nil || !validAndroidCrashReport(report) {
		http.Error(w, "Invalid crash report", http.StatusBadRequest)
		return
	}
	logger := slog.Default()
	if m.root != nil && m.root.log != nil {
		logger = m.root.log
	}
	logger.Error("android crash report",
		"exception", report.Exception,
		"app_version", report.AppVersion,
		"android", report.Android,
		"device", report.Device,
		"frames", report.Frames,
	)
	w.WriteHeader(http.StatusNoContent)
}

func validAndroidCrashReport(report androidCrashReport) bool {
	if !androidCrashSymbol.MatchString(report.Exception) || len(report.Frames) < 1 || len(report.Frames) > 80 || len(report.AppVersion) > 32 || len(report.Android) > 32 || len(report.Device) > 120 {
		return false
	}
	for _, frame := range report.Frames {
		if !androidCrashSymbol.MatchString(frame.Class) || !androidCrashSymbol.MatchString(frame.Method) || frame.Line < -1 || frame.Line > 10_000_000 {
			return false
		}
		if frame.File != "" && (!androidCrashSymbol.MatchString(frame.File) || strings.ContainsAny(frame.File, `/\\`)) {
			return false
		}
	}
	return true
}
