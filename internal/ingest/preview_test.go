package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBatchPreviewFindsConflictsWithoutCreatingJournalState(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	if err := os.MkdirAll(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "one.epub"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "two.epub"), []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor, err := NewExecutor(incoming, managed, NewJournalStore(filepath.Join(root, "journal.json")), 1024)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := executor.PreviewBatch([]PlanEntry{
		{SourcePath: "one.epub", TargetPath: "Author/Book.epub"},
		{SourcePath: "two.epub", TargetPath: "author/book.epub"},
	})
	if err != nil || preview.Ready || preview.Conflicts != 2 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	for _, entry := range preview.Entries {
		if len(entry.Issues) != 1 || entry.Issues[0].Code != "duplicate_target" {
			t.Fatalf("entry=%+v", entry)
		}
	}
	transactions, err := executor.Transactions()
	if err != nil || len(transactions) != 0 {
		t.Fatalf("preview mutated journal: %+v err=%v", transactions, err)
	}
}

func TestBatchPreviewBoundsAndValidatesEveryEntry(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	if err := os.MkdirAll(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "large.epub"), []byte("too large"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor, err := NewExecutor(incoming, managed, NewJournalStore(filepath.Join(root, "journal.json")), 2)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := executor.PreviewBatch([]PlanEntry{
		{SourcePath: "missing.epub", TargetPath: "missing.epub"},
		{SourcePath: "large.epub", TargetPath: "large.epub"},
		{SourcePath: "../escape.epub", TargetPath: "../escape.epub"},
	})
	if err != nil || preview.Ready || preview.Conflicts != 3 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	if preview.Entries[0].Issues[0].Code != "source_unavailable" || preview.Entries[1].Issues[0].Code != "source_too_large" || preview.Entries[2].Issues[0].Code != "unsafe_path" {
		t.Fatalf("issues=%+v", preview.Entries)
	}
	if _, err = executor.PreviewBatch(make([]PlanEntry, MaximumBatchPreviewEntries+1)); err == nil {
		t.Fatal("oversized preview was accepted")
	}
}

// Plan is a client-facing endpoint of its own, so the batch cap has to hold there and not only in preview: a thousand
// entries queued as one transaction bypasses the hundred-per-call rhythm every screen follows.
func TestPlanRefusesMoreThanOneBatch(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	if err := os.MkdirAll(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	executor, err := NewExecutor(incoming, managed, NewJournalStore(filepath.Join(root, "journal.json")), 2)
	if err != nil {
		t.Fatal(err)
	}
	entries := make([]PlanEntry, MaximumBatchPreviewEntries+1)
	for index := range entries {
		name := fmt.Sprintf("book-%03d.epub", index)
		entries[index] = PlanEntry{SourcePath: name, TargetPath: name}
	}
	if _, err = executor.Plan(entries); err == nil {
		t.Fatalf("plan accepted %d entries", len(entries))
	}
}
