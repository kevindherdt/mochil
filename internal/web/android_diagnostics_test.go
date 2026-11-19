package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidAndroidDiagnosticReport(t *testing.T) {
	valid := androidDiagnosticReport{
		AppVersion: "0.10.4", Android: "16", Device: "Example phone",
		Events: []androidDiagnosticEvent{{At: 1, Kind: "network", Name: "load_dashboard", DurationMS: 42, Outcome: "ok"}},
	}
	if !validAndroidDiagnosticReport(valid) {
		t.Fatal("valid bounded report rejected")
	}
	invalid := valid
	invalid.Events = []androidDiagnosticEvent{{At: 1, Kind: "network", Name: "private/path", Outcome: "ok"}}
	if validAndroidDiagnosticReport(invalid) {
		t.Fatal("unbounded diagnostic name accepted")
	}
	invalid = valid
	invalid.Events = []androidDiagnosticEvent{{At: 2, Kind: "network", Name: "one", Outcome: "ok"}, {At: 1, Kind: "network", Name: "two", Outcome: "ok"}}
	if validAndroidDiagnosticReport(invalid) {
		t.Fatal("out-of-order events accepted")
	}
	invalid = valid
	invalid.Events[0].Detail = "private detail"
	if validAndroidDiagnosticReport(invalid) {
		t.Fatal("diagnostic detail accepted")
	}
}

func TestAndroidDiagnosticStorePersistsAndRotates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "android-diagnostics.jsonl")
	store := newAndroidDiagnosticStore(path, 180)
	report := androidDiagnosticReport{
		AppVersion: "0.16.6", Android: "16", Device: "Test device",
		Events: []androidDiagnosticEvent{{At: 1, Kind: "playback", Name: "video_reconnect_retry", Outcome: "ok"}},
	}
	if err := store.Append(report); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "video_reconnect_retry") {
		t.Fatalf("diagnostic report was not persisted: %q %v", data, err)
	}
	if err := store.Append(report); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".previous"); err != nil {
		t.Fatalf("bounded journal did not rotate: %v", err)
	}
}
