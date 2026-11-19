package web

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mochil/internal/audio"
)

// audioHLSSegment matches the fragmented-MP4 segment names the audio playlist lists
// (seg00012.m4s). The video player lists MPEG-TS names (seg00012.ts) and has its own
// pattern; using that one here made on-demand segment preparation unreachable for audio.
var audioHLSSegment = regexp.MustCompile(`^seg([0-9]{5})\.m4s$`)

func (m *multiServer) audioHLSHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || m.audioHLS == nil {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/audio-hls/"), "/"), "/")
	if len(parts) != 3 && len(parts) != 4 {
		http.NotFound(w, r)
		return
	}
	libraryID, e1 := url.PathUnescape(parts[0])
	itemID, e2 := url.PathUnescape(parts[1])
	quality, resource := audio.HLSQualityHigh, parts[2]
	if len(parts) == 4 {
		var valid bool
		quality, valid = audio.ParseHLSQuality(parts[2])
		if !valid {
			http.Error(w, "Unknown audio quality", http.StatusBadRequest)
			return
		}
		resource = parts[3]
	}
	library, ok := m.library(libraryID)
	if e1 != nil || e2 != nil || !ok || library.Audio == nil {
		http.NotFound(w, r)
		return
	}
	item, exists := library.Audio.Item(itemID)
	if !exists {
		http.NotFound(w, r)
		return
	}
	if resource == "index.m3u8" {
		file, opened, err := library.Audio.Open(itemID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		key, err := m.audioHLS.Start(file, opened, quality)
		if err != nil {
			http.Error(w, "This track cannot be prepared", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		prefix := "/api/audio-hls/"
		if r.Header.Get("X-Mochil-Android-Audio") == "1" {
			prefix = "/api/android/v1/audio-hls/"
		}
		base := prefix + url.PathEscape(libraryID) + "/" + url.PathEscape(itemID) + "/" + string(quality) + "/"
		_, _ = w.Write(audio.RewriteHLSPlaylist(m.audioHLS.Playlist(opened), base, key))
		return
	}
	key := m.audioHLS.Key(item, quality)
	if matches := audioHLSSegment.FindStringSubmatch(resource); len(matches) == 2 {
		segment, _ := strconv.Atoi(matches[1])
		file, opened, err := library.Audio.Open(itemID)
		if err != nil || m.audioHLS.EnsureSegment(file, opened, quality, segment) != nil {
			http.Error(w, "This part of the track could not be prepared", http.StatusServiceUnavailable)
			return
		}
	}
	file, mediaType, err := m.audioHLS.Open(key, resource)
	if err != nil {
		http.Error(w, "Compatibility stream is not ready", http.StatusServiceUnavailable)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeContent(w, r, resource, info.ModTime(), file)
}

func (m *multiServer) audioArtworkHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.SplitN(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/audio-artwork/"), "/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, e1 := url.PathUnescape(parts[0])
	itemID, e2 := url.PathUnescape(parts[1])
	library, ok := m.library(libraryID)
	if e1 != nil || e2 != nil || !ok || library.Audio == nil {
		http.NotFound(w, r)
		return
	}
	item, ok := library.Audio.Item(itemID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if m.audioArtwork == nil {
		http.NotFound(w, r)
		return
	}
	variant := audio.ArtworkCard
	switch r.URL.Query().Get("size") {
	case "", string(audio.ArtworkCard):
	case string(audio.ArtworkPlayer):
		variant = audio.ArtworkPlayer
	default:
		http.Error(w, "Unknown artwork size", http.StatusBadRequest)
		return
	}
	var generated string
	var err error
	if sidecar, _, found := library.Audio.OpenSidecar(itemID); found {
		info, statErr := sidecar.Stat()
		if statErr != nil {
			sidecar.Close()
			http.NotFound(w, r)
			return
		}
		sourceRevision := strconv.FormatInt(info.Size(), 10) + ":" + strconv.FormatInt(info.ModTime().UnixNano(), 10)
		generated, err = m.audioArtwork.CoverVariant(sidecar, item, variant, sourceRevision)
	} else {
		source, _, openErr := library.Audio.Open(itemID)
		if openErr != nil {
			http.NotFound(w, r)
			return
		}
		generated, err = m.audioArtwork.CoverVariant(source, item, variant, item.Revision)
		if err != nil {
			// Some albums embed artwork in only one track. Reuse a nearby track's
			// image instead of making every other row look unillustrated.
			for _, sibling := range library.Audio.ArtworkSiblings(itemID, 8) {
				siblingSource, _, openErr := library.Audio.Open(sibling.ID)
				if openErr != nil {
					continue
				}
				generated, err = m.audioArtwork.CoverVariant(siblingSource, sibling, variant, sibling.Revision)
				if err == nil {
					break
				}
			}
		}
	}
	if err != nil {
		// A track without art stays without art until the file changes, and the address carries its revision (v=),
		// so the browser may remember the answer instead of asking again on every redraw.
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "private, max-age=3600")
		}
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(generated)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	serveVideoArtwork(w, r, file, filepath.Ext(generated), info.ModTime())
}

func (m *multiServer) streamAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.SplitN(strings.Trim(strings.TrimPrefix(r.URL.Path, "/audio/"), "/"), "/", 3)
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, e1 := url.PathUnescape(parts[0])
	itemID, e2 := url.PathUnescape(parts[1])
	library, ok := m.library(libraryID)
	if e1 != nil || e2 != nil || !ok || library.Audio == nil {
		http.NotFound(w, r)
		return
	}
	file, item, err := library.Audio.Open(itemID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", item.MediaType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(path.Base(item.Path)))
	}
	http.ServeContent(w, r, path.Base(item.Path), item.Modified, file)
}
