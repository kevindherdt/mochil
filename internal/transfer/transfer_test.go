package transfer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func exists(root, rel string) bool {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func setup(t *testing.T) (*Mover, string, string) {
	t.Helper()
	base := t.TempDir()
	from, to := filepath.Join(base, "one"), filepath.Join(base, "two")
	for _, dir := range []string{from, to} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mover, err := New(from, to)
	if err != nil {
		t.Fatal(err)
	}
	return mover, from, to
}

func TestMoveCarriesTheBookAndItsDescriptionAndTidiesUp(t *testing.T) {
	mover, from, to := setup(t)
	write(t, from, "Dune/Dune.epub", "book")
	write(t, from, "Dune/metadata.opf", "<package/>")
	done, err := mover.Run([]Entry{{Source: "Dune/Dune.epub", Target: "Frank Herbert/Dune.epub"}}, true)
	if err != nil || len(done) != 2 {
		t.Fatalf("done = %v, err = %v", done, err)
	}
	if !exists(to, "Frank Herbert/Dune.epub") || !exists(to, "Frank Herbert/metadata.opf") {
		t.Fatal("the book and its description should be in the other library")
	}
	if exists(from, "Dune/Dune.epub") || exists(from, "Dune") {
		t.Fatal("the book and its now-empty folder should be gone from the source")
	}
}

func TestCopyLeavesTheOriginalAndNeverOverwrites(t *testing.T) {
	mover, from, to := setup(t)
	write(t, from, "a.epub", "original")
	write(t, to, "taken.epub", "someone else")
	if _, err := mover.Run([]Entry{{Source: "a.epub", Target: "taken.epub"}}, false); err == nil {
		t.Fatal("a name that is taken must be refused")
	}
	if body, _ := os.ReadFile(filepath.Join(to, "taken.epub")); string(body) != "someone else" {
		t.Fatal("an existing file was overwritten")
	}
	if _, err := mover.Run([]Entry{{Source: "a.epub", Target: "b.epub"}}, false); err != nil {
		t.Fatal(err)
	}
	if !exists(from, "a.epub") || !exists(to, "b.epub") {
		t.Fatal("a copy keeps the original")
	}
}

func TestCompanionsStayBehindWhenSeveralBooksShareAFolder(t *testing.T) {
	mover, from, to := setup(t)
	write(t, from, "Series/one.epub", "1")
	write(t, from, "Series/two.epub", "2")
	write(t, from, "Series/metadata.opf", "belongs to someone")
	done, err := mover.Run([]Entry{{Source: "Series/one.epub", Target: "one.epub"}}, true)
	if err != nil || len(done) != 1 || exists(to, "metadata.opf") || !exists(from, "Series/metadata.opf") {
		t.Fatalf("a shared folder's description must not travel: %v %v", done, err)
	}
}

func TestCheckExplainsWhatWouldGoWrong(t *testing.T) {
	mover, from, to := setup(t)
	write(t, from, "a.epub", "a")
	write(t, from, "b.epub", "b")
	write(t, to, "x.epub", "x")
	problems := mover.Check([]Entry{
		{Source: "a.epub", Target: "x.epub"},
		{Source: "b.epub", Target: "y.epub"},
		{Source: "a.epub", Target: "Y.epub"},
		{Source: "missing.epub", Target: "z.epub"},
		{Source: "../escape.epub", Target: "w.epub"},
	})
	want := []string{"already has", "", "same name", "no longer there", "not inside"}
	for i, problem := range problems {
		if !strings.Contains(problem.Reason, want[i]) {
			t.Fatalf("entry %d reason = %q, want it to contain %q", i, problem.Reason, want[i])
		}
	}
}

func TestUndoPutsBackAMoveAndRemovesACopy(t *testing.T) {
	mover, from, to := setup(t)
	write(t, from, "Dune/Dune.epub", "book")
	write(t, from, "Dune/metadata.opf", "<package/>")
	moved, err := mover.Run([]Entry{{Source: "Dune/Dune.epub", Target: "D/Dune.epub"}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := mover.Undo(moved, true); err != nil {
		t.Fatal(err)
	}
	if !exists(from, "Dune/Dune.epub") || !exists(from, "Dune/metadata.opf") || exists(to, "D") {
		t.Fatal("undoing a move should restore the source and tidy the destination")
	}
	copied, err := mover.Run([]Entry{{Source: "Dune/Dune.epub", Target: "C/Dune.epub"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := mover.Undo(copied, false); err != nil || exists(to, "C/Dune.epub") || !exists(from, "Dune/Dune.epub") {
		t.Fatalf("undoing a copy should delete only the copy: %v", err)
	}
}

func TestUndoRefusesToDeleteAFileThatChanged(t *testing.T) {
	mover, from, to := setup(t)
	write(t, from, "a.epub", "original")
	copied, _ := mover.Run([]Entry{{Source: "a.epub", Target: "b.epub"}}, false)
	write(t, to, "b.epub", "edited since, so a different size")
	if err := mover.Undo(copied, false); err == nil || !exists(to, "b.epub") {
		t.Fatal("a file that changed after the copy must be left alone")
	}
}

func TestOverlappingFoldersAreRefused(t *testing.T) {
	base := t.TempDir()
	inner := filepath.Join(base, "inner")
	_ = os.MkdirAll(inner, 0o755)
	if _, err := New(base, inner); err == nil {
		t.Fatal("a library inside another must be refused")
	}
}

// Across two disks a move cannot link, so it copies, checks and then removes the original.
func TestMoveAcrossDisksCopiesThenRemoves(t *testing.T) {
	shm, err := os.MkdirTemp("/dev/shm", "transfer-test-")
	if err != nil {
		t.Skip("no second filesystem available")
	}
	defer os.RemoveAll(shm)
	other := t.TempDir()
	probe, probeErr := os.Create(filepath.Join(shm, "probe"))
	if probeErr != nil {
		t.Skip("cannot write to the second filesystem")
	}
	probe.Close()
	if err := os.Link(filepath.Join(shm, "probe"), filepath.Join(other, "probe")); err == nil {
		t.Skip("both folders are on one filesystem")
	}
	mover, err := New(shm, other)
	if err != nil {
		t.Fatal(err)
	}
	write(t, shm, "a.epub", "across")
	if _, err := mover.Run([]Entry{{Source: "a.epub", Target: "A/a.epub"}}, true); err != nil {
		t.Fatal(err)
	}
	if body, _ := os.ReadFile(filepath.Join(other, "A", "a.epub")); string(body) != "across" || exists(shm, "a.epub") {
		t.Fatal("the book should have arrived intact and left the source")
	}
}

// A move across filesystems copies and then unlinks; a restart between the two
// finds both files, and only identical content makes finishing that move safe.
func TestSameComparesContent(t *testing.T) {
	base := t.TempDir()
	left, right := filepath.Join(base, "left.epub"), filepath.Join(base, "right.epub")
	write(t, base, "left.epub", "same bytes")
	write(t, base, "right.epub", "same bytes")
	if !Same(left, right) {
		t.Fatal("identical files compared different")
	}
	write(t, base, "right.epub", "other bytes")
	if Same(left, right) {
		t.Fatal("different files compared equal")
	}
	write(t, base, "right.epub", "same bytes!")
	if Same(left, right) {
		t.Fatal("a size difference compared equal")
	}
}
