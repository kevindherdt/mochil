package vaultwriter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

var tinyPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func TestAttachmentsAreCheckedAndNeverOverwrite(t *testing.T) {
	root := t.TempDir()
	writer, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := writer.CreateAttachment("../shot.PNG", tinyPNG)
	if err != nil || first != "shot.png" {
		t.Fatalf("first = %q, %v", first, err)
	}
	second, err := writer.CreateAttachment("shot.png", tinyPNG)
	if err != nil || second != "shot 1.png" {
		t.Fatalf("second = %q, %v", second, err)
	}
	if _, err = writer.CreateAttachment("note.png", []byte("<script>alert(1)</script>")); !errors.Is(err, ErrUnsupportedAttachment) {
		t.Fatalf("disguised file accepted: %v", err)
	}
	if _, err = writer.CreateAttachment("page.svg", tinyPNG); !errors.Is(err, ErrUnsupportedAttachment) {
		t.Fatalf("svg accepted: %v", err)
	}
	if _, err = writer.CreateAttachment(".hidden.png", tinyPNG); err != nil {
		t.Fatalf("leading dots are stripped, not refused: %v", err)
	}
	if _, err = os.Stat(filepath.Join(root, "hidden.png")); err != nil {
		t.Fatal(err)
	}
}

func TestMoveFolderRefusesExistingAndSelfNesting(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"A/Sub", "B"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "A/Sub/p.md"), []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	writer, _ := New(root)
	if err := writer.MoveFolder("A", "B"); !errors.Is(err, ErrExists) {
		t.Fatalf("existing target: %v", err)
	}
	if err := writer.MoveFolder("A", "A/Sub/Deeper"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("nesting into itself: %v", err)
	}
	if err := writer.MoveFolder("A", "Renamed"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "Renamed/Sub/p.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "A")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old folder still there")
	}
}
