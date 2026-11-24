package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"mochil/internal/catalog"
	"mochil/internal/jobs"
)

// An editorJob is a long action on a library's books that runs in the background: counting, checking quality, looking
// for ISBNs. One handler gives every one of them the same behaviour: POST starts it on some books (or all of them),
// GET says how far it has got, DELETE stops it at once. What a stopped job already did stays done.
type editorJob struct {
	key       func(library string) string
	available func(*multiServer) bool
	// run does the work. It should look at ctx between books and return when it is canceled.
	run func(ctx context.Context, m *multiServer, lib Library, books []catalog.Book, opts editorJobOptions) (string, error)
	// unavailable and busy are the sentences a person sees when it cannot start.
	unavailable, busy string
}

// editorJobOptions are what the person asked for when starting the job. `recount` re-runs work that was already done;
// `recheck` means "look again at books that already have what the job looks for" — the ISBN search uses it to read books
// that already carry an ISBN, which is how a wrong or duplicated ISBN becomes visible.
type editorJobOptions struct {
	Recount bool
	Recheck bool
}

func (m *multiServer) serveEditorJob(w http.ResponseWriter, r *http.Request, job editorJob) {
	if !job.available(m) || m.backgroundJobs == nil {
		http.Error(w, job.unavailable, http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		state, found := m.backgroundJobs.State(job.key(r.URL.Query().Get("library")))
		if !found {
			writeJSON(w, http.StatusOK, map[string]any{"status": "idle"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": state.Status, "current": state.ProgressCurrent, "total": state.ProgressTotal, "message": state.ProgressMessage, "error": state.Error})
	case http.MethodDelete:
		lib, ok := m.editorLibrary(w, r.URL.Query().Get("library"))
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stopped": m.backgroundJobs.Cancel(job.key(lib.ID))})
	case http.MethodPost:
		m.startEditorJob(w, r, job)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) startEditorJob(w http.ResponseWriter, r *http.Request, job editorJob) {
	var req struct {
		Library string   `json:"library"`
		IDs     []string `json:"ids"`
		Recount bool     `json:"recount"`
		Recheck bool     `json:"recheck"`
	}
	if err := decodeJSONRequest(w, r, 1<<20, &req, true); err != nil {
		http.Error(w, "The request could not be read.", http.StatusBadRequest)
		return
	}
	lib, ok := m.editorLibrary(w, req.Library)
	if !ok {
		return
	}
	books := lib.Store.Books()
	if len(req.IDs) > 0 {
		wanted := map[string]bool{}
		for _, id := range req.IDs {
			wanted[id] = true
		}
		kept := books[:0:0]
		for _, book := range books {
			if wanted[book.ID] {
				kept = append(kept, book)
			}
		}
		books = kept
	}
	err := m.backgroundJobs.Submit(job.key(lib.ID), func(ctx context.Context) error {
		message, err := job.run(ctx, m, lib, books, editorJobOptions{Recount: req.Recount, Recheck: req.Recheck})
		if err != nil {
			return err
		}
		total := int64(len(books))
		jobs.Report(ctx, total, total, message)
		return nil
	})
	switch {
	case errors.Is(err, jobs.ErrDuplicate):
		http.Error(w, job.busy, http.StatusConflict)
	case err != nil:
		http.Error(w, job.unavailable, http.StatusServiceUnavailable)
	default:
		writeJSON(w, http.StatusAccepted, map[string]any{"books": len(books)})
	}
}

// reportAsking tells the page which book Hermes is on, so a slow answer reads as work in progress, not a stuck page.
func reportAsking(ctx context.Context, index, total, asked int, book catalog.Book) {
	title := []rune(strings.TrimSpace(book.Title))
	if len(title) > 40 {
		title = append(title[:39], '…')
	}
	jobs.Report(ctx, int64(index), int64(total), fmt.Sprintf("Asking Hermes about “%s” (question %d)", string(title), asked))
}
