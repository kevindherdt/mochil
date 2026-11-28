package metadata

import (
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/catalog"
)

func TestJobStoreRejectsTrailingState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reviews.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"jobs":[]} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenJobStore(path); err == nil {
		t.Fatal("review-job state with trailing JSON was accepted")
	}
}

func TestJobStorePersistsReviewAndProposal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reviews.json")
	store, err := OpenJobStore(path)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{{ID: "one", Path: "Saga/Book.epub", Title: "Book Volume 3", Authors: []string{"Writer"}, Series: "Saga"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(job.Items) != 1 || job.Items[0].Proposal == nil || job.Items[0].Proposal.Proposed != "3" {
		t.Fatalf("unexpected job: %#v", job)
	}
	reopened, err := OpenJobStore(path)
	if err != nil {
		t.Fatal(err)
	}
	jobs := reopened.List("books")
	if len(jobs) != 1 || jobs[0].ID != job.ID || jobs[0].Items != nil {
		t.Fatalf("unexpected jobs: %#v", jobs)
	}
}

func TestJobStorePersistsDecisions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reviews.json")
	store, err := OpenJobStore(path)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{{ID: "one", Title: "Book", Authors: []string{"Unknown"}}})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Decide(job.ID, "one", "missing_author", "accepted", "Writer")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "resolved" || updated.Items[0].DecisionValue != "Writer" {
		t.Fatalf("unexpected decision: %#v", updated)
	}
	reopened, err := OpenJobStore(path)
	if err != nil {
		t.Fatal(err)
	}
	saved, ok := reopened.Get(job.ID)
	if !ok || saved.Items[0].Decision != "accepted" {
		t.Fatalf("decision was not persisted: %#v", saved)
	}
	applied, err := reopened.MarkApplied(job.ID)
	if err != nil || applied.Status != "applied" || applied.AppliedAt == nil {
		t.Fatalf("job was not marked applied: %#v %v", applied, err)
	}
}

func TestNewInspectionInheritsLatestDecisionAndPendingClearsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reviews.json")
	store, err := OpenJobStore(path)
	if err != nil {
		t.Fatal(err)
	}
	books := []catalog.Book{{ID: "one", Title: "Book", Authors: []string{"Unknown"}}}
	first, err := store.Create("books", "Books", books)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Decide(first.ID, "one", "missing_author", "accepted", "Writer"); err != nil {
		t.Fatal(err)
	}
	second, err := store.Create("books", "Books", books)
	if err != nil {
		t.Fatal(err)
	}
	if second.InheritedDecisions != 1 || second.Status != "resolved" || second.Items[0].Decision != "accepted" || second.Items[0].DecisionValue != "Writer" {
		t.Fatalf("decision was not inherited: %#v", second)
	}
	if _, err := store.Decide(second.ID, "one", "missing_author", "pending", ""); err != nil {
		t.Fatal(err)
	}
	third, err := store.Create("books", "Books", books)
	if err != nil {
		t.Fatal(err)
	}
	if third.InheritedDecisions != 0 || third.Status != "review" || third.Items[0].Decision != "" {
		t.Fatalf("pending tombstone did not clear remembered decision: %#v", third)
	}
}

