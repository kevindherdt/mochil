package web

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"mochil/internal/catalog"
	"mochil/internal/metadata"
)

func (m *multiServer) apiMetadataIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	libraryID := r.URL.Query().Get("library")
	library, ok := m.library(libraryID)
	if !ok || library.Store == nil {
		http.Error(w, "Unknown library", http.StatusNotFound)
		return
	}
	if metadataHydrating(library.Store) {
		w.Header().Set("Retry-After", "10")
		http.Error(w, "Embedded book metadata is still loading from this shelf. Try the inspection again shortly.", http.StatusConflict)
		return
	}
	report := metadata.Inspect(library.Store.Books())
	limit := 100
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 && value <= 500 {
		limit = value
	}
	offset := 0
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	end := min(offset+limit, len(report.Issues))
	if offset > len(report.Issues) {
		offset = len(report.Issues)
		end = offset
	}
	report.Issues = report.Issues[offset:end]
	writeJSON(w, http.StatusOK, report)
}

func (m *multiServer) apiMetadataJobs(w http.ResponseWriter, r *http.Request) {
	if m.metadataJobs == nil {
		http.Error(w, "Metadata review storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	libraryID := r.URL.Query().Get("library")
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, m.metadataJobs.List(libraryID))
	case http.MethodPost:
		library, ok := m.library(libraryID)
		if !ok || library.Store == nil {
			http.Error(w, "Unknown library", http.StatusNotFound)
			return
		}
		if metadataHydrating(library.Store) {
			w.Header().Set("Retry-After", "10")
			http.Error(w, "Embedded book metadata is still loading from this shelf. Try the inspection again shortly.", http.StatusConflict)
			return
		}
		job, err := m.metadataJobs.Create(library.ID, library.Name, library.Store.Books())
		if err != nil {
			http.Error(w, "Could not save metadata review: "+err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, job)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func metadataHydrating(store catalog.Backend) bool {
	hydrator, ok := catalog.MetadataHydrationCapability(store)
	return ok && hydrator.MetadataHydrating()
}

func (m *multiServer) apiMetadataJob(w http.ResponseWriter, r *http.Request) {
	if m.metadataJobs == nil {
		http.Error(w, "Metadata review storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	remainder := strings.TrimPrefix(r.URL.Path, "/api/metadata/jobs/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 2 && parts[1] == "apply" && r.Method == http.MethodPost {
		m.apiMetadataApply(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "preview" && r.Method == http.MethodGet {
		m.apiMetadataApplyPreview(w, parts[0])
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		job, ok := m.metadataJobs.Get(parts[0])
		if !ok {
			http.Error(w, "Review job not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}
	if len(parts) == 2 && parts[1] == "items" && r.Method == http.MethodPut {
		var input struct {
			Decisions []metadata.Decision `json:"decisions"`
		}
		if err := decodeJSONRequest(w, r, 256<<10, &input, true); err != nil {
			http.Error(w, "Invalid review decisions", http.StatusBadRequest)
			return
		}
		job, err := m.metadataJobs.DecideMany(parts[0], input.Decisions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}
	if len(parts) == 4 && parts[1] == "items" && r.Method == http.MethodPut {
		bookID, err := url.PathUnescape(parts[2])
		if err != nil {
			http.Error(w, "Invalid book ID", http.StatusBadRequest)
			return
		}
		code, err := url.PathUnescape(parts[3])
		if err != nil {
			http.Error(w, "Invalid issue code", http.StatusBadRequest)
			return
		}
		var input struct {
			Decision string `json:"decision"`
			Value    string `json:"value"`
		}
		if err := decodeJSONRequest(w, r, 4096, &input, true); err != nil {
			http.Error(w, "Invalid decision", http.StatusBadRequest)
			return
		}
		job, err := m.metadataJobs.Decide(parts[0], bookID, code, input.Decision, input.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}
	http.NotFound(w, r)
}

func (m *multiServer) apiMetadataApplyPreview(w http.ResponseWriter, jobID string) {
	job, ok := m.metadataJobs.Get(jobID)
	if !ok {
		http.Error(w, "Review job not found", http.StatusNotFound)
		return
	}
	library, ok := m.library(job.LibraryID)
	if !ok || library.Store == nil {
		http.Error(w, "The reviewed library no longer exists", http.StatusConflict)
		return
	}
	preview := metadata.BuildApplyPreview(job, library.Store, metadata.ApplyCapabilities{SourceType: library.SourceType, WritesEnabled: library.MetadataWrites, CalibreAvailable: m.calibre != nil, OverridesAvailable: m.metadataOverrides != nil})
	writeJSON(w, http.StatusOK, preview)
}

func (m *multiServer) apiMetadataApply(w http.ResponseWriter, r *http.Request, jobID string) {
	if !m.metadataApplyMu.TryLock() {
		http.Error(w, "Another metadata apply is already running", http.StatusConflict)
		return
	}
	defer m.metadataApplyMu.Unlock()
	var input struct {
		Confirmation string `json:"confirmation"`
	}
	if err := decodeJSONRequest(w, r, 4096, &input, true); err != nil || input.Confirmation != "APPLY "+jobID {
		http.Error(w, "Type the exact confirmation shown before applying", http.StatusBadRequest)
		return
	}
	job, ok := m.metadataJobs.Get(jobID)
	if !ok {
		http.Error(w, "Review job not found", http.StatusNotFound)
		return
	}
	if job.Status != "resolved" {
		http.Error(w, "Review job must be fully resolved", http.StatusConflict)
		return
	}
	library, ok := m.library(job.LibraryID)
	if !ok || library.Store == nil {
		http.Error(w, "The reviewed library no longer exists", http.StatusConflict)
		return
	}
	if library.SourceType == "calibre" && library.MetadataWrites && m.calibre != nil {
		m.applyCalibreMetadata(w, job, library)
		return
	}
	if library.SourceType != "filesystem" || !library.MetadataWrites {
		m.applyCatalogueOverrides(w, job, library)
		return
	}
	store, ok := underlyingCatalogStore(library.Store)
	if !ok {
		http.Error(w, "This filesystem backend cannot apply metadata", http.StatusConflict)
		return
	}
	changeSet, err := metadata.BuildChangeSet(job, library.Store)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	plan := metadata.BuildFilesystemApplyPlan(job.ID, m.cfg.MetadataBackupDirectory, changeSet, store.AbsolutePath)
	if len(plan.OverrideChanges) != 0 && m.metadataOverrides == nil {
		http.Error(w, "Catalogue override storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	rollback, err := metadata.ApplyEPUBBatch(plan.Targets)
	if err != nil {
		status := http.StatusConflict
		if metadata.TransactionNeedsAttention(err) {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}
	updated, err := metadata.CommitReview(m.metadataJobs, m.metadataOverrides, metadata.CommitRequest{JobID: job.ID, LibraryID: library.ID, OverrideChanges: plan.OverrideChanges, RollbackSource: rollback})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	effects := m.finishMetadataSourceApply(library, plan.SourceBooks)
	response := newMetadataApplyResponse(updated, "filesystem", len(plan.Targets), len(plan.OverrideChanges), effects)
	response.Backups = filepath.Join(m.cfg.MetadataBackupDirectory, job.ID)
	writeJSON(w, http.StatusOK, response)
}

// metadataApplyResponse makes file impact an authoritative server contract.
// The browser must not have to infer whether applying catalogue metadata
// rewrote a publication archive from a transient preview kept in memory.
type metadataApplyResponse struct {
	Job                        metadata.Job `json:"job"`
	ApplyMode                  string       `json:"apply_mode"`
	AppliedBooks               int          `json:"applied_books"`
	SourceBooks                int          `json:"source_books"`
	OverrideBooks              int          `json:"override_books"`
	PublicationArchivesChanged bool         `json:"publication_archives_changed"`
	ComicInfoChanged           bool         `json:"comic_info_changed"`
	Backups                    string       `json:"backups,omitempty"`
	ScanBooks                  int          `json:"scan_books,omitempty"`
	ScanWarning                string       `json:"scan_warning,omitempty"`
}

func newMetadataApplyResponse(job metadata.Job, mode string, sourceBooks, overrideBooks int, effects metadataApplyEffects) metadataApplyResponse {
	return metadataApplyResponse{
		Job: job, ApplyMode: mode,
		AppliedBooks: sourceBooks + overrideBooks,
		SourceBooks:  sourceBooks, OverrideBooks: overrideBooks,
		PublicationArchivesChanged: mode == "filesystem" && sourceBooks > 0,
		ComicInfoChanged:           false,
		ScanBooks:                  effects.Scanned, ScanWarning: effects.Warning,
	}
}

func underlyingCatalogStore(backend catalog.Backend) (*catalog.Store, bool) {
	for {
		if store, ok := backend.(*catalog.Store); ok {
			return store, true
		}
		wrapper, ok := backend.(interface{ Underlying() catalog.Backend })
		if !ok {
			return nil, false
		}
		backend = wrapper.Underlying()
	}
}

func acceptedOverrideChanges(job metadata.Job, library Library) (map[string]map[string]string, error) {
	changeSet, err := metadata.BuildChangeSet(job, library.Store)
	if err != nil {
		return nil, err
	}
	changes := make(map[string]map[string]string, len(changeSet))
	for _, change := range changeSet {
		changes[change.Book.ID] = change.Fields
	}
	return changes, nil
}

func (m *multiServer) applyCatalogueOverrides(w http.ResponseWriter, job metadata.Job, library Library) {
	if m.metadataOverrides == nil {
		http.Error(w, "Catalogue override storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	changes, err := acceptedOverrideChanges(job, library)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	updated, err := metadata.CommitReview(m.metadataJobs, m.metadataOverrides, metadata.CommitRequest{JobID: job.ID, LibraryID: library.ID, OverrideChanges: changes})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, newMetadataApplyResponse(updated, "catalogue-overrides", 0, len(changes), metadataApplyEffects{}))
}

func (m *multiServer) applyCalibreMetadata(w http.ResponseWriter, job metadata.Job, library Library) {
	if m.calibre == nil {
		http.Error(w, m.calibreError+"; install Calibre in the runtime or configure MOCHIL_CALIBREDB", http.StatusServiceUnavailable)
		return
	}
	if strings.TrimSpace(library.SourcePath) == "" {
		http.Error(w, "The Calibre library path is unavailable", http.StatusConflict)
		return
	}
	changeSet, err := metadata.BuildChangeSet(job, library.Store)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	targets, targetBooks := metadata.BuildCalibreTargets(job.ID, library.SourcePath, m.cfg.MetadataBackupDirectory, changeSet)
	// Once a confirmed transaction starts it must finish or roll back even if
	// the browser disconnects midway through it.
	rollback, err := metadata.ApplyCalibreBatch(context.Background(), m.calibre, library.SourcePath, targets)
	if err != nil {
		status := http.StatusConflict
		if metadata.TransactionNeedsAttention(err) {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}
	updated, err := metadata.CommitReview(m.metadataJobs, m.metadataOverrides, metadata.CommitRequest{JobID: job.ID, LibraryID: library.ID, RollbackSource: rollback})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	effects := m.finishMetadataSourceApply(library, targetBooks)
	response := newMetadataApplyResponse(updated, "calibre-database", len(targets), 0, effects)
	response.Backups = filepath.Join(m.cfg.MetadataBackupDirectory, job.ID)
	writeJSON(w, http.StatusOK, response)
}
