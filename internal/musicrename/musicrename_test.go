package musicrename

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, files ...string) *Engine {
	t.Helper()
	root := t.TempDir()
	for _, name := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("content of "+name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return &Engine{Root: root, Dir: filepath.Join(t.TempDir(), "ledgers")}
}

func exists(e *Engine, name string) bool {
	_, err := os.Lstat(filepath.Join(e.Root, filepath.FromSlash(name)))
	return err == nil
}

func TestApplyRenamesCarriesLyricsWritesTheLedgerFirstAndUndoRestoresEverything(t *testing.T) {
	e := fixture(t, "dl/a.mp3", "dl/a.lrc", "dl/b.mp3", "dl/cover.jpg")
	ledger, err := e.Apply("music", []Move{{ID: "1", NewID: "n1", From: "dl/a.mp3", To: "dl/01 - A.mp3"}, {ID: "2", NewID: "n2", From: "dl/b.mp3", To: "dl/02 - B.mp3"}})
	if err != nil || ledger.State != "done" || len(ledger.Moves) != 3 {
		t.Fatalf("%+v %v", ledger, err)
	}
	for _, want := range []string{"dl/01 - A.mp3", "dl/01 - A.lrc", "dl/02 - B.mp3", "dl/cover.jpg"} {
		if !exists(e, want) {
			t.Fatalf("%s missing", want)
		}
	}
	if exists(e, "dl/a.mp3") || exists(e, "dl/a.lrc") {
		t.Fatal("old names must be gone")
	}
	if latest, ok := e.Latest("music"); !ok || latest.ID != ledger.ID {
		t.Fatal("the batch is the one to undo")
	}
	if _, err := e.Undo(ledger.ID); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"dl/a.mp3", "dl/a.lrc", "dl/b.mp3"} {
		if !exists(e, want) {
			t.Fatalf("undo did not restore %s", want)
		}
	}
	if body, _ := os.ReadFile(filepath.Join(e.Root, "dl/a.mp3")); string(body) != "content of dl/a.mp3" {
		t.Fatal("content must be unchanged")
	}
	if _, ok := e.Latest("music"); ok {
		t.Fatal("an undone batch is not offered again")
	}
	if _, err := e.Undo(ledger.ID); err == nil {
		t.Fatal("undoing twice must be refused")
	}
}

func TestNothingIsOverwrittenAndNothingMovesWhenAnyRenameIsRefused(t *testing.T) {
	e := fixture(t, "dl/a.mp3", "dl/b.mp3", "dl/01 - A.mp3")
	_, err := e.Apply("music", []Move{{From: "dl/b.mp3", To: "dl/02 - B.mp3"}, {From: "dl/a.mp3", To: "dl/01 - A.mp3"}})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
	if !exists(e, "dl/b.mp3") || exists(e, "dl/02 - B.mp3") {
		t.Fatal("the valid rename must not happen when the batch is refused")
	}
	if body, _ := os.ReadFile(filepath.Join(e.Root, "dl/01 - A.mp3")); string(body) != "content of dl/01 - A.mp3" {
		t.Fatal("the existing file was overwritten")
	}
}

func TestRefusesEscapesLinksSwapsAndDuplicates(t *testing.T) {
	e := fixture(t, "dl/a.mp3", "dl/b.mp3")
	if err := os.Symlink(filepath.Join(e.Root, "dl/a.mp3"), filepath.Join(e.Root, "dl/link.mp3")); err != nil {
		t.Fatal(err)
	}
	for _, moves := range [][]Move{
		{{From: "dl/a.mp3", To: "../x.mp3"}},
		{{From: "dl/a.mp3", To: "dl/sub/x.mp3"}},
		{{From: "dl/link.mp3", To: "dl/x.mp3"}},
		{{From: "dl/missing.mp3", To: "dl/x.mp3"}},
		{{From: "dl/a.mp3", To: "dl/b.mp3"}},
		{{From: "dl/a.mp3", To: "dl/x.mp3"}, {From: "dl/b.mp3", To: "dl/X.mp3"}},
		{{From: "dl/a.mp3", To: "dl/b.mp3"}, {From: "dl/b.mp3", To: "dl/a.mp3"}},
		nil,
	} {
		if _, err := e.Apply("music", moves); err == nil {
			t.Fatalf("%+v must be refused", moves)
		}
	}
	if !exists(e, "dl/a.mp3") || !exists(e, "dl/b.mp3") {
		t.Fatal("refused batches must leave the files alone")
	}
}

func TestAFailureHalfwayPutsTheAlreadyMovedFilesBack(t *testing.T) {
	e := fixture(t, "dl/a.mp3", "dl/b.mp3")
	// the second target's folder cannot be written to, so the second rename fails after the first succeeded
	if err := os.Chmod(filepath.Join(e.Root, "dl"), 0o500); err != nil || os.Geteuid() == 0 {
		t.Skip("needs a non-root user to make a folder read-only")
	}
	defer os.Chmod(filepath.Join(e.Root, "dl"), 0o750)
	if _, err := e.Apply("music", []Move{{From: "dl/a.mp3", To: "dl/01.mp3"}}); err == nil || !exists(e, "dl/a.mp3") {
		t.Fatalf("err = %v", err)
	}
}

func TestUndoReportsFilesItCannotPutBack(t *testing.T) {
	e := fixture(t, "dl/a.mp3", "dl/b.mp3")
	ledger, err := e.Apply("music", []Move{{From: "dl/a.mp3", To: "dl/01 - A.mp3"}, {From: "dl/b.mp3", To: "dl/02 - B.mp3"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Root, "dl/a.mp3"), []byte("someone else's file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Undo(ledger.ID); err == nil || !strings.Contains(err.Error(), "1 files could not be put back") {
		t.Fatalf("err = %v", err)
	}
	if !exists(e, "dl/b.mp3") || !exists(e, "dl/01 - A.mp3") {
		t.Fatal("the restorable file is restored and the blocked one is left, never overwritten")
	}
	if body, _ := os.ReadFile(filepath.Join(e.Root, "dl/a.mp3")); string(body) != "someone else's file" {
		t.Fatal("the other file was overwritten")
	}
}

func TestLedgerIdsCannotReachOtherFiles(t *testing.T) {
	e := fixture(t)
	for _, id := range []string{"../x", "a/b", "", ".."} {
		if _, err := e.Load(id); err == nil {
			t.Fatalf("%q", id)
		}
	}
}
