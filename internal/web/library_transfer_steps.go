package web

import (
	"fmt"
	"os"
	"path/filepath"

	"mochil/internal/transfer"
)

// transferStep is one part of a library transfer. apply returns the action that
// takes the step back, and the journal records the step only after apply
// succeeded, so an interrupted transfer always names its missing work.
type transferStep struct {
	name  string
	apply func() (func() error, error)
}

func noTransferStep() func() error { return func() error { return nil } }

// reverseTransferUndos takes every completed step back, newest first. It is the
// undo token of a fully applied transfer.
func reverseTransferUndos(undos []func() error) func() error {
	return func() error {
		var first error
		for i := len(undos) - 1; i >= 0; i-- {
			if err := undos[i](); err != nil && first == nil {
				first = err
			}
		}
		return first
	}
}

// transferSkippedSteps names the steps reconcile must never replay for this
// mode. A copy leaves the source books in place, so the source still holds the
// values the transfer carried; replaying that carry after the copies were edited
// would silently overwrite the edits with the old values. Recording the skip in
// the journal is the difference between finishing a transfer and corrupting one.
func transferSkippedSteps(mode string) map[string]string {
	if mode != "copy" {
		return nil
	}
	reason := "a copy keeps its source values, so replaying this step could overwrite later edits on the copies"
	return map[string]string{transferStepMetadata: reason, transferStepColumns: reason}
}

// transferSteps builds the ordered work of a transfer. The order matters: the
// files must exist in the target library before its metadata, columns and
// reading places are addressed there.
func (m *multiServer) transferSteps(from, to Library, ids map[string]string, paths []string, move bool) []transferStep {
	return []transferStep{
		{name: transferStepFiles, apply: func() (func() error, error) {
			return applyTransferFiles(from, to, paths, move)
		}},
		{name: transferStepMetadata, apply: func() (func() error, error) {
			return m.applyTransferMetadata(from.ID, to.ID, ids, move)
		}},
		{name: transferStepColumns, apply: func() (func() error, error) {
			return m.applyTransferColumns(from.ID, to.ID, ids, move)
		}},
		{name: transferStepReading, apply: func() (func() error, error) {
			return m.applyTransferReading(from.ID, to.ID, ids, move)
		}},
		{name: transferStepNotes, apply: func() (func() error, error) {
			return m.applyTransferNotes(from.ID, to.ID, ids, move)
		}},
	}
}

// applyTransferFiles moves or copies the books, resuming from what an earlier
// attempt already did. It is the only step that touches files and it refuses to
// guess: a book already in the target library counts as done, a move whose hard
// link survived is finished by dropping the leftover link, and anything else
// unclear is an error rather than a second move.
func applyTransferFiles(from, to Library, paths []string, move bool) (func() error, error) {
	mover, err := transfer.New(from.SourcePath, to.SourcePath)
	if err != nil {
		return nil, err
	}
	pending := make([]transfer.Entry, 0, len(paths))
	for _, relative := range paths {
		source := filepath.Join(from.SourcePath, filepath.FromSlash(relative))
		target := filepath.Join(to.SourcePath, filepath.FromSlash(relative))
		sourceInfo, sourceErr := os.Lstat(source)
		targetInfo, targetErr := os.Lstat(target)
		switch {
		case targetErr == nil && sourceErr != nil:
			continue // already transferred
		case targetErr == nil && sourceErr == nil && move && (os.SameFile(sourceInfo, targetInfo) || transfer.Same(source, target)):
			// The move placed the new copy (a hard link, or a cross-device copy) and
			// stopped before unlinking the source: the only thing left is the unlink.
			if err := os.Remove(source); err != nil {
				return nil, fmt.Errorf("%s: %w", relative, err)
			}
			continue
		case targetErr == nil && sourceErr == nil:
			if move {
				return nil, fmt.Errorf("%s: both libraries already have a file with this name", relative)
			}
			continue
		case sourceErr != nil:
			return nil, fmt.Errorf("%s: the book is no longer there", relative)
		}
		pending = append(pending, transfer.Entry{Source: relative, Target: relative})
	}
	done := []transfer.Item{}
	if len(pending) > 0 {
		items, runErr := mover.Run(pending, move)
		done = append(done, items...)
		if runErr != nil {
			return nil, runErr
		}
	}
	return func() error { return mover.Undo(done, move) }, nil
}

// applyTransferMetadata carries the editor's corrections. A move deletes each
// correction from the source, so replaying it only carries what is still there;
// once the step has run there is nothing left and the replay is a no-op.
func (m *multiServer) applyTransferMetadata(from, to string, ids map[string]string, move bool) (func() error, error) {
	if m.metadataOverrides == nil {
		return noTransferStep(), nil
	}
	if !move {
		return m.metadataOverrides.Transfer(from, to, ids, move)
	}
	remaining := make(map[string]string, len(ids))
	for oldID, newID := range ids {
		if _, ok := m.metadataOverrides.Get(from, oldID); ok {
			remaining[oldID] = newID
		}
	}
	if len(remaining) == 0 {
		return noTransferStep(), nil
	}
	return m.metadataOverrides.Transfer(from, to, remaining, move)
}

// applyTransferColumns carries added-column values. A move clears the source
// cells, so a replay finds nothing to carry and does nothing; a copy would find
// the same values again, which is why copy entries record this step as skipped.
func (m *multiServer) applyTransferColumns(from, to string, ids map[string]string, move bool) (func() error, error) {
	if m.customColumns == nil {
		return noTransferStep(), nil
	}
	carried, cleared := map[string]map[string]string{}, map[string]map[string]string{}
	for oldID, values := range m.customColumns.Library(from) {
		newID, moving := ids[oldID]
		if !moving {
			continue
		}
		carried[newID] = values
		if !move {
			continue
		}
		cleared[oldID] = map[string]string{}
		for column := range values {
			cleared[oldID][column] = ""
		}
	}
	if len(carried) == 0 {
		return noTransferStep(), nil
	}
	undo, err := m.customColumns.Set(to, carried)
	if err != nil {
		return nil, err
	}
	undos := []func() error{undo}
	if move {
		undo, err := m.customColumns.Set(from, cleared)
		if err != nil {
			// The target values are written; the entry stays open and reconcile
			// clears the source cells on its next run.
			return nil, err
		}
		undos = append(undos, undo)
	}
	return reverseTransferUndos(undos), nil
}

// applyTransferReading carries everyone's place in the books. A move deletes the
// old keys, so a replay finds nothing to carry: the step is safe to repeat.
func (m *multiServer) applyTransferReading(from, to string, ids map[string]string, move bool) (func() error, error) {
	if !move || m.reading == nil {
		return noTransferStep(), nil
	}
	return m.reading.Transfer(from, to, ids)
}

// applyTransferNotes carries everyone's highlights, notes and bookmarks in the
// books. They are keyed by library and book id like a reading place, so without
// this step a move left them under a library that no longer holds the file:
// invisible where the book went, and inherited by any later file that took the
// old id. A copy is a new book and starts clean, as it does for reading.
func (m *multiServer) applyTransferNotes(from, to string, ids map[string]string, move bool) (func() error, error) {
	if !move || m.annotations == nil {
		return noTransferStep(), nil
	}
	return m.annotations.Transfer(from, to, ids)
}
