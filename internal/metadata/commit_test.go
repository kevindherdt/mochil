package metadata

import (
	"path/filepath"
	"testing"

	"mochil/internal/catalog"
)

func TestCommitReviewPersistsOverridesAndAppliedState(t *testing.T) {
	root := t.TempDir()
	jobs, err := OpenJobStore(filepath.Join(root, "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, err := jobs.Create("books", "Books", []catalog.Book{{ID: "one", Title: "Book", Authors: []string{"Unknown"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Decide(job.ID, "one", "missing_author", "accepted", "Writer"); err != nil {
		t.Fatal(err)
	}
	overrides, err := catalog.OpenOverrideStore(filepath.Join(root, "overrides.json"))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := CommitReview(jobs, overrides, CommitRequest{JobID: job.ID, LibraryID: "books", OverrideChanges: map[string]map[string]string{"one": {"authors": "Writer"}}})
	if err != nil || updated.Status != "applied" {
		t.Fatalf("commit = %#v, %v", updated, err)
	}
}

func TestCommitReviewRollsBackEveryLayerWhenJobCannotCommit(t *testing.T) {
	root := t.TempDir()
	jobs, err := OpenJobStore(filepath.Join(root, "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, err := jobs.Create("books", "Books", []catalog.Book{{ID: "one", Title: "Book", Authors: []string{"Unknown"}}})
	if err != nil {
		t.Fatal(err)
	}
	overridePath := filepath.Join(root, "overrides.json")
	overrides, err := catalog.OpenOverrideStore(overridePath)
	if err != nil {
		t.Fatal(err)
	}
	rolledBack := 0
	_, err = CommitReview(jobs, overrides, CommitRequest{
		JobID: job.ID, LibraryID: "books",
		OverrideChanges: map[string]map[string]string{"one": {"authors": "Writer"}},
		RollbackSource:  func() error { rolledBack++; return nil },
	})
	if err == nil || rolledBack != 1 {
		t.Fatalf("commit error = %v, source rollbacks = %d", err, rolledBack)
	}
	reopened, err := catalog.OpenOverrideStore(overridePath)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := reopened.Wrap("books", planBackend{books: []catalog.Book{{ID: "one", Authors: []string{"Unknown"}}}})
	book, _ := wrapped.Book("one")
	if len(book.Authors) != 1 || book.Authors[0] != "Unknown" {
		t.Fatalf("catalogue override was not rolled back: %#v", book)
	}
}
