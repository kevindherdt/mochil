package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A server that hosts the app must be able to do it by dropping the build into its data directory;
// requiring an environment variable meant a fresh install answered 404 to "check for updates" with
// no hint about why.
func TestAndroidAPKIsFoundInTheDataDirectoryWithoutConfiguration(t *testing.T) {
	dir := t.TempDir()
	apk := filepath.Join(dir, "mochil-android.apk")
	if err := os.WriteFile(apk, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOCHIL_ANDROID_APK", "")
	t.Setenv("MOCHIL_DATA_DIR", t.TempDir())
	if got := androidAPKPath(dir); got != apk {
		t.Fatalf("want %q, got %q", apk, got)
	}
}

func TestAndroidAPKIsEmptyWhenNothingIsHosted(t *testing.T) {
	t.Setenv("MOCHIL_ANDROID_APK", "")
	t.Setenv("MOCHIL_DATA_DIR", t.TempDir())
	if got := androidAPKPath(t.TempDir()); got != "" {
		t.Fatalf("want no app, got %q", got)
	}
}

func TestConfiguredAndroidAPKWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mochil-android.apk"), []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOCHIL_ANDROID_APK", "/srv/custom.apk")
	t.Setenv("MOCHIL_DATA_DIR", t.TempDir())
	if got := androidAPKPath(dir); got != "/srv/custom.apk" {
		t.Fatalf("want the configured path, got %q", got)
	}
}

func TestADirectoryIsNotAnApp(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "mochil-android.apk"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOCHIL_ANDROID_APK", "")
	t.Setenv("MOCHIL_DATA_DIR", t.TempDir())
	if got := androidAPKPath(dir); got != "" {
		t.Fatalf("a directory must not be served as the app, got %q", got)
	}
}

// The container's data volume is /data while its cache directory lives under /config, and a build
// dropped in either place must be found: "put the file somewhere sensible" is the whole point.
func TestAndroidAPKIsFoundInTheDataVolume(t *testing.T) {
	volume := t.TempDir()
	if err := os.WriteFile(filepath.Join(volume, "mochil-android.apk"), []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOCHIL_ANDROID_APK", "")
	t.Setenv("MOCHIL_DATA_DIR", volume)
	if got := androidAPKPath(t.TempDir()); got != filepath.Join(volume, "mochil-android.apk") {
		t.Fatalf("want the build from the data volume, got %q", got)
	}
}
