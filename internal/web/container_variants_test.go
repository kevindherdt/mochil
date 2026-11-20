package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The narration image is the same server as the standard one; only its base image and the voice
// runtime differ. Its command drifted anyway -- it kept the old single-folder form, "-library /books
// -cache /data/catalog.json", while the standard image had moved to "-config /config/config.json".
// Deploying it therefore started a server with no configuration: no account, no libraries, just the
// "create your account" page, on top of a data directory that was perfectly intact. Compose hid it,
// because compose.yaml overrides the command.
//
// Every one of these differences is invisible at build time and only shows up as "my server is
// empty" after a deploy, so they are checked here.
func TestTheNarrationImageRunsTheSameServerAsTheStandardOne(t *testing.T) {
	root := repoRoot(t)
	standard := readContainerfile(t, filepath.Join(root, "Containerfile"))
	narration := readContainerfile(t, filepath.Join(root, "Containerfile.narration"))

	for _, key := range []string{"CMD", "ENTRYPOINT", "VOLUME", "EXPOSE"} {
		want := directive(standard, key)
		got := directive(narration, key)
		if want == "" {
			t.Fatalf("Containerfile has no %s to compare against", key)
		}
		if got != want {
			t.Errorf("%s differs between the two images\n  standard:  %s\n  narration: %s", key, want, got)
		}
	}

	// -config-template points at /bootstrap/config.json, which only exists if the file was copied.
	if !strings.Contains(narration, "deployment/bootstrap.json /bootstrap/config.json") {
		t.Error("the narration image does not copy the bootstrap config its own CMD points at")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// These tests run in internal/web, two levels below the repository root.
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func readContainerfile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func directive(file, key string) string {
	for _, line := range strings.Split(file, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+" ") || strings.HasPrefix(trimmed, key+" [") {
			return trimmed
		}
	}
	return ""
}
