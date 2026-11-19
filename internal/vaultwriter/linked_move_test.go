package vaultwriter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkedMoveSyncsDestinationBeforeRemovingOriginal(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "sync failure"}[fail], func(t *testing.T) {
			root := t.TempDir()
			source, target := filepath.Join(root, "source.md"), filepath.Join(root, "target.md")
			if err := os.WriteFile(source, []byte("keep"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Link(source, target); err != nil {
				t.Fatal(err)
			}
			failure := errors.New("sync failed")
			calls := 0
			err := completeLinkedMove(source, target, func(dir string) error {
				calls++
				if dir != root {
					t.Fatalf("unexpected directory %s", dir)
				}
				_, sourceErr := os.Stat(source)
				if calls == 1 && sourceErr != nil {
					t.Fatal("original removed before destination sync")
				}
				if calls == 2 && !errors.Is(sourceErr, os.ErrNotExist) {
					t.Fatal("original still present after removal")
				}
				if fail {
					return failure
				}
				return nil
			})
			if fail && (!errors.Is(err, failure) || calls != 1) {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
			if !fail && (err != nil || calls != 2) {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
			if fail {
				if _, err := os.Stat(source); err != nil {
					t.Fatal("original lost after failed sync")
				}
			}
			if content, err := os.ReadFile(target); err != nil || string(content) != "keep" {
				t.Fatalf("destination=%q error=%v", content, err)
			}
		})
	}
}

func TestLinkedMoveAcrossDirectoriesSyncsInRecoveryOrder(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "Archive")
	if err := os.Mkdir(archive, 0o700); err != nil {
		t.Fatal(err)
	}
	source, target := filepath.Join(root, "Note.md"), filepath.Join(archive, "Note.md")
	if err := os.WriteFile(source, []byte("note"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(source, target); err != nil {
		t.Fatal(err)
	}
	var directories []string
	if err := completeLinkedMove(source, target, func(dir string) error {
		directories = append(directories, dir)
		return syncDirectory(dir)
	}); err != nil {
		t.Fatal(err)
	}
	if len(directories) != 2 || directories[0] != archive || directories[1] != root {
		t.Fatalf("sync order=%v", directories)
	}
}
