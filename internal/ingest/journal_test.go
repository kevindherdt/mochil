package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validJournal() Journal {
	return Journal{Version: journalVersion, Transactions: []Transaction{{
		ID: "ingest-01", State: TransactionPlanned, CreatedAt: 100, UpdatedAt: 100,
		Entries: []JournalEntry{{SourcePath: "incoming/book.epub", TargetPath: "Author/Book/book.epub", TemporaryPath: ".ingest/ingest-01/book.part", Size: 42}},
	}}}
}

func TestJournalStoreRoundTripAndMissingState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "ingest.json")
	store := NewJournalStore(path)
	empty, err := store.Load()
	if err != nil || empty.Version != journalVersion || len(empty.Transactions) != 0 {
		t.Fatalf("missing journal = %#v, %v", empty, err)
	}
	wanted := validJournal()
	if err = store.Save(wanted); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil || len(got.Transactions) != 1 || got.Transactions[0].Entries[0].TargetPath != "Author/Book/book.epub" {
		t.Fatalf("journal = %#v, %v", got, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("journal permissions = %v, %v", info.Mode(), err)
	}
}

func TestJournalRejectsTraversalCorruptionAndOversize(t *testing.T) {
	for _, badPath := range []string{"../outside.epub", "/absolute.epub", "folder/../outside.epub", `folder\..\outside.epub`, ""} {
		journal := validJournal()
		journal.Transactions[0].Entries[0].TargetPath = badPath
		if err := validateJournal(journal); err == nil {
			t.Fatalf("accepted unsafe path %q", badPath)
		}
	}
	collision := validJournal()
	collision.Transactions[0].Entries = append(collision.Transactions[0].Entries, JournalEntry{SourcePath: "incoming/two.epub", TargetPath: "Author/Book/book.epub", TemporaryPath: ".ingest/ingest-01/two.part"})
	if err := validateJournal(collision); err == nil {
		t.Fatal("accepted colliding target paths")
	}
	directory := t.TempDir()
	for name, body := range map[string]string{
		"trailing.json":  `{"version":1,"transactions":[]} {}`,
		"unknown.json":   `{"version":9,"transactions":[]}`,
		"duplicate.json": `{"version":1,"transactions":[{"id":"same","state":"planned","created_at":1,"updated_at":1,"entries":[{"source_path":"a","target_path":"b","temporary_path":"c","size":1}]},{"id":"same","state":"planned","created_at":1,"updated_at":1,"entries":[{"source_path":"d","target_path":"e","temporary_path":"f","size":1}]}]}`,
	} {
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewJournalStore(path).Load(); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	oversized := filepath.Join(directory, "oversized.json")
	if err := os.WriteFile(oversized, []byte(strings.Repeat("x", maximumJournalBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewJournalStore(oversized).Load(); err == nil {
		t.Fatal("accepted oversized journal")
	}
}

func TestTransactionTransitionsAreOneWayAndTimestamped(t *testing.T) {
	transaction := validJournal().Transactions[0]
	next, err := Transition(transaction, TransactionCopying, time.Unix(101, 0), "")
	if err != nil || next.State != TransactionCopying || next.UpdatedAt != 101 {
		t.Fatalf("transition = %#v, %v", next, err)
	}
	if _, err := Transition(next, TransactionCommitted, time.Unix(102, 0), ""); err == nil {
		t.Fatal("skipped verification and commit boundary")
	}
	if _, err := Transition(next, TransactionVerified, time.Unix(99, 0), ""); err == nil {
		t.Fatal("accepted a backwards timestamp")
	}
}

func TestRecoveryNeverOverwritesUnknownTargets(t *testing.T) {
	transaction := validJournal().Transactions[0]
	transaction.State = TransactionCommitting
	steps := PlanRecovery(transaction, []EntryObservation{{TargetExists: true, TargetMatches: false, BackupExists: true}})
	if len(steps) != 1 || steps[0].Action != RecoveryManualReview {
		t.Fatalf("unsafe recovery = %#v", steps)
	}
	transaction.State = TransactionVerified
	steps = PlanRecovery(transaction, []EntryObservation{{TemporaryExists: true}})
	if len(steps) != 1 || steps[0].Action != RecoveryResumeCommit {
		t.Fatalf("verified recovery = %#v", steps)
	}
	transaction.State = TransactionCommitting
	steps = PlanRecovery(transaction, []EntryObservation{{TargetExists: true, TargetMatches: true, BackupExists: true}})
	if len(steps) != 2 || steps[0].Action != RecoveryRemoveBackup || steps[1].Action != RecoveryMarkCommitted {
		t.Fatalf("completed recovery = %#v", steps)
	}
}
