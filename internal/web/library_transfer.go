package web

import (
	"fmt"
	"net/http"

	"mochil/internal/catalog"
	"mochil/internal/transfer"
)

const transferMaximumBooks = 500

type transferRequest struct {
	Library string   `json:"library"`
	To      string   `json:"to"`
	IDs     []string `json:"ids"`
	Mode    string   `json:"mode"` // "move" or "copy"
	Preview bool     `json:"preview"`
}

type transferRow struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	Reason string `json:"reason,omitempty"`
}

// editorTransfer moves or copies books from one folder library to another. The book keeps its path, so its place in
// the other library is predictable; edits, custom columns and (for a move) everyone's reading place go with it. With
// preview it only says what would happen. Otherwise it records the transfer, applies it step by step, and answers a
// token to undo it; a step that fails leaves the record open and the answer is 409, never a quiet success.
func (m *multiServer) editorTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input transferRequest
	if err := decodeJSONRequest(w, r, editorMaximumBody, &input, true); err != nil {
		http.Error(w, "The request could not be read.", http.StatusBadRequest)
		return
	}
	if len(input.IDs) == 0 || len(input.IDs) > transferMaximumBooks || (input.Mode != "move" && input.Mode != "copy") {
		http.Error(w, fmt.Sprintf("Choose between 1 and %d books, and move or copy.", transferMaximumBooks), http.StatusBadRequest)
		return
	}
	from, ok := m.editorLibrary(w, input.Library)
	if !ok {
		return
	}
	to, found := m.library(input.To)
	if !found || input.To == input.Library || !importable(from) || !importable(to) {
		http.Error(w, "Books can only go to a different library that is an ordinary folder of books.", http.StatusBadRequest)
		return
	}
	mover, err := transfer.New(from.SourcePath, to.SourcePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	byID := map[string]catalog.Book{}
	for _, book := range from.Store.Books() {
		byID[book.ID] = book
	}
	entries := make([]transfer.Entry, 0, len(input.IDs))
	rows := make([]transferRow, 0, len(input.IDs))
	ids := make(map[string]string, len(input.IDs))
	paths := make([]string, 0, len(input.IDs))
	for _, id := range input.IDs {
		book, exists := byID[id]
		if !exists {
			http.Error(w, "A chosen book is no longer in the library.", http.StatusNotFound)
			return
		}
		entries = append(entries, transfer.Entry{Source: book.Path, Target: book.Path})
		rows = append(rows, transferRow{ID: id, Title: book.Title, Path: book.Path})
		ids[id] = catalog.BookID(book.Path)
		paths = append(paths, book.Path)
	}
	blocked := 0
	for i, problem := range mover.Check(entries) {
		rows[i].Reason = problem.Reason
		if problem.Reason != "" {
			blocked++
		}
	}
	if input.Preview || blocked > 0 {
		writeJSON(w, http.StatusOK, map[string]any{"books": rows, "ready": len(rows) - blocked, "blocked": blocked, "to": to.Name})
		return
	}
	m.metadataApplyMu.Lock()
	defer m.metadataApplyMu.Unlock()
	move := input.Mode == "move"
	// Record the intent before the first write. An entry that stays open names
	// the steps still missing, and reconcile finishes them later; without it a
	// failure would leave files in one library and their bookkeeping in another.
	journal, err := m.transfers()
	if err != nil {
		http.Error(w, "Nothing was changed: the transfer could not be recorded. "+err.Error(), http.StatusInternalServerError)
		return
	}
	entry, err := journal.begin(transferJournalEntry{
		From:    from.ID,
		To:      to.ID,
		Mode:    input.Mode,
		Books:   ids,
		Paths:   paths,
		Skipped: transferSkippedSteps(input.Mode),
	})
	if err != nil {
		http.Error(w, "Nothing was changed: the transfer could not be recorded. "+err.Error(), http.StatusInternalServerError)
		return
	}
	undos := make([]func() error, 0, len(transferStepsInOrder))
	for _, step := range m.transferSteps(from, to, ids, paths, move) {
		undo, applyErr := step.apply()
		if applyErr == nil {
			applyErr = journal.markDone(entry.ID, step.name)
		}
		if applyErr != nil {
			// The entry stays open on purpose: reconcile replays the steps that
			// are not recorded as done, and every step is written to be replayed.
			_ = journal.fail(entry.ID, step.name, applyErr)
			writeTransferIncomplete(w, step.name, applyErr)
			return
		}
		undos = append(undos, undo)
	}
	if err := journal.finish(entry.ID); err != nil {
		// Every step is in place, but the journal still lists the transfer, so
		// no undo token is handed out. Reconcile will replay it (a no-op now)
		// and clear the entry.
		writeTransferIncomplete(w, "journal", err)
		return
	}
	token := m.editorUndo.remember(reverseTransferUndos(undos))
	writeJSON(w, http.StatusOK, map[string]any{"done": len(entries), "mode": input.Mode, "to": to.Name, "token": token, "rescan": []string{from.ID, to.ID}})
}

// writeTransferIncomplete answers a transfer whose steps did not all finish.
// The status is 409, not 200: the caller must know that the move is recorded
// rather than complete, and that the server will finish it by itself.
func writeTransferIncomplete(w http.ResponseWriter, step string, err error) {
	writeJSON(w, http.StatusConflict, map[string]any{
		"failed":   step,
		"error":    err.Error(),
		"recorded": true,
		"message":  "The transfer is recorded and will be completed automatically.",
	})
}

// transferTargets lists the libraries books from this one can be moved or copied to: other ordinary folders of books.
func (m *multiServer) transferTargets(from Library) []importDestination {
	out := []importDestination{}
	if !importable(from) {
		return out
	}
	for _, lib := range m.libraries {
		if lib.ID != from.ID && importable(lib) {
			out = append(out, importDestination{ID: lib.ID, Name: lib.Name})
		}
	}
	return out
}
