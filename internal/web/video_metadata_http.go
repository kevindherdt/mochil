package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mochil/internal/video"
)

func (m *multiServer) apiVideoMetadata(w http.ResponseWriter, r *http.Request) {
	if m.videoMetadata == nil {
		http.Error(w, "Video metadata storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-metadata/"), "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, libraryErr := url.PathUnescape(parts[0])
	itemID, itemErr := url.PathUnescape(parts[1])
	library, ok := m.library(libraryID)
	if libraryErr != nil || itemErr != nil || !ok || library.Video == nil || library.MetadataProvider != "tmdb" {
		http.NotFound(w, r)
		return
	}
	if _, exists := library.Video.Item(itemID); !exists {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		value, exists := m.videoMetadata.Get(libraryID, itemID)
		writeJSON(w, http.StatusOK, map[string]any{"configured": exists, "metadata": value})
	case http.MethodPut:
		var value video.MetadataOverride
		if err := decodeJSONRequest(w, r, 16<<10, &value, true); err != nil {
			http.Error(w, "Invalid video metadata selection", http.StatusBadRequest)
			return
		}
		if err := m.videoMetadata.Set(libraryID, itemID, value); err != nil {
			http.Error(w, "Could not save video metadata: "+err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"metadata": value})
	case http.MethodDelete:
		if err := m.videoMetadata.Delete(libraryID, itemID); err != nil {
			http.Error(w, "Could not remove video metadata", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"removed": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) apiVideoShowMetadata(w http.ResponseWriter, r *http.Request) {
	if m.videoMetadata == nil {
		http.Error(w, "Video metadata storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	libraryID, err := url.PathUnescape(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-show-metadata/"), "/"))
	show := strings.TrimSpace(r.URL.Query().Get("show"))
	library, ok := m.library(libraryID)
	if err != nil || !ok || library.Video == nil || (library.Kind != "tv" && library.Kind != "anime") || library.MetadataProvider != "tmdb" || show == "" {
		http.NotFound(w, r)
		return
	}
	found := false
	for _, item := range library.Video.Items() {
		if item.Show == show {
			found = true
			break
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		value, exists := m.videoMetadata.GetShow(libraryID, show)
		writeJSON(w, http.StatusOK, map[string]any{"configured": exists, "metadata": value})
	case http.MethodPut:
		var value video.MetadataOverride
		if err := decodeJSONRequest(w, r, 16<<10, &value, true); err != nil {
			http.Error(w, "Invalid show metadata selection", http.StatusBadRequest)
			return
		}
		if err := m.videoMetadata.SetShow(libraryID, show, value); err != nil {
			http.Error(w, "Could not save show metadata: "+err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"metadata": value})
	case http.MethodDelete:
		if err := m.videoMetadata.DeleteShow(libraryID, show); err != nil {
			http.Error(w, "Could not remove show metadata", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"removed": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) videoMetadataArtworkHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-metadata-artwork/"), "/"), "/")
	if len(parts) != 2 || m.videoMetadata == nil || m.metadataArtwork == nil {
		http.NotFound(w, r)
		return
	}
	libraryID, libraryErr := url.PathUnescape(parts[0])
	itemID, itemErr := url.PathUnescape(parts[1])
	library, ok := m.library(libraryID)
	if libraryErr != nil || itemErr != nil || !ok || library.Video == nil {
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("still") == "1" {
		m.videoEpisodeStill(w, r, library, itemID)
		return
	}
	value, exists := m.videoMetadata.Get(libraryID, itemID)
	source := metadataImageSource(value, r)
	if !exists || source == "" {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	path, contentType, err := m.metadataArtwork.Fetch(ctx, source)
	if err != nil {
		http.Error(w, "Artwork is temporarily unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, path)
}

func (m *multiServer) videoShowMetadataArtworkHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	libraryID, err := url.PathUnescape(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-show-metadata-artwork/"), "/"))
	show := strings.TrimSpace(r.URL.Query().Get("show"))
	library, ok := m.library(libraryID)
	if err != nil || !ok || library.Video == nil || m.videoMetadata == nil || m.metadataArtwork == nil {
		http.NotFound(w, r)
		return
	}
	value, exists := m.videoMetadata.GetShow(libraryID, show)
	source := metadataImageSource(value, r)
	if !exists || source == "" {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	path, contentType, fetchErr := m.metadataArtwork.Fetch(ctx, source)
	if fetchErr != nil {
		http.Error(w, "Artwork is temporarily unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, path)
}

// metadataImageSource is the saved poster, or with ?backdrop=1 the saved wide backdrop.
func metadataImageSource(value video.MetadataOverride, r *http.Request) string {
	if r.URL.Query().Get("backdrop") == "1" {
		return value.BackdropURL
	}
	return value.ArtworkURL
}
