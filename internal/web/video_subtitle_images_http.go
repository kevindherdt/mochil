package web

// Image-based (PGS) subtitle serving. Split out of video_http.go, which is already at its line budget, and because
// this is a different job: that file plans an encode, this one hands out pre-rendered cue images.

import (
	"net/http"
	"os"
	"time"

	"mochil/internal/video"
)

// serveSubtitleImage answers the two requests an image-based subtitle track needs: its manifest, and one cue image.
//
// The manifest is produced by a background ffmpeg pass (a two-hour film takes about a minute), so the first request
// answers 202 and the player retries — that keeps every request short instead of pinning a connection for a minute,
// and it means two people opening the same film share one extraction rather than racing each other.
func (m *multiServer) serveSubtitleImage(w http.ResponseWriter, r *http.Request, openItem func(string) (*os.File, video.Item, error), item video.Item, streamIndex int, name string) {
	if m.subtitleImages == nil {
		http.Error(w, "Image subtitles are unavailable", http.StatusServiceUnavailable)
		return
	}
	known := false
	for _, subtitle := range item.Subtitles {
		if subtitle.Index == streamIndex {
			known = true
			break
		}
	}
	if !known {
		http.NotFound(w, r)
		return
	}

	if name != "manifest.json" {
		file, err := m.subtitleImages.Open(item, streamIndex, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		// A cue never changes once written (the directory is keyed by revision and swapped in whole), so it caches.
		w.Header().Set("Cache-Control", "private, max-age=86400, immutable")
		w.Header().Set("Content-Type", "image/png")
		http.ServeContent(w, r, name, time.Time{}, file)
		return
	}

	itemID := item.ID
	images, ready, err := m.subtitleImages.Manifest(r.Context(), item, nil, streamIndex, func() (*os.File, error) {
		file, _, openErr := openItem(itemID)
		return file, openErr
	})
	if err != nil {
		http.Error(w, "This subtitle track cannot be read", http.StatusUnprocessableEntity)
		return
	}
	if !ready {
		w.Header().Set("Retry-After", "5")
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "preparing"})
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	writeJSON(w, http.StatusOK, images)
}
