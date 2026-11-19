package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The narration image ships the voice runtime at /opt/voice/bin/python. Choosing that image used to
// be enough to install the runtime but not enough to have narration work: the engine reports "not
// set up" unless the interpreter's path is written into the config, so a server running exactly the
// image built for narration still showed "Narration is not set up on this server yet".
//
// The path is detected now, and an explicit setting always wins so a hand-installed runtime is not
// disturbed.
func TestTheBundledVoiceRuntimeIsUsedWhenNoneIsConfigured(t *testing.T) {
	// This test can only assert the negative (nothing configured, nothing to find) on a machine
	// without the runtime, plus the precedence rule. The positive case needs the real path, which is
	// built into the narration image.
	// Pretend the image is in place: a real interpreter path, so the positive case runs here too.
	dir := t.TempDir()
	runtime := filepath.Join(dir, "python")
	if err := os.WriteFile(runtime, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	restore := bundledVoicePython
	bundledVoicePython = runtime
	t.Cleanup(func() { bundledVoicePython = restore })

	cfg, err := Decode(strings.NewReader(`{"libraries":[]}`), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Narration.Python != runtime {
		t.Fatalf("bundled runtime present but not detected: %q", cfg.Narration.Python)
	}
	if cfg.Narration.Cache != "/data/voice" {
		t.Fatalf("bundled runtime should cache on the volume, got %q", cfg.Narration.Cache)
	}

	// And with nothing installed, nothing is invented.
	bundledVoicePython = filepath.Join(dir, "missing-python")
	empty, err := Decode(strings.NewReader(`{"libraries":[]}`), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if empty.Narration.Python != "" {
		t.Fatalf("no runtime installed but a python path was invented: %q", empty.Narration.Python)
	}
}

func TestAnExplicitVoiceRuntimeIsNeverOverridden(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "python")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Decode(strings.NewReader(`{"libraries":[],"narration":{"python":"`+custom+`","cache":"`+dir+`"}}`), dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Narration.Python != custom {
		t.Fatalf("configured runtime was replaced: %q", cfg.Narration.Python)
	}
	if cfg.Narration.Cache != dir {
		t.Fatalf("configured cache was replaced: %q", cfg.Narration.Cache)
	}
}
