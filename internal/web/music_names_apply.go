package web

import (
	"net/http"
	"strings"
	"time"

	"mochil/internal/audio"
	"mochil/internal/musicrename"
)

type musicNamesApplyRequest struct {
	Library string `json:"library"`
	GroupID string `json:"group_id"`
	Names   []struct {
		ID string `json:"id"`
		To string `json:"to"`
	} `json:"names"`
}

// carryTrackHistory moves the queue, favourites, playlists and progress to the renamed tracks' new ids. It reports
// the first failure; the files stay renamed either way, and Undo reverses ids and files together.
func (m *multiServer) carryTrackHistory(library string, ids audio.Remap) error {
	var first error
	keep := func(err error) {
		if err != nil && first == nil {
			first = err
		}
	}
	if m.audioQueue != nil {
		keep(m.audioQueue.Remap(library, ids))
	}
	if m.audioFavorites != nil {
		keep(m.audioFavorites.Remap(library, ids))
	}
	if m.audioPlaylists != nil {
		keep(m.audioPlaylists.Remap(library, ids))
	}
	if m.audioProgress != nil {
		keep(m.audioProgress.Remap(library, ids))
	}
	return first
}

func ledgerRemap(l musicrename.Ledger) audio.Remap {
	ids := audio.Remap{}
	for _, move := range l.Moves {
		if move.ID != "" && move.NewID != "" && move.Done {
			ids[move.ID] = move.NewID
		}
	}
	return ids
}

func (m *multiServer) rescanAfterRename(library string) {
	if m.scanLibrary != nil {
		_, _ = m.scanLibrary(library)
	}
}

func (m *multiServer) musicNamesApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input musicNamesApplyRequest
	if err := decodeJSONRequest(w, r, 1<<20, &input, true); err != nil || len(input.Names) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	lib, items, ok := m.musicAlbum(w, input.Library, input.GroupID)
	if !ok {
		return
	}
	byID := map[string]audio.Item{}
	for _, item := range items {
		byID[item.ID] = item
	}
	moves := make([]musicrename.Move, 0, len(input.Names))
	seen := map[string]bool{}
	for _, name := range input.Names {
		item, found := byID[name.ID]
		if !found || seen[name.ID] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "A track in the request is not part of this album."})
			return
		}
		seen[name.ID] = true
		renamed, valid := audio.Identify(lib.ID, name.To, item.Size, time.Now())
		if !valid {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "That is not a name Music can use."})
			return
		}
		moves = append(moves, musicrename.Move{ID: item.ID, NewID: renamed.ID, From: item.Path, To: name.To})
	}
	ledger, err := m.musicEngine(lib).Apply(lib.ID, moves)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	response := map[string]any{"ledger": ledger.ID, "renamed": len(moves)}
	if err := m.carryTrackHistory(lib.ID, ledgerRemap(ledger)); err != nil {
		response["warning"] = "The files were renamed, but queue or playlist entries could not all be carried over. Use Undo to put everything back."
	}
	m.rescanAfterRename(lib.ID)
	writeJSON(w, http.StatusOK, response)
}

func (m *multiServer) musicNamesUndo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Library string `json:"library"`
		Ledger  string `json:"ledger"`
	}
	if err := decodeJSONRequest(w, r, 4096, &input, true); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	lib, ok := m.library(strings.TrimSpace(input.Library))
	if !ok || lib.Kind != "music" || lib.SourcePath == "" {
		http.NotFound(w, r)
		return
	}
	engine := m.musicEngine(lib)
	before, err := engine.Load(input.Ledger)
	if err != nil || before.Library != lib.ID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Unknown rename record."})
		return
	}
	done := ledgerRemap(before) // what was carried over, as it stood before the files are put back
	after, undoErr := engine.Undo(input.Ledger)
	restored := audio.Remap{}
	for _, move := range after.Moves {
		if id := done[move.ID]; move.ID != "" && !move.Done && id != "" {
			restored[id] = move.ID
		}
	}
	if err := m.carryTrackHistory(lib.ID, restored); err != nil && undoErr == nil {
		undoErr = err
	}
	m.rescanAfterRename(lib.ID)
	if undoErr != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": undoErr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"undone": true})
}

func (m *multiServer) musicNamesLast(w http.ResponseWriter, r *http.Request) {
	lib, ok := m.library(strings.TrimSpace(r.URL.Query().Get("library")))
	if !ok || lib.Kind != "music" || lib.SourcePath == "" {
		http.NotFound(w, r)
		return
	}
	last, found := m.musicEngine(lib).Latest(lib.ID)
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{"ledger": ""})
		return
	}
	files := 0
	for _, move := range last.Moves {
		if move.ID != "" {
			files++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ledger": last.ID, "created": last.Created, "files": files, "state": last.State})
}
