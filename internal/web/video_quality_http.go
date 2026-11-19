package web

import (
	"net/http"
	"strings"

	"mochil/internal/video"
)

func (m *multiServer) apiVideoQuality(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(strings.TrimSpace(r.URL.Query().Get("library")))
	if !ok || library.Video == nil || !videoLibraryKind(library.Kind) {
		http.Error(w, "Video library not found", http.StatusNotFound)
		return
	}
	report := video.InspectQuality(library.Kind, library.Video.Items(), func(item video.Item) bool {
		if m.videoMetadata == nil {
			return false
		}
		_, exists := m.videoMetadata.Get(library.ID, item.ID)
		return exists
	}, func(show string) bool {
		if m.videoMetadata == nil {
			return false
		}
		_, exists := m.videoMetadata.GetShow(library.ID, show)
		return exists
	})
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, report)
}

func videoLibraryKind(kind string) bool {
	switch kind {
	case "movies", "tv", "tvshows", "anime", "video":
		return true
	default:
		return false
	}
}
