package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A server that cannot say which version it is confuses two things at once: the settings page shows the
// app an update it can never take, and every other client's "which release am I talking to" answer is
// wrong. The release workflow passes VERSION to these builds, so a Containerfile that compiles
// cmd/mochil has to declare it and hand it to the linker -- the narration image accepted the build-arg
// and silently ignored it for weeks, shipping binaries that called themselves 0.0.0-dev.
func TestEveryServerImageStampsItsVersion(t *testing.T) {
	root := "../.."
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "Containerfile") || entry.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if !strings.Contains(text, "./cmd/mochil") {
			continue // a runtime-only image, it does not build the server
		}
		checked++
		if !strings.Contains(text, "ARG VERSION") {
			t.Errorf("%s builds the server but never declares ARG VERSION, so a VERSION build-arg is ignored", name)
		}
		if !strings.Contains(text, "-X mochil/internal/version.Version=${VERSION}") {
			t.Errorf("%s builds the server without stamping the version into it", name)
		}
	}
	if checked == 0 {
		t.Fatal("no server Containerfile was found; this test is looking in the wrong place")
	}
}
