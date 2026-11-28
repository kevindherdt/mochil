package web

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"mochil/internal/ingest"
	"mochil/internal/jobs"
	"mochil/internal/opf"
)

func (m *multiServer) apiManagedIngest(w http.ResponseWriter, r *http.Request) {
	if m.managedIngest == nil {
		if r.Method == http.MethodGet && r.URL.Path == "/api/metadata/managed-ingest" {
			writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "transactions": []ingest.Transaction{}})
			return
		}
		http.Error(w, "Managed ingest is not configured", http.StatusConflict)
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/metadata/managed-ingest"), "/")
	if suffix == "" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		transactions, err := m.managedIngest.Transactions()
		if err != nil {
			http.Error(w, "Managed ingest journal is unavailable", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "transactions": transactions})
		return
	}
	if suffix == "plan" {
		m.planManagedIngest(w, r)
		return
	}
	if suffix == "preview" {
		m.previewManagedIngest(w, r)
		return
	}
	if suffix == "sidecars" {
		m.writeManagedSidecars(w, r)
		return
	}
	if suffix == "embed" {
		m.embedManagedMetadata(w, r)
		return
	}
	if suffix == "discover" {
		m.discoverManagedIngest(w, r)
		return
	}
	parts := strings.Split(suffix, "/")
	if len(parts) == 1 && r.Method == http.MethodGet {
		ex := m.executorOfTransaction(parts[0])
		if ex == nil {
			http.NotFound(w, r)
			return
		}
		transaction, err := ex.Transaction(parts[0])
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Managed ingest journal is unavailable", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, transaction)
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost && (parts[1] == "execute" || parts[1] == "recover") {
		m.runManagedIngest(w, r, parts[0], parts[1] == "recover")
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost && parts[1] == "discard" {
		ex := m.executorOfTransaction(parts[0])
		if ex == nil {
			http.NotFound(w, r)
			return
		}
		transaction, err := ex.Discard(parts[0])
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Only a plan that has not started can be discarded", http.StatusConflict)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, transaction)
		return
	}
	http.NotFound(w, r)
}

func (m *multiServer) previewManagedIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Entries []ingest.PlanEntry `json:"entries"`
	}
	if decodeJSONRequest(w, r, 128<<10, &input, true) != nil {
		http.Error(w, "Invalid managed ingest preview", http.StatusBadRequest)
		return
	}
	ex, ok := m.executorFromRequest(w, r)
	if !ok {
		return
	}
	preview, err := ex.PreviewBatch(input.Entries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, preview)
}

func (m *multiServer) discoverManagedIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid discovery query", http.StatusBadRequest)
		return
	}
	limit := ingest.MaximumDiscoveryItems
	if raw := r.Form.Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > ingest.MaximumDiscoveryItems {
			http.Error(w, "Discovery limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = value
	}
	if !ingest.ValidNamingPreset(r.Form.Get("naming")) {
		http.Error(w, "Naming must be default, calibre or kavita", http.StatusBadRequest)
		return
	}
	page, err := m.managedIngest.DiscoverNamed(r.Context(), ingest.Kind(r.Form.Get("kind")), r.Form.Get("cursor"), limit, r.Form.Get("naming"))
	if err != nil {
		http.Error(w, "Could not inspect the incoming folder", http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, page)
}

func (m *multiServer) planManagedIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Entries []ingest.PlanEntry `json:"entries"`
	}
	if decodeJSONRequest(w, r, 128<<10, &input, true) != nil {
		http.Error(w, "Invalid managed ingest plan", http.StatusBadRequest)
		return
	}
	ex, ok := m.executorFromRequest(w, r)
	if !ok {
		return
	}
	transaction, err := ex.Plan(input.Entries)
	if err != nil {
		// The executor's errors name absolute paths inside the drop folder ("lstat /root/.../incoming/x.epub: no such file").
		// A fixed answer here, the detail in the log: the plan's per-book reasons are what preview is for.
		slog.Warn("managed ingest plan refused", "error", err)
		http.Error(w, "The plan was refused; open the preview for per-book reasons.", http.StatusConflict)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, transaction)
}

