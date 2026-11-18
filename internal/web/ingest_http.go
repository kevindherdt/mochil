package web

import (
	"net/http"
	"strconv"

	"mochil/internal/catalog"
	"mochil/internal/ingest"
)

type cachedIngestPreview struct {
	version uint64
	set     ingest.PublicationPreview
}

// apiIngestPreview exposes a read-only, bounded view of deterministic path
// proposals. Applying or moving files is intentionally a separate contract.
func (m *multiServer) apiIngestPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Store == nil {
		http.Error(w, "Publication library not found", http.StatusNotFound)
		return
	}
	if library.SourceType != "filesystem" && library.SourceType != "calibre" {
		http.Error(w, "Ingest preview is available only for local publication libraries", http.StatusConflict)
		return
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	set := m.publicationIngestPreview(library)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, set.Page(offset, limit))
}

func (m *multiServer) publicationIngestPreview(library Library) ingest.PublicationPreview {
	versioned, cacheable := library.Store.(catalog.VersionedBackend)
	version := uint64(0)
	if cacheable {
		version = versioned.CatalogVersion()
		m.ingestPreviewMu.Lock()
		cached, ok := m.ingestPreviews[library.ID]
		m.ingestPreviewMu.Unlock()
		if ok && cached.version == version {
			return cached.set
		}
	}
	books := library.Store.Books()
	current := make([]ingest.CurrentPublication, 0, len(books))
	for _, book := range books {
		current = append(current, ingest.CurrentPublication{ID: book.ID, Path: book.Path, Title: book.Title, Authors: book.Authors, Series: book.Series, SeriesIndex: book.SeriesIndex, Identifier: book.Identifier})
	}
	set := ingest.BuildPublicationPreview(current)
	if cacheable && versioned.CatalogVersion() == version {
		m.ingestPreviewMu.Lock()
		if m.ingestPreviews == nil {
			m.ingestPreviews = make(map[string]cachedIngestPreview)
		}
		m.ingestPreviews[library.ID] = cachedIngestPreview{version: version, set: set}
		m.ingestPreviewMu.Unlock()
	}
	return set
}
