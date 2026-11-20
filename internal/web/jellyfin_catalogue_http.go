package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mochil/internal/audio"
	"mochil/internal/jellyfin"
	"mochil/internal/video"
)

func (m *multiServer) jellyfinLibraries() []map[string]any {
	out := []map[string]any{}
	for _, library := range m.libraries {
		if library.Video != nil || library.Audio != nil {
			out = append(out, jellyfin.Library(library.ID, library.Name, library.Kind, m.jellyfinServerID()))
		}
	}
	return out
}
func (m *multiServer) jellyfinLibraryByID(id string) (Library, bool) {
	for _, library := range m.libraries {
		if jellyfin.ID("library", library.ID) == id {
			return library, true
		}
	}
	return Library{}, false
}
func (m *multiServer) jellyfinVideoByID(id string) (Library, video.Item, bool) {
	for _, library := range m.libraries {
		if library.Video == nil {
			continue
		}
		// Ids only: Items() clones every record (chapters and stream lists
		// included) and this runs for each artwork, stream and progress request.
		for _, candidate := range library.Video.IDs() {
			if jellyfin.ID("video", candidate) != id {
				continue
			}
			if item, ok := library.Video.Item(candidate); ok {
				return library, item, true
			}
		}
	}
	return Library{}, video.Item{}, false
}
func (m *multiServer) jellyfinAudioByID(id string) (Library, audio.Item, bool) {
	for _, library := range m.libraries {
		if library.Audio == nil {
			continue
		}
		for _, candidate := range library.Audio.IDs() {
			if jellyfin.ID("audio", candidate) != id {
				continue
			}
			if item, ok := library.Audio.Item(candidate); ok {
				return library, item, true
			}
		}
	}
	return Library{}, audio.Item{}, false
}
func seriesKey(libraryID, show string) string { return libraryID + "\x00" + show }
func seasonKey(libraryID, show string, season int) string {
	return seriesKey(libraryID, show) + "\x00" + strconv.Itoa(season)
}
func (m *multiServer) jellyfinSeriesByID(id string) (Library, string, bool) {
	for _, library := range m.libraries {
		if library.Video == nil {
			continue
		}
		seen := map[string]bool{}
		for _, item := range library.Video.Items() {
			if item.Show == "" || seen[item.Show] {
				continue
			}
			seen[item.Show] = true
			if jellyfin.ID("series", seriesKey(library.ID, item.Show)) == id {
				return library, item.Show, true
			}
		}
	}
	return Library{}, "", false
}
func (m *multiServer) jellyfinSeasonByID(id string) (Library, string, int, bool) {
	for _, library := range m.libraries {
		if library.Video == nil {
			continue
		}
		seen := map[string]bool{}
		for _, item := range library.Video.Items() {
			if item.Show == "" {
				continue
			}
			key := seasonKey(library.ID, item.Show, item.Season)
			if seen[key] {
				continue
			}
			seen[key] = true
			if jellyfin.ID("season", key) == id {
				return library, item.Show, item.Season, true
			}
		}
	}
	return Library{}, "", 0, false
}
func (m *multiServer) jellyfinItems(w http.ResponseWriter, r *http.Request) {
	parent := r.URL.Query().Get("ParentId")
	items := []map[string]any{}
	if parent == "" {
		items = m.jellyfinLibraries()
	} else if library, ok := m.jellyfinLibraryByID(parent); ok {
		if library.Video != nil {
			if library.Kind == "tv" || library.Kind == "anime" {
				seen := map[string]bool{}
				for _, item := range library.Video.Items() {
					if item.Show == "" {
						items = append(items, jellyfin.Video(item.ID, item.Title, item.Show, item.Season, item.Episode, item.Duration, m.jellyfinServerID(), false))
						continue
					}
					if !seen[item.Show] {
						seen[item.Show] = true
						items = append(items, jellyfin.Series(seriesKey(library.ID, item.Show), item.Show, m.jellyfinServerID()))
					}
				}
			} else {
				for _, item := range library.Video.Items() {
					items = append(items, jellyfin.Video(item.ID, item.Title, item.Show, item.Season, item.Episode, item.Duration, m.jellyfinServerID(), false))
				}
			}
		}
		if library.Audio != nil {
			for _, item := range library.Audio.Items() {
				items = append(items, jellyfin.Audio(item.ID, item.Title, item.Artist, item.Album, item.Duration, m.jellyfinServerID()))
			}
		}
	} else if library, show, ok := m.jellyfinSeriesByID(parent); ok {
		seen := map[int]bool{}
		for _, item := range library.Video.Items() {
			if item.Show != show || seen[item.Season] {
				continue
			}
			seen[item.Season] = true
			name := "Season " + strconv.Itoa(item.Season)
			if item.Season == 0 {
				name = "Specials"
			}
			items = append(items, jellyfin.Season(seasonKey(library.ID, show, item.Season), name, seriesKey(library.ID, show), show, item.Season, m.jellyfinServerID()))
		}
	} else if library, show, season, ok := m.jellyfinSeasonByID(parent); ok {
		for _, item := range library.Video.Items() {
			if item.Show == show && item.Season == season {
				items = append(items, jellyfin.Video(item.ID, item.Title, item.Show, item.Season, item.Episode, item.Duration, m.jellyfinServerID(), false))
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"Items": items, "TotalRecordCount": len(items), "StartIndex": 0})
}
func (m *multiServer) jellyfinShows(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/Shows/"), "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	library, show, ok := m.jellyfinSeriesByID(parts[0])
	if !ok {
		http.NotFound(w, r)
		return
	}
	items := []map[string]any{}
	switch parts[1] {
	case "Seasons":
		seen := map[int]bool{}
		for _, item := range library.Video.Items() {
			if item.Show != show || seen[item.Season] {
				continue
			}
			seen[item.Season] = true
			name := "Season " + strconv.Itoa(item.Season)
			if item.Season == 0 {
				name = "Specials"
			}
			items = append(items, jellyfin.Season(seasonKey(library.ID, show, item.Season), name, seriesKey(library.ID, show), show, item.Season, m.jellyfinServerID()))
		}
	case "Episodes":
		seasonID := r.URL.Query().Get("SeasonId")
		for _, item := range library.Video.Items() {
			if item.Show != show {
				continue
			}
			if seasonID != "" && jellyfin.ID("season", seasonKey(library.ID, show, item.Season)) != seasonID {
				continue
			}
			items = append(items, jellyfin.Video(item.ID, item.Title, item.Show, item.Season, item.Episode, item.Duration, m.jellyfinServerID(), false))
		}
	default:
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"Items": items, "TotalRecordCount": len(items), "StartIndex": 0})
}

