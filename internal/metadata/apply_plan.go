package metadata

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"mochil/internal/catalog"
)

type ApplyCapabilities struct {
	SourceType         string
	WritesEnabled      bool
	CalibreAvailable   bool
	OverridesAvailable bool
}

type ApplyChange struct {
	BookID string `json:"book_id"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	Field  string `json:"field"`
	Value  string `json:"value"`
	Format string `json:"format"`
	Mode   string `json:"mode"`
}

type ApplyPreview struct {
	JobID         string        `json:"job_id"`
	LibraryID     string        `json:"library_id"`
	SourceType    string        `json:"source_type"`
	Mode          string        `json:"mode"`
	Message       string        `json:"message"`
	WritesEnabled bool          `json:"writes_enabled"`
	Resolved      bool          `json:"resolved"`
	Changes       []ApplyChange `json:"changes"`
	Unsupported   int           `json:"unsupported"`
}

func BuildApplyPreview(job Job, backend catalog.Backend, capabilities ApplyCapabilities) ApplyPreview {
	preview := ApplyPreview{
		JobID: job.ID, LibraryID: job.LibraryID, SourceType: capabilities.SourceType,
		Mode: "catalogue-overrides", Message: "Corrections will be stored in Mochil and shown in the web library and OPDS. Original files remain unchanged.",
		WritesEnabled: capabilities.WritesEnabled, Resolved: job.Status == "resolved",
		Changes: make([]ApplyChange, 0),
	}
	if capabilities.SourceType == "filesystem" && capabilities.WritesEnabled {
		preview.Mode = "mixed"
		preview.Message = "EPUB corrections update validated source files with backups. Unsupported formats use reversible Mochil catalogue overrides."
	} else if capabilities.SourceType == "calibre" && capabilities.WritesEnabled && capabilities.CalibreAvailable {
		preview.Mode = "calibre-database"
		preview.Message = "Eligible changes can be applied through Calibre after one final confirmation. Mochil stores the original OPF metadata and verifies every database update."
	}
	for _, item := range job.Items {
		if item.Decision != "accepted" {
			continue
		}
		book, found := backend.Book(item.BookID)
		field := FieldForIssue(item.Code)
		if !found || field == "" {
			preview.Unsupported++
			continue
		}
		mode := "catalogue override"
		switch {
		case capabilities.SourceType == "filesystem" && capabilities.WritesEnabled && strings.EqualFold(book.Format, "epub"):
			mode = "source file"
		case capabilities.SourceType == "calibre" && capabilities.WritesEnabled && capabilities.CalibreAvailable:
			mode = "Calibre database"
		case !capabilities.OverridesAvailable:
			preview.Unsupported++
			continue
		}
		preview.Changes = append(preview.Changes, ApplyChange{BookID: book.ID, Title: book.Title, Path: book.Path, Field: field, Value: item.DecisionValue, Format: book.Format, Mode: mode})
	}
	return preview
}

type FilesystemApplyPlan struct {
	Targets         []EPUBTarget
	SourceBooks     map[string]catalog.Book
	OverrideChanges map[string]map[string]string
}

func BuildFilesystemApplyPlan(jobID, backupDirectory string, changes []BookChanges, absolutePath func(catalog.Book) string) FilesystemApplyPlan {
	plan := FilesystemApplyPlan{Targets: make([]EPUBTarget, 0, len(changes)), SourceBooks: make(map[string]catalog.Book), OverrideChanges: make(map[string]map[string]string)}
	for _, change := range changes {
		book := change.Book
		if !strings.EqualFold(book.Format, "epub") {
			plan.OverrideChanges[book.ID] = change.Fields
			continue
		}
		sum := sha256.Sum256([]byte(jobID + "\x00" + book.ID))
		plan.Targets = append(plan.Targets, EPUBTarget{BookID: book.ID, Title: book.Title, Path: absolutePath(book), Backup: filepath.Join(backupDirectory, jobID, fmt.Sprintf("%x.epub", sum[:12])), Changes: change.Fields})
		plan.SourceBooks[book.ID] = book
	}
	return plan
}

func BuildCalibreTargets(jobID, libraryPath, backupDirectory string, changes []BookChanges) ([]CalibreTarget, map[string]catalog.Book) {
	targets := make([]CalibreTarget, 0, len(changes))
	books := make(map[string]catalog.Book, len(changes))
	for _, change := range changes {
		book := change.Book
		sum := sha256.Sum256([]byte(jobID + "\x00" + book.ID))
		targets = append(targets, CalibreTarget{BookPath: filepath.Join(libraryPath, filepath.FromSlash(book.Path)), Backup: filepath.Join(backupDirectory, jobID, fmt.Sprintf("%x.opf", sum[:12])), Changes: change.Fields})
		books[book.ID] = book
	}
	return targets, books
}
