package web

import "testing"

func TestAndroidCrashReportRejectsPathsAndUnboundedFrames(t *testing.T) {
	valid := androidCrashReport{Exception: "java.lang.IllegalStateException", AppVersion: "0.10.2", Android: "16", Device: "Samsung", Frames: []androidCrashFrame{{Class: "net.kdhmedia.Reader", Method: "open", File: "Reader.kt", Line: 42}}}
	if !validAndroidCrashReport(valid) {
		t.Fatal("valid bounded report was rejected")
	}
	valid.Frames[0].File = "/data/user/book.epub"
	if validAndroidCrashReport(valid) {
		t.Fatal("path-bearing report was accepted")
	}
}
