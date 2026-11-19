package main

import (
	"os"
	"strings"
	"testing"
)

// The prefix moved from OPDSHELF_ to MOCHIL_ with the rename. An install that still has the old
// variables in its compose file has to keep booting, and a half-migrated one must not silently pick
// the stale value.
func TestEnvReadsBothPrefixes(t *testing.T) {
	t.Setenv("MOCHIL_TEST_VALUE", "")
	t.Setenv("OPDSHELF_TEST_VALUE", "old")
	if got := env("MOCHIL_TEST_VALUE", "fallback"); got != "old" {
		t.Fatalf("legacy name not honoured: got %q", got)
	}

	t.Setenv("MOCHIL_TEST_VALUE", "new")
	if got := env("MOCHIL_TEST_VALUE", ""); got != "new" {
		t.Fatalf("new name must win when both are set: got %q", got)
	}

	os.Unsetenv("MOCHIL_TEST_VALUE")
	os.Unsetenv("OPDSHELF_TEST_VALUE")
	if got := env("MOCHIL_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("fallback not used: got %q", got)
	}
}

func TestEnvIntAndDurationUseTheSameLookup(t *testing.T) {
	t.Setenv("MOCHIL_TEST_INT", "")
	t.Setenv("OPDSHELF_TEST_INT", "7")
	if got := envInt("MOCHIL_TEST_INT", 1); got != 7 {
		t.Fatalf("legacy integer name not honoured: got %d", got)
	}
	t.Setenv("OPDSHELF_TEST_DURATION", "90s")
	if got := envDuration("MOCHIL_TEST_DURATION", 0); got.String() != "1m30s" {
		t.Fatalf("legacy duration name not honoured: got %s", got)
	}
}

// A name without the new prefix is read verbatim: these helpers are used for PATH-looking values
// such as MOCHIL_FFMPEG, where inventing a fallback would be worse than returning nothing.
func TestEnvDoesNotInventPrefixes(t *testing.T) {
	if got := lookupEnv("PATH"); !strings.Contains(got, "/") {
		t.Fatalf("PATH should still be readable, got %q", got)
	}
}
