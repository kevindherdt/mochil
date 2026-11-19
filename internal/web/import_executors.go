package web

import (
	"net/http"
	"os"
	"path/filepath"

	"mochil/internal/ingest"
)

// importable reports whether books can be copied straight into a library: an ordinary folder library of books that
// Mochil may write to. Calibre libraries are read-only here, and a library that overlaps the drop folder is refused
// when its executor is made.
func importable(lib Library) bool {
	if lib.SourceType != "filesystem" || lib.SourcePath == "" || !publicationKind(lib.Kind) {
		return false
	}
	info, err := os.Stat(lib.SourcePath)
	return err == nil && info.IsDir()
}

// executorFor returns the executor that copies into a destination library; "" is the managed folder itself.
func (m *multiServer) executorFor(destination string) (*ingest.Executor, bool) {
	if m.managedIngest == nil {
		return nil, false
	}
	if destination == "" {
		return m.managedIngest, true
	}
	lib, ok := m.library(destination)
	if !ok || !importable(lib) {
		return nil, false
	}
	root := filepath.Clean(lib.SourcePath)
	m.ingestMu.Lock()
	defer m.ingestMu.Unlock()
	if cached, ok := m.ingestTargets[root]; ok {
		return cached, true
	}
	executor, err := m.managedIngest.ForTarget(root)
	if err != nil {
		return nil, false
	}
	if m.ingestTargets == nil {
		m.ingestTargets = map[string]*ingest.Executor{}
	}
	m.ingestTargets[root] = executor
	return executor, true
}

// importDestinations lists the libraries books can be imported into.
func (m *multiServer) importDestinations() []importDestination {
	out := []importDestination{}
	if m.managedIngest == nil {
		return out
	}
	for _, lib := range m.libraries {
		if _, ok := m.executorFor(lib.ID); ok {
			out = append(out, importDestination{ID: lib.ID, Name: lib.Name})
		}
	}
	return out
}

// executorOfTransaction finds the executor whose journal holds a transaction, whichever destination it was for.
func (m *multiServer) executorOfTransaction(id string) *ingest.Executor {
	if m.managedIngest == nil {
		return nil
	}
	if _, err := m.managedIngest.Transaction(id); err == nil {
		return m.managedIngest
	}
	for _, lib := range m.libraries {
		if executor, ok := m.executorFor(lib.ID); ok {
			if _, err := executor.Transaction(id); err == nil {
				return executor
			}
		}
	}
	return nil
}

// executorFromRequest picks the executor named by ?destination=<library id>, or the managed folder's own.
func (m *multiServer) executorFromRequest(w http.ResponseWriter, r *http.Request) (*ingest.Executor, bool) {
	executor, ok := m.executorFor(r.URL.Query().Get("destination"))
	if !ok {
		http.Error(w, "Books cannot be imported into that library.", http.StatusNotFound)
	}
	return executor, ok
}
