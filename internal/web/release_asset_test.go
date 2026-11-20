package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The app falls back to the published release when a paired server offers an update but serves a
// file that is missing or older than the version it advertises. That fallback is a hard-coded URL in
// the Android build, which nothing else would catch if it were mistyped, pointed at a stale release,
// or changed without the README and the release workflow following.
//
// This is a drift gate, not a network test: it checks that the constant is the canonical download
// path for this repository and that the file it expects is the one CI attaches.
func TestTheAppFallsBackToTheCanonicalReleaseAsset(t *testing.T) {
	root := repoRoot(t)
	gradle := readContainerfile(t, filepath.Join(root, "android", "app", "build.gradle.kts"))

	const canonical = "https://github.com/kevindherdt/mochil/releases/latest/download/mochil-android.apk"
	if !strings.Contains(gradle, canonical) {
		t.Fatalf("the direct build does not carry the canonical release URL %s", canonical)
	}

	// The release workflow must attach exactly that asset name, or the fallback 404s at the one
	// moment it is needed.
	workflow := readContainerfile(t, filepath.Join(root, ".github", "workflows", "release.yml"))
	if !strings.Contains(workflow, "mochil-android.apk") {
		t.Fatal("the release workflow no longer attaches mochil-android.apk, which the fallback URL points at")
	}

	// And the README tells people the same address, so the two cannot drift apart.
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "releases/latest/download/mochil-android.apk") {
		t.Error("the README no longer links the permanent download the app falls back to")
	}
}
