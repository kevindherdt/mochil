package web

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/catalog"
)

// androidPodcastShows exposes only bounded subscription summaries. Episode
// histories remain on the cursor-paged audio endpoint.
func (m *multiServer) androidPodcastShows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.podcasts == nil {
		http.NotFound(w, r)
		return
	}
	m.schedulePodcastRefresh()
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	all := summarizePodcastFeeds(m.podcasts.Feeds())
	filtered := all[:0]
	for _, show := range all {
		if query == "" || strings.Contains(strings.ToLower(show.Title+" "+show.Description), query) {
			filtered = append(filtered, show)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return catalog.NaturalLess(filtered[i].Title, filtered[j].Title) })
	offset, limit := boundedAndroidAudioPage(r, len(filtered))
	end := min(len(filtered), offset+limit)
	w.Header().Set("X-Total-Count", strconv.Itoa(len(filtered)))
	if end < len(filtered) {
		w.Header().Set("X-Next-Offset", strconv.Itoa(end))
	}
	writeJSON(w, http.StatusOK, filtered[offset:end])
}
