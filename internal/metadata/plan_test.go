package metadata

import (
	"context"
	"path/filepath"
	"testing"

	"mochil/internal/catalog"
)

type planBackend struct{ books []catalog.Book }

func (b planBackend) Scan() (catalog.Stats, error) { return catalog.Stats{Books: len(b.books)}, nil }
func (b planBackend) Books() []catalog.Book        { return append([]catalog.Book(nil), b.books...) }
func (b planBackend) Book(id string) (catalog.Book, bool) {
	for _, book := range b.books {
		if book.ID == id {
			return book, true
		}
	}
	return catalog.Book{}, false
}

func TestBuildApplyPreviewSelectsCapabilitiesOutsideHTTP(t *testing.T) {
	backend := planBackend{books: []catalog.Book{
		{ID: "epub", Title: "EPUB", Path: "EPUB.epub", Format: "epub"},
		{ID: "pdf", Title: "PDF", Path: "PDF.pdf", Format: "pdf"},
	}}
	job := Job{ID: "job", LibraryID: "books", Status: "resolved", Items: []ReviewItem{
		{Issue: Issue{BookID: "epub", Code: "missing_author"}, Decision: "accepted", DecisionValue: "Writer"},
		{Issue: Issue{BookID: "pdf", Code: "missing_title"}, Decision: "accepted", DecisionValue: "Title"},
		{Issue: Issue{BookID: "missing", Code: "missing_author"}, Decision: "accepted", DecisionValue: "Nobody"},
	}}
	filesystem := BuildApplyPreview(job, backend, ApplyCapabilities{SourceType: "filesystem", WritesEnabled: true, OverridesAvailable: true})
	if filesystem.Mode != "mixed" || !filesystem.Resolved || filesystem.Unsupported != 1 || len(filesystem.Changes) != 2 || filesystem.Changes[0].Mode != "source file" || filesystem.Changes[1].Mode != "catalogue override" {
		t.Fatalf("filesystem preview = %#v", filesystem)
	}
	readonly := BuildApplyPreview(job, backend, ApplyCapabilities{SourceType: "readonly", OverridesAvailable: true})
	if readonly.Mode != "catalogue-overrides" || readonly.WritesEnabled || len(readonly.Changes) != 2 || readonly.Changes[0].Mode != "catalogue override" {
		t.Fatalf("read-only preview = %#v", readonly)
	}
	calibre := BuildApplyPreview(job, backend, ApplyCapabilities{SourceType: "calibre", WritesEnabled: true, CalibreAvailable: true})
	if calibre.Mode != "calibre-database" || len(calibre.Changes) != 2 || calibre.Changes[1].Mode != "Calibre database" {
		t.Fatalf("Calibre preview = %#v", calibre)
	}
	withoutOverrides := BuildApplyPreview(job, backend, ApplyCapabilities{SourceType: "filesystem", WritesEnabled: true})
	if len(withoutOverrides.Changes) != 1 || withoutOverrides.Unsupported != 2 || withoutOverrides.Changes[0].BookID != "epub" {
		t.Fatalf("preview without overrides = %#v", withoutOverrides)
	}
}

func TestApplyTargetBuildersPartitionAndNameBackupsDeterministically(t *testing.T) {
	changes := []BookChanges{
		{Book: catalog.Book{ID: "epub", Title: "Book", Path: "Series/Book.epub", Format: "EPUB"}, Fields: map[string]string{"authors": "Writer"}},
		{Book: catalog.Book{ID: "pdf", Title: "PDF", Path: "Series/PDF.pdf", Format: "pdf"}, Fields: map[string]string{"title": "PDF title"}},
	}
	filesystem := BuildFilesystemApplyPlan("job", "/backups", changes, func(book catalog.Book) string { return filepath.Join("/library", filepath.FromSlash(book.Path)) })
	if len(filesystem.Targets) != 1 || len(filesystem.OverrideChanges) != 1 || filesystem.Targets[0].Path != filepath.Join("/library", "Series", "Book.epub") || filepath.Ext(filesystem.Targets[0].Backup) != ".epub" || filesystem.SourceBooks["epub"].Title != "Book" {
		t.Fatalf("filesystem plan = %#v", filesystem)
	}
	second := BuildFilesystemApplyPlan("job", "/backups", changes, func(book catalog.Book) string { return book.Path })
	if filepath.Base(filesystem.Targets[0].Backup) != filepath.Base(second.Targets[0].Backup) {
		t.Fatalf("backup names differ: %q != %q", filesystem.Targets[0].Backup, second.Targets[0].Backup)
	}
	calibre, books := BuildCalibreTargets("job", "/calibre", "/backups", changes)
	if len(calibre) != 2 || calibre[0].BookPath != filepath.Join("/calibre", "Series", "Book.epub") || filepath.Ext(calibre[0].Backup) != ".opf" || books["pdf"].Title != "PDF" {
		t.Fatalf("Calibre targets = %#v, books = %#v", calibre, books)
	}
}
func (b planBackend) Acquire(context.Context, catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{}, nil
}

func TestBuildChangeSetGroupsAcceptedFieldsAndIgnoresClassifications(t *testing.T) {
	backend := planBackend{books: []catalog.Book{{ID: "one", Title: "Old"}}}
	job := Job{Items: []ReviewItem{
		{Issue: Issue{BookID: "one", Code: "missing_title"}, Decision: "accepted", DecisionValue: "New"},
		{Issue: Issue{BookID: "one", Code: "missing_author"}, Decision: "accepted", DecisionValue: "Writer"},
		{Issue: Issue{BookID: "one", Code: "duplicate_edition"}, Decision: "ignored"},
	}}
	changes, err := BuildChangeSet(job, backend)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Fields["title"] != "New" || changes[0].Fields["authors"] != "Writer" {
		t.Fatalf("changes = %#v", changes)
	}
}

func TestBuildChangeSetRejectsConflictingAndUnsupportedDecisions(t *testing.T) {
	backend := planBackend{books: []catalog.Book{{ID: "one"}}}
	conflicting := Job{Items: []ReviewItem{
		{Issue: Issue{BookID: "one", Code: "missing_author"}, Decision: "accepted", DecisionValue: "First"},
		{Issue: Issue{BookID: "one", Code: "missing_author"}, Decision: "accepted", DecisionValue: "Second"},
	}}
	if _, err := BuildChangeSet(conflicting, backend); err == nil {
		t.Fatal("expected conflicting fields to fail")
	}
	unsupported := Job{Items: []ReviewItem{{Issue: Issue{BookID: "one", Code: "duplicate_edition"}, Decision: "accepted", DecisionValue: "yes"}}}
	if _, err := BuildChangeSet(unsupported, backend); err == nil {
		t.Fatal("expected unsupported accepted decision to fail")
	}
}
