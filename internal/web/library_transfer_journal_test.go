package web

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTransferJournalSurvivesReopenAndFinishesEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library-transfers.json")
	journal, err := openTransferJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("opening an absent journal must not create it: %v", err)
	}
	entry, err := journal.begin(transferJournalEntry{
		From:  "books",
		To:    "shelf",
		Mode:  "move",
		Books: map[string]string{"a1": "b1"},
		Paths: []string{"Dune/Dune 1.epub"},
	})
	if err != nil || entry.ID == "" || entry.CreatedAt == 0 {
		t.Fatalf("begin = %+v, %v", entry, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("the journal file must be private: %v %v", info, err)
	}
	reopened, err := openTransferJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	open := reopened.open()
	if len(open) != 1 || open[0].ID != entry.ID || open[0].From != "books" || open[0].To != "shelf" || open[0].Mode != "move" || open[0].Books["a1"] != "b1" || len(open[0].Paths) != 1 || open[0].Paths[0] != "Dune/Dune 1.epub" {
		t.Fatalf("an entry must survive a reopen: %+v", open)
	}
	if err := reopened.markDone(entry.ID, transferStepFiles); err != nil {
		t.Fatal(err)
	}
	if err := reopened.skip(entry.ID, transferStepMetadata, "a copy keeps its source values"); err != nil {
		t.Fatal(err)
	}
	if err := reopened.fail(entry.ID, transferStepReading, os.ErrPermission); err != nil {
		t.Fatal(err)
	}
	again, err := openTransferJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	open = again.open()
	if len(open) != 1 || !open[0].done(transferStepFiles) || open[0].Skipped[transferStepMetadata] == "" || open[0].Failed != transferStepReading || open[0].Failure == "" || open[0].Attempts != 1 {
		t.Fatalf("bookkeeping must survive a reopen: %+v", open)
	}
	if err := again.finish(entry.ID); err != nil {
		t.Fatal(err)
	}
	finished, err := openTransferJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(finished.open()) != 0 {
		t.Fatalf("a finished entry must be gone: %+v", finished.open())
	}
}