// serveArtworkHere answers an artwork request on the already-authenticated
// Jellyfin route by calling the artwork handler directly, instead of redirecting
// the client into the browser's authentication namespace.
func (m *multiServer) serveArtworkHere(w http.ResponseWriter, r *http.Request, path, query string, video bool) {
	clone := r.Clone(r.Context())
	clone.URL.Path = path
	clone.URL.RawQuery = query
	if video {
		m.videoArtworkHTTP(w, clone)
		return
	}
	m.audioArtworkHTTP(w, clone)
}

func (m *multiServer) jellyfinItemRoute(w http.ResponseWriter, r *http.Request) {
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/Items/"), "/")
	if strings.HasSuffix(suffix, "/PlaybackInfo") {
		id := strings.TrimSuffix(suffix, "/PlaybackInfo")
		m.jellyfinPlaybackInfo(w, r, id)
		return
	}
	if strings.HasSuffix(suffix, "/Images/Primary") {
		id := strings.TrimSuffix(suffix, "/Images/Primary")
		// Serve the cover from here. Redirecting to /api/video-artwork handed the
		// client a route that only accepts a browser session or Basic credentials,
		// so a Jellyfin client holding a perfectly good Jellyfin token followed the
		// redirect into the sign-in page instead of getting an image.
		if library, item, ok := m.jellyfinVideoByID(id); ok {
			m.serveArtworkHere(w, r, "/api/video-artwork/"+url.PathEscape(library.ID)+"/"+url.PathEscape(item.ID), "v="+url.QueryEscape(item.Revision), true)
			return
		}
		if library, item, ok := m.jellyfinAudioByID(id); ok {
			target := audioArtworkURL(library.ID, item, audio.ArtworkPlayer)
			pathPart, queryPart, _ := strings.Cut(target, "?")
			m.serveArtworkHere(w, r, pathPart, queryPart, false)
			return
		}
	}
	if library, show, ok := m.jellyfinSeriesByID(suffix); ok {
		writeJSON(w, http.StatusOK, jellyfin.Series(seriesKey(library.ID, show), show, m.jellyfinServerID()))
		return
	}
	if library, show, season, ok := m.jellyfinSeasonByID(suffix); ok {
		name := "Season " + strconv.Itoa(season)
		if season == 0 {
			name = "Specials"
		}
		writeJSON(w, http.StatusOK, jellyfin.Season(seasonKey(library.ID, show, season), name, seriesKey(library.ID, show), show, season, m.jellyfinServerID()))
		return
	}
	if _, item, ok := m.jellyfinVideoByID(suffix); ok {
		writeJSON(w, http.StatusOK, jellyfin.Video(item.ID, item.Title, item.Show, item.Season, item.Episode, item.Duration, m.jellyfinServerID(), false))
		return
	}
	if _, item, ok := m.jellyfinAudioByID(suffix); ok {
		writeJSON(w, http.StatusOK, jellyfin.Audio(item.ID, item.Title, item.Artist, item.Album, item.Duration, m.jellyfinServerID()))
		return
	}
	http.NotFound(w, r)
}
