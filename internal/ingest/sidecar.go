package ingest

import (
	"errors"
	"path/filepath"
	"strings"

	"mochil/internal/opf"
)

// WriteSidecar writes the Calibre-style sidecar for a book that is already inside the managed library. Only the managed
// root is ever written to: the book must be there as a real file (not a link out of the root), the path may not point
// into the journal's area, and an existing sidecar is left alone unless overwrite is asked for.
func (e *Executor) WriteSidecar(targetPath string, meta opf.Metadata, overwrite bool) (string, error) {
	if !safeJournalPath(targetPath) || strings.HasPrefix(targetPath, ".ingest/") {
		return "", errors.New("the path is not a managed library path")
	}
	if kind, ok := kindForExtension(filepath.Ext(targetPath)); !ok || kind != Publication {
		return "", errors.New("sidecars are written for books only")
	}
	resolved, err := filepath.EvalSymlinks(e.target(targetPath))
	if err != nil {
		return "", errors.New("the book is not in the managed library")
	}
	if !withinRoot(e.targetRoot, resolved) {
		return "", errors.New("the book resolves outside the managed library")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	written, err := opf.WriteBeside(resolved, meta, overwrite)
	if err != nil {
		return "", err
	}
	relative, relErr := filepath.Rel(e.targetRoot, written)
	if relErr != nil {
		return "", relErr
	}
	return filepath.ToSlash(relative), nil
}
