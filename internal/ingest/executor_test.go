package ingest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestExecutor(t *testing.T, limit int64) (*Executor, string, string, *JournalStore) {
	t.Helper()
	root := t.TempDir()
	incoming := filepath.Join(root, "incoming")
	managed := filepath.Join(root, "managed")
	if err := os.MkdirAll(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	journal := NewJournalStore(filepath.Join(root, "state", "ingest.json"))
	executor, err := NewExecutor(incoming, managed, journal, limit)
	if err != nil {
		t.Fatal(err)
	}
	return executor, incoming, managed, journal
}

func TestExecutorPlansCopiesVerifiesAndKeepsSource(t *testing.T) {
	executor, incoming, managed, journal := newTestExecutor(t, 1<<20)
	body := []byte("verified publication")
	if err := os.MkdirAll(filepath.Join(incoming, "drop"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(incoming, "drop", "book.epub")
	if err := os.WriteFile(source, body, 0o600); err != nil {
		t.Fatal(err)
	}

	planned, err := executor.Plan([]PlanEntry{{SourcePath: "drop/book.epub", TargetPath: "Author/Series/book.epub"}})
	if err != nil || planned.State != TransactionPlanned {
		t.Fatalf("plan = %#v, %v", planned, err)
	}
	if _, err = os.Stat(filepath.Join(managed, "Author", "Series", "book.epub")); !os.IsNotExist(err) {
		t.Fatal("planning created the target media file")
	}
	committed, err := executor.Execute(planned.ID)
	if err != nil || committed.State != TransactionCommitted || committed.Entries[0].SHA256 == "" {
		t.Fatalf("execute = %#v, %v", committed, err)
	}
	got, err := os.ReadFile(filepath.Join(managed, "Author", "Series", "book.epub"))
	if err != nil || string(got) != string(body) {
		t.Fatalf("target = %q, %v", got, err)
	}
	if got, err = os.ReadFile(source); err != nil || string(got) != string(body) {
		t.Fatalf("source was modified = %q, %v", got, err)
	}
	saved, err := journal.Load()
	if err != nil || len(saved.Transactions) != 1 || saved.Transactions[0].State != TransactionCommitted {
		t.Fatalf("journal = %#v, %v", saved, err)
	}
}

func TestExecutorRejectsOverlappingResolvedRoots(t *testing.T) {
	root := t.TempDir()
	incoming := filepath.Join(root, "incoming")
	managed := filepath.Join(incoming, "managed")
	if err := os.MkdirAll(managed, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := NewExecutor(incoming, managed, NewJournalStore(filepath.Join(root, "journal.json")), 1<<20); err == nil {
		t.Fatal("expected overlapping roots to be rejected")
	}
}

func TestExecutorDiscardsOnlyUnstartedPlan(t *testing.T) {
	executor, incoming, managed, _ := newTestExecutor(t, 1<<20)
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	planned, err := executor.Plan([]PlanEntry{{SourcePath: "book.epub", TargetPath: "book.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	discarded, err := executor.Discard(planned.ID)
	if err != nil || discarded.State != TransactionRolledBack {
		t.Fatalf("discard = %#v, %v", discarded, err)
	}
	if _, err = os.Stat(filepath.Join(incoming, "book.epub")); err != nil {
		t.Fatalf("source changed: %v", err)
	}
	if _, err = os.Stat(filepath.Join(managed, "book.epub")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("discard created target: %v", err)
	}
	if _, err = executor.Discard(planned.ID); err == nil {
		t.Fatal("expected terminal transaction discard to fail")
	}
}

func TestExecutorCancellationRollsBackWithoutPublishingTarget(t *testing.T) {
	executor, incoming, managed, journal := newTestExecutor(t, 1<<20)
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	planned, err := executor.Plan([]PlanEntry{{SourcePath: "book.epub", TargetPath: "Library/book.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rolledBack, err := executor.ExecuteContext(ctx, planned.ID)
	if !errors.Is(err, context.Canceled) || rolledBack.State != TransactionRolledBack {
		t.Fatalf("cancelled execute = %#v, %v", rolledBack, err)
	}
	if _, statErr := os.Stat(filepath.Join(managed, "Library", "book.epub")); !os.IsNotExist(statErr) {
		t.Fatal("cancelled ingest published a target")
	}
	state, loadErr := journal.Load()
	if loadErr != nil || state.Transactions[0].State != TransactionRolledBack {
		t.Fatalf("cancelled journal = %#v, %v", state, loadErr)
	}
}

func TestExecutorRefusesTraversalSymlinksAndOversizedSources(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 8)
	if err := os.WriteFile(filepath.Join(incoming, "large.epub"), []byte("123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.epub")
	if err := os.WriteFile(outside, []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(incoming, "linked.epub")); err != nil {
		t.Fatal(err)
	}
	for name, entry := range map[string]PlanEntry{
		"source traversal": {SourcePath: "../outside.epub", TargetPath: "book.epub"},
		"target traversal": {SourcePath: "large.epub", TargetPath: "../book.epub"},
		"symlink escape":   {SourcePath: "linked.epub", TargetPath: "book.epub"},
		"oversize":         {SourcePath: "large.epub", TargetPath: "book.epub"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := executor.Plan([]PlanEntry{entry}); err == nil {
				t.Fatal("unsafe ingest plan was accepted")
			}
		})
	}
}

func TestExecutorNeverOverwritesTargetAppearingAfterPlan(t *testing.T) {
	executor, incoming, managed, journal := newTestExecutor(t, 1<<20)
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("incoming"), 0o600); err != nil {
		t.Fatal(err)
	}
	planned, err := executor.Plan([]PlanEntry{{SourcePath: "book.epub", TargetPath: "Library/book.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(managed, "Library"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(managed, "Library", "book.epub")
	if err = os.WriteFile(target, []byte("someone else's file"), 0o600); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := executor.Execute(planned.ID)
	if err == nil || rolledBack.State != TransactionRolledBack {
		t.Fatalf("collision result = %#v, %v", rolledBack, err)
	}
	body, readErr := os.ReadFile(target)
	if readErr != nil || string(body) != "someone else's file" {
		t.Fatalf("existing target changed = %q, %v", body, readErr)
	}
	saved, loadErr := journal.Load()
	if loadErr != nil || !strings.Contains(saved.Transactions[0].Error, "nothing was overwritten") {
		t.Fatalf("collision was not durably explained: %#v, %v", saved, loadErr)
	}
}

func TestExecutorDoesNotClaimIdenticalTargetAppearingAfterPlan(t *testing.T) {
	executor, incoming, managed, _ := newTestExecutor(t, 1<<20)
	body := []byte("identical but externally created")
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	planned, err := executor.Plan([]PlanEntry{{SourcePath: "book.epub", TargetPath: "Library/book.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(managed, "Library", "book.epub")
	if err = os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(target, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = executor.Execute(planned.ID); err == nil {
		t.Fatal("target collision unexpectedly committed")
	}
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != string(body) {
		t.Fatalf("identical external target was removed or changed: %q, %v", got, readErr)
	}
}

func TestCommitPrimitiveNeverClobbersAnExistingTarget(t *testing.T) {
	executor, _, managed, _ := newTestExecutor(t, 1<<20)
	entry := JournalEntry{TargetPath: "Library/book.epub", TemporaryPath: ".ingest/test/book.part"}
	temporary := filepath.Join(managed, ".ingest", "test", "book.part")
	target := filepath.Join(managed, "Library", "book.epub")
	if err := os.MkdirAll(filepath.Dir(temporary), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temporary, []byte("verified temporary"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("existing target"), 0o600); err != nil {
		t.Fatal(err)
	}
	if linked, err := executor.commitEntry(entry); err == nil || linked {
		t.Fatal("no-clobber commit accepted an existing target")
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "existing target" {
		t.Fatalf("existing target was replaced: %q, %v", got, err)
	}
	if got, err := os.ReadFile(temporary); err != nil || string(got) != "verified temporary" {
		t.Fatalf("failed commit consumed its temporary file: %q, %v", got, err)
	}
}

func TestExecutorRemovesOnlyItsCommittedTargetsOnPartialFailure(t *testing.T) {
	executor, incoming, managed, _ := newTestExecutor(t, 1<<20)
	for _, name := range []string{"one.epub", "two.epub"} {
		if err := os.WriteFile(filepath.Join(incoming, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	planned, err := executor.Plan([]PlanEntry{
		{SourcePath: "one.epub", TargetPath: "Library/one.epub"},
		{SourcePath: "two.epub", TargetPath: "Library/two.epub"},
	})
	if err != nil {
		t.Fatal(err)
	}
	realCommit := executor.commit
	commits := 0
	executor.commit = func(entry JournalEntry) (bool, error) {
		commits++
		if commits == 2 {
			return false, errors.New("injected commit failure")
		}
		return realCommit(entry)
	}
	rolledBack, err := executor.Execute(planned.ID)
	if err == nil || rolledBack.State != TransactionRolledBack {
		t.Fatalf("partial commit = %#v, %v", rolledBack, err)
	}
	if _, statErr := os.Stat(filepath.Join(managed, "Library", "one.epub")); !os.IsNotExist(statErr) {
		t.Fatal("executor-owned first target survived rollback")
	}
	if _, statErr := os.Stat(filepath.Join(managed, "Library", "two.epub")); !os.IsNotExist(statErr) {
		t.Fatal("failed second target unexpectedly exists")
	}
}

func TestJournalRotationKeepsRecoverableTransactions(t *testing.T) {
	transactions := make([]Transaction, 0, maximumTransactions)
	for index := 0; index < maximumTransactions-1; index++ {
		transactions = append(transactions, Transaction{ID: "done", State: TransactionCommitted})
	}
	active := Transaction{ID: "active", State: TransactionCopying}
	transactions = append(transactions, active)

	retained := journalWithRoom(transactions, maximumTransactions-1)
	if len(retained) != maximumTransactions-1 {
		t.Fatalf("retained %d transactions", len(retained))
	}
	foundActive := false
	for _, transaction := range retained {
		foundActive = foundActive || transaction.ID == active.ID
	}
	if !foundActive {
		t.Fatal("journal rotation evicted a recoverable transaction")
	}
}

func TestRecoverRestartsInterruptedCopy(t *testing.T) {
	executor, incoming, managed, journal := newTestExecutor(t, 1<<20)
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	planned, err := executor.Plan([]PlanEntry{{SourcePath: "book.epub", TargetPath: "Library/book.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	state, _ := journal.Load()
	interrupted, err := Transition(planned, TransactionCopying, executor.now(), "")
	if err != nil {
		t.Fatal(err)
	}
	state.Transactions[0] = interrupted
	if err = journal.Save(state); err != nil {
		t.Fatal(err)
	}
	stale := executor.target(interrupted.Entries[0].TemporaryPath)
	if err = os.MkdirAll(filepath.Dir(stale), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(stale, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}

	recovered, steps, err := executor.Recover(planned.ID)
	if err != nil || recovered.State != TransactionCommitted {
		t.Fatalf("recover copy = %#v, %#v, %v", recovered, steps, err)
	}
	if got, err := os.ReadFile(filepath.Join(managed, "Library", "book.epub")); err != nil || string(got) != "book" {
		t.Fatalf("recovered target = %q, %v", got, err)
	}
}

func TestRecoverFinalizesOnlyDigestMatchedCommittingTarget(t *testing.T) {
	executor, incoming, _, journal := newTestExecutor(t, 1<<20)
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	planned, err := executor.Plan([]PlanEntry{{SourcePath: "book.epub", TargetPath: "Library/book.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	committing := moveToCommitting(t, executor, journal, planned)
	if linked, err := executor.commitEntry(committing.Entries[0]); err != nil || !linked {
		t.Fatalf("simulate committed link = %v, %v", linked, err)
	}

	recovered, steps, err := executor.Recover(planned.ID)
	if err != nil || recovered.State != TransactionCommitted {
		t.Fatalf("recover commit = %#v, %#v, %v", recovered, steps, err)
	}
}

func TestRecoverLeavesMismatchedCommittingTargetForReview(t *testing.T) {
	executor, incoming, managed, journal := newTestExecutor(t, 1<<20)
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	planned, err := executor.Plan([]PlanEntry{{SourcePath: "book.epub", TargetPath: "Library/book.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	moveToCommitting(t, executor, journal, planned)
	target := filepath.Join(managed, "Library", "book.epub")
	if err = os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(target, []byte("different"), 0o600); err != nil {
		t.Fatal(err)
	}

	recovered, _, err := executor.Recover(planned.ID)
	if !errors.Is(err, ErrManualReview) || recovered.State != TransactionCommitting {
		t.Fatalf("mismatched recovery = %#v, %v", recovered, err)
	}
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != "different" {
		t.Fatalf("manual-review target changed = %q, %v", got, readErr)
	}
}

func moveToCommitting(t *testing.T, executor *Executor, journal *JournalStore, planned Transaction) Transaction {
	t.Helper()
	copying, err := Transition(planned, TransactionCopying, executor.now(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err = executor.copyEntry(context.Background(), &copying.Entries[0]); err != nil {
		t.Fatal(err)
	}
	verified, err := Transition(copying, TransactionVerified, executor.now(), "")
	if err != nil {
		t.Fatal(err)
	}
	committing, err := Transition(verified, TransactionCommitting, executor.now(), "")
	if err != nil {
		t.Fatal(err)
	}
	state, err := journal.Load()
	if err != nil {
		t.Fatal(err)
	}
	state.Transactions[0] = committing
	if err = journal.Save(state); err != nil {
		t.Fatal(err)
	}
	return committing
}

func TestCopiedBooksAreRememberedAcrossRestartsAndSeededFromTheJournal(t *testing.T) {
	executor, incoming, managed, journal := newTestExecutor(t, 1<<20)
	if err := os.WriteFile(filepath.Join(incoming, "a.epub"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "b.epub"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	planned, err := executor.Plan([]PlanEntry{{SourcePath: "a.epub", TargetPath: "A/a.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	if executor.Imported().Has("a.epub") {
		t.Fatal("a book counted as copied before it was copied")
	}
	if _, err = executor.Execute(planned.ID); err != nil {
		t.Fatal(err)
	}
	if !executor.Imported().Has("a.epub") || executor.Imported().Has("b.epub") {
		t.Fatal("only the copied book should be remembered")
	}
	// A new process (or a second device) reads the same server-side record.
	again, err := NewExecutor(incoming, managed, journal, 1<<20)
	if err != nil || !again.Imported().Has("a.epub") {
		t.Fatalf("restart forgot the copied book: %v", err)
	}
	// Losing the record still recovers what the journal knows.
	_ = os.Remove(strings.TrimSuffix(journal.path, ".json") + "-imported.json")
	seeded, err := NewExecutor(incoming, managed, journal, 1<<20)
	if err != nil || !seeded.Imported().Has("a.epub") {
		t.Fatalf("journal did not seed the record: %v", err)
	}
}
