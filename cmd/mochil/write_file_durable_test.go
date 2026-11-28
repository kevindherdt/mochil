package main

import (
	"os"
	"path/filepath"
	"testing"
)

// An interrupted write must leave the previous file intact, not a truncated one: bootstrapConfig only copies the template
// when the file is missing, so a half-written config would keep every later boot failing.
func TestWriteFileDurableReplacesWithoutTruncatingInPlace(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	if err := writeFileDurable(path, []byte(`{"scan_every":"6h"}`), 0o600); err != nil {
		t.Fatalf("first write: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A write that fails must not touch the destination. A directory in place of the target makes the rename fail after
	// the temporary has been written and synced.
	blocked := filepath.Join(directory, "blocked")
	if err := os.Mkdir(blocked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeFileDurable(blocked, []byte("nope"), 0o600); err == nil {
		t.Fatal("writing over a directory must fail")
	}
	if err := writeFileDurable(path, []byte(`{"scan_every":"5m"}`), 0o600); err != nil {
		t.Fatalf("second write: %v", err)
	}
	// A rename publishes a different inode; writing in place would keep the old one, which is exactly the shape that let an
	// interrupted boot leave a config that exists and does not parse.
	firstInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeFileDurable(path, []byte(`{"scan_every":"1m"}`), 0o600); err != nil {
		t.Fatalf("third write: %v", err)
	}
	secondInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(firstInfo, secondInfo) {
		t.Fatal("the destination was written in place; a crash mid-write would leave it truncated")
	}
	if err := writeFileDurable(path, []byte(`{"scan_every":"5m"}`), 0o600); err != nil {
		t.Fatalf("fourth write: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != `{"scan_every":"5m"}` {
		t.Fatalf("content = %s", after)
	}
	if len(after) == 0 || string(before) == string(after) {
		t.Fatal("the second write did not land")
	}
	// No temporary may be left behind next to the file.
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "config.json" && entry.Name() != "blocked" {
			t.Fatalf("left a stray file: %s", entry.Name())
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
}
