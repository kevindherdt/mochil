package web

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRecoveringMovesAsideAUnreadableFileAndStartsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	if err := os.WriteFile(path, []byte("{cut off"), 0o600); err != nil {
		t.Fatal(err)
	}
	opens := 0
	open := func() (string, error) {
		opens++
		if data, err := os.ReadFile(path); err == nil && strings.HasPrefix(string(data), "{") {
			return "", errors.New("unexpected EOF")
		}
		return "empty store", nil
	}
	store, err := openRecovering("queue", path, open)
	if err != nil || store != "empty store" || opens != 2 {
		t.Fatalf("store=%q err=%v opens=%d", store, err, opens)
	}
	aside, _ := filepath.Glob(path + ".corrupt-*")
	if len(aside) != 1 {
		t.Fatalf("expected the damaged file to be kept aside, got %v", aside)
	}
	if kept, _ := os.ReadFile(aside[0]); string(kept) != "{cut off" {
		t.Fatalf("damaged file changed: %q", kept)
	}
}

func TestOpenRecoveringLeavesAHealthyOrMissingFileAlone(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "none.json")
	if _, err := openRecovering("x", missing, func() (int, error) { return 0, errors.New("permission denied") }); err == nil {
		t.Fatal("an error with no file to move aside must be returned")
	}
	if got, err := openRecovering("x", missing, func() (int, error) { return 7, nil }); got != 7 || err != nil {
		t.Fatalf("healthy open = %d %v", got, err)
	}
}