func (m *multiServer) runManagedIngest(w http.ResponseWriter, r *http.Request, id string, recover bool) {
	var input struct {
		Confirmation string `json:"confirmation"`
	}
	if decodeJSONRequest(w, r, 4096, &input, true) != nil || input.Confirmation != id {
		http.Error(w, "Type the exact transaction ID to confirm managed ingest", http.StatusBadRequest)
		return
	}
	ex := m.executorOfTransaction(id)
	if ex == nil {
		http.NotFound(w, r)
		return
	}
	if _, err := ex.Transaction(id); errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "Managed ingest journal is unavailable", http.StatusServiceUnavailable)
		return
	}
	action := "execute"
	if recover {
		action = "recover"
	}
	key := "managed-ingest:" + action + ":" + id
	if m.backgroundJobs == nil {
		http.Error(w, "Managed ingest queue is unavailable", http.StatusServiceUnavailable)
		return
	}
	err := m.backgroundJobs.Submit(key, func(ctx context.Context) error {
		if recover {
			_, _, recoverErr := ex.RecoverContext(ctx, id)
			if errors.Is(recoverErr, ingest.ErrManualReview) {
				return errors.New("managed ingest requires manual review")
			}
			if recoverErr != nil {
				return errors.New("managed ingest recovery failed; review the transaction")
			}
			return nil
		}
		_, executeErr := ex.ExecuteContext(ctx, id)
		if executeErr != nil {
			return errors.New("managed ingest failed; review the transaction")
		}
		return nil
	})
	if errors.Is(err, jobs.ErrDuplicate) {
		http.Error(w, "This managed ingest action is already running", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "Managed ingest queue is unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusAccepted, map[string]string{"job": key, "transaction_id": id})
}

type sidecarFields struct {
	Title       string            `json:"title"`
	TitleSort   string            `json:"title_sort"`
	Authors     []string          `json:"authors"`
	AuthorSort  string            `json:"author_sort"`
	Series      string            `json:"series"`
	SeriesIndex *float64          `json:"series_index"`
	Tags        []string          `json:"tags"`
	Publisher   string            `json:"publisher"`
	Published   string            `json:"published"`
	Language    string            `json:"language"`
	Rating      float64           `json:"rating"` // stars, 0 to 5
	Description string            `json:"description"`
	Identifiers map[string]string `json:"identifiers"`
}

// writeManagedSidecars writes the metadata.opf of books already in the managed library, one result per book so a single
// failure does not hide the others. It never replaces a sidecar unless asked to.
func (m *multiServer) writeManagedSidecars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Entries []struct {
			TargetPath string        `json:"target_path"`
			Metadata   sidecarFields `json:"metadata"`
		} `json:"entries"`
		Overwrite bool `json:"overwrite"`
	}
	if decodeJSONRequest(w, r, 1<<20, &input, true) != nil || len(input.Entries) == 0 || len(input.Entries) > ingest.MaximumBatchPreviewEntries {
		http.Error(w, "Send between 1 and 100 books", http.StatusBadRequest)
		return
	}
	type result struct {
		TargetPath string `json:"target_path"`
		Sidecar    string `json:"sidecar,omitempty"`
		Status     string `json:"status"` // written, exists, failed
		Message    string `json:"message,omitempty"`
	}
	ex, ok := m.executorFromRequest(w, r)
	if !ok {
		return
	}
	results := make([]result, 0, len(input.Entries))
	for _, entry := range input.Entries {
		meta := entry.Metadata.metadata()
		written, err := ex.WriteSidecar(entry.TargetPath, meta, input.Overwrite)
		switch {
		case err == nil:
			results = append(results, result{TargetPath: entry.TargetPath, Sidecar: written, Status: "written"})
		case errors.Is(err, opf.ErrExists):
			results = append(results, result{TargetPath: entry.TargetPath, Status: "exists", Message: "A sidecar is already there and was left alone."})
		default:
			results = append(results, result{TargetPath: entry.TargetPath, Status: "failed", Message: err.Error()})
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (f sidecarFields) metadata() opf.Metadata {
	return opf.Metadata{Title: f.Title, TitleSort: f.TitleSort, Authors: f.Authors, AuthorSort: f.AuthorSort, Series: f.Series, SeriesIndex: f.SeriesIndex,
		Tags: f.Tags, Publisher: f.Publisher, Published: f.Published, Language: f.Language, Rating: int(math.Round(f.Rating * 2)), Description: f.Description, Identifiers: f.Identifiers}
}

// embedManagedMetadata writes fields into the EPUBs Mochil copied into the managed library. It never touches the
// originals in the drop folder, and each book is checked and replaced on its own, so one failure leaves the others done.
func (m *multiServer) embedManagedMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Entries []struct {
			TargetPath string        `json:"target_path"`
			Metadata   sidecarFields `json:"metadata"`
		} `json:"entries"`
	}
	if decodeJSONRequest(w, r, 1<<20, &input, true) != nil || len(input.Entries) == 0 || len(input.Entries) > ingest.MaximumBatchPreviewEntries {
		http.Error(w, "Send between 1 and 100 books", http.StatusBadRequest)
		return
	}
	type result struct {
		TargetPath string `json:"target_path"`
		Status     string `json:"status"` // embedded, failed
		Message    string `json:"message,omitempty"`
	}
	ex, ok := m.executorFromRequest(w, r)
	if !ok {
		return
	}
	results := make([]result, 0, len(input.Entries))
	for _, entry := range input.Entries {
		if err := ex.EmbedMetadata(entry.TargetPath, entry.Metadata.metadata()); err != nil {
			results = append(results, result{TargetPath: entry.TargetPath, Status: "failed", Message: err.Error()})
			continue
		}
		results = append(results, result{TargetPath: entry.TargetPath, Status: "embedded"})
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
