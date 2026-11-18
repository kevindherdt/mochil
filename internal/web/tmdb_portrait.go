package web

import (
	"context"
	"net/http"
	"time"

	"mochil/internal/enrichment"
)

// tmdbPortrait only accepts a single TMDB filename; it never proxies a caller URL.
func (m *multiServer) tmdbPortrait(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	values := r.URL.Query()["path"]
	if len(values) != 1 || !enrichment.ValidTMDBProfilePath(values[0]) {
		http.Error(w, "Invalid portrait path", http.StatusBadRequest)
		return
	}
	if m.metadataArtwork == nil {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	path, mime, err := m.metadataArtwork.Fetch(ctx, "https://image.tmdb.org/t/p/w342"+values[0])
	if err != nil {
		http.Error(w, "Artwork is temporarily unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", mime)
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "private, max-age=86400")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}