func TestDuplicateVolumeProposalUsesDifferentTitleMarker(t *testing.T) {
	one := 1.0
	jobPath := filepath.Join(t.TempDir(), "reviews.json")
	store, err := OpenJobStore(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{
		{ID: "one", Path: "Saga/Volume 1.epub", Title: "Saga, Vol. 1", Authors: []string{"Writer"}, Series: "Saga", SeriesIndex: &one},
		{ID: "ten", Path: "Saga/Volume 10.epub", Title: "Saga, Vol. 10", Authors: []string{"Writer"}, Series: "Saga", SeriesIndex: &one},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range job.Items {
		if item.BookID == "ten" {
			if item.Proposal == nil || item.Proposal.Current != "1" || item.Proposal.Proposed != "10" || item.Proposal.Confidence != "high" {
				t.Fatalf("unexpected duplicate correction: %#v", item.Proposal)
			}
			return
		}
	}
	t.Fatal("duplicate review item not found")
}

func TestPartMarkerRemainsMediumConfidence(t *testing.T) {
	store, err := OpenJobStore(filepath.Join(t.TempDir(), "reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{{ID: "one", Title: "Saga Part 3", Series: "Saga", Authors: []string{"Writer"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(job.Items) != 1 || job.Items[0].Proposal == nil || job.Items[0].Proposal.Confidence != "medium" {
		t.Fatalf("part marker should require individual review: %#v", job.Items)
	}
}

func TestDecideManyIsAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reviews.json")
	store, err := OpenJobStore(path)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{{ID: "one", Title: "Book", Authors: []string{"Unknown"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DecideMany(job.ID, []Decision{
		{BookID: "one", Code: "missing_author", Decision: "accepted", Value: "Writer"},
		{BookID: "missing", Code: "missing_author", Decision: "ignored"},
	})
	if err == nil {
		t.Fatal("expected missing review item error")
	}
	unchanged, ok := store.Get(job.ID)
	if !ok || unchanged.Items[0].Decision != "" {
		t.Fatalf("partial decision was retained: %#v", unchanged)
	}
}

func TestMissingAuthorProposalUsesConsistentSeriesSiblings(t *testing.T) {
	store, err := OpenJobStore(filepath.Join(t.TempDir(), "reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{
		{ID: "one", Title: "Saga 1", Series: "Saga", Authors: []string{"Writer", "Illustrator"}},
		{ID: "two", Title: "Saga 2", Series: " Saga ", Authors: []string{"Writer", "Illustrator"}},
		{ID: "three", Title: "Saga 3", Series: "SAGA", Authors: []string{"Unknown"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range job.Items {
		if item.Code != "missing_author" {
			continue
		}
		if item.Proposal == nil || item.Proposal.Field != "authors" || item.Proposal.Proposed != "Writer & Illustrator" || item.Proposal.Source != "consistent series siblings" {
			t.Fatalf("unexpected author proposal: %#v", item)
		}
		return
	}
	t.Fatal("missing-author review item not found")
}

func TestMissingAuthorProposalRejectsConflictingSeriesSiblings(t *testing.T) {
	store, err := OpenJobStore(filepath.Join(t.TempDir(), "reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{
		{ID: "one", Title: "Saga 1", Series: "Saga", Authors: []string{"First Writer"}},
		{ID: "two", Title: "Saga 2", Series: "Saga", Authors: []string{"Second Writer"}},
		{ID: "three", Title: "Saga 3", Series: "Saga", Authors: []string{"Unknown"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range job.Items {
		if item.Code == "missing_author" && item.Proposal != nil {
			t.Fatalf("conflicting authors produced a proposal: %#v", item)
		}
	}
}

func TestMissingAuthorProposalUsesStructuredFolderEvidence(t *testing.T) {
	store, err := OpenJobStore(filepath.Join(t.TempDir(), "reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{{ID: "one", Path: "Ursula K. Le Guin/Earthsea/Earthsea - Vol 01 - A Wizard of Earthsea.epub", Title: "A Wizard of Earthsea", Series: "Earthsea", Authors: []string{"Unknown"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range job.Items {
		if item.Code == "missing_author" && item.Proposal != nil && item.Proposal.Proposed == "Ursula K. Le Guin" && item.Proposal.Source == "author folder" && item.Proposal.Confidence == "medium" {
			return
		}
	}
	t.Fatalf("unexpected folder proposal: %#v", job.Items)
}

func TestMissingTitleProposalRemovesStructuredSeriesAndVolume(t *testing.T) {
	store, err := OpenJobStore(filepath.Join(t.TempDir(), "reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("books", "Books", []catalog.Book{{ID: "one", Path: "Brandon Sanderson/Mistborn/Mistborn - Vol 01 - The Final Empire.epub", Authors: []string{"Brandon Sanderson"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range job.Items {
		if item.Code == "missing_title" && item.Proposal != nil && item.Proposal.Proposed == "The Final Empire" && item.Proposal.Source == "structured filename" {
			return
		}
	}
	t.Fatalf("unexpected title proposal: %#v", job.Items)
}

// A decision is only carried over while the issue is still the same issue. The old rule keyed on book and code alone, so a
// book whose title changed inherited the old "accept" value and came back resolved with a stale answer already applied.
func TestChangedIssueDoesNotInheritAnOldDecision(t *testing.T) {
	store, err := OpenJobStore(filepath.Join(t.TempDir(), "reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Create("books", "Books", []catalog.Book{{ID: "one", Title: "Book", Authors: []string{"Unknown"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Decide(first.ID, "one", "missing_author", "accepted", "Writer"); err != nil {
		t.Fatal(err)
	}
	second, err := store.Create("books", "Books", []catalog.Book{{ID: "one", Title: "A Renamed Book", Authors: []string{"Unknown"}}})
	if err != nil {
		t.Fatal(err)
	}
	if second.InheritedDecisions != 0 {
		t.Fatalf("a changed issue inherited %d decisions: %#v", second.InheritedDecisions, second.Items)
	}
	if second.Items[0].Decision != "" || second.Status != "review" {
		t.Fatalf("the changed issue came back with a stale decision: %#v", second.Items[0])
	}
}
