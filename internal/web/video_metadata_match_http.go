package web

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"mochil/internal/enrichment"
	"mochil/internal/video"
)

// maximumMatchesPerRequest keeps one request within TMDB's pace; the caller continues from next_offset.
const maximumMatchesPerRequest = 40

type videoMatchTarget struct {
	id   string // item id; empty for a show
	show string
	name string
	// saved is the existing TMDB match that only lacks its wide backdrop; the run adds that and changes nothing else.
	saved *video.MetadataOverride
}

type videoMatchRow struct {
	Name       string `json:"name"`
	Title      string `json:"title,omitempty"`
	Year       int    `json:"year,omitempty"`
	ProviderID string `json:"provider_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// apiVideoMetadataMatch finds TMDB matches for every movie or show that has none yet. Without apply=1 it only reports
// what it would save; with it, only matches whose words and year agree are saved, and nothing already matched by hand
// is touched. Everything else is listed for a person to choose.
func (m *multiServer) apiVideoMetadataMatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	libraryID, err := url.PathUnescape(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-metadata-match/"), "/"))
	library, ok := m.library(libraryID)
	if err != nil || !ok || library.Video == nil || library.MetadataProvider != "tmdb" || m.tmdb == nil || m.videoMetadata == nil {
		http.NotFound(w, r)
		return
	}
	apply := r.URL.Query().Get("apply") == "1"
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	targets := m.videoMatchTargets(library, libraryID)
	start, end := pageWindow(offset, maximumMatchesPerRequest, len(targets))
	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Second)
	defer cancel()
	matched, uncertain := []videoMatchRow{}, []videoMatchRow{}
	for _, target := range targets[start:end] {
		title, year := video.ParseMediaName(target.name)
		if target.saved != nil {
			title, year = target.saved.Title, target.saved.Year
		}
		results, searchErr := m.tmdb.Search(ctx, enrichment.TMDBQuery{Title: title, Kind: library.Kind})
		if searchErr != nil {
			uncertain = append(uncertain, videoMatchRow{Name: target.name, Reason: "search failed"})
			continue
		}
		var pick *enrichment.Suggestion
		for i := range results {
			if target.saved != nil && results[i].ProviderID == target.saved.ProviderID && results[i].BackdropURL != "" {
				pick = &results[i]
				break
			}
			if target.saved == nil && video.ConfidentMatch(title, year, results[i].Title, results[i].Year) {
				pick = &results[i]
				break
			}
		}
		if pick == nil && target.saved != nil {
			continue // TMDB has no wide image for it; the poster stays as it is
		}
		if pick == nil {
			row := videoMatchRow{Name: target.name, Reason: "no clear match"}
			if len(results) > 0 {
				row.Title, row.Year = results[0].Title, results[0].Year
			}
			uncertain = append(uncertain, row)
			continue
		}
		override := video.MetadataOverride{Title: pick.Title, Year: pick.Year, Overview: pick.Overview, ArtworkURL: pick.CoverURL, BackdropURL: pick.BackdropURL, ProviderID: pick.ProviderID, Provider: "TMDB"}
		reason := ""
		if target.saved != nil {
			override, reason = *target.saved, "adds a wide backdrop"
			override.BackdropURL = pick.BackdropURL
		}
		if apply {
			var saveErr error
			if target.id != "" {
				saveErr = m.videoMetadata.Set(libraryID, target.id, override)
			} else {
				saveErr = m.videoMetadata.SetShow(libraryID, target.show, override)
			}
			if saveErr != nil {
				uncertain = append(uncertain, videoMatchRow{Name: target.name, Title: pick.Title, Year: pick.Year, Reason: "could not save"})
				continue
			}
		}
		matched = append(matched, videoMatchRow{Name: target.name, Title: pick.Title, Year: pick.Year, ProviderID: pick.ProviderID, Reason: reason})
	}
	body := map[string]any{"applied": apply, "matched": matched, "uncertain": uncertain, "unmatched_total": len(targets)}
	if end < len(targets) {
		body["next_offset"] = end
	}
	writeJSON(w, http.StatusOK, body)
}

// videoMatchTargets lists the movies or shows with no TMDB match yet, in a stable order so offsets can continue a run.
func (m *multiServer) videoMatchTargets(library Library, libraryID string) []videoMatchTarget {
	var targets []videoMatchTarget
	seen := map[string]bool{}
	for _, item := range library.Video.Items() {
		if library.Kind == "tv" || library.Kind == "anime" {
			if item.Show == "" || seen[item.Show] {
				continue
			}
			seen[item.Show] = true
			saved, matched := m.videoMetadata.GetShow(libraryID, item.Show)
			if !matched {
				targets = append(targets, videoMatchTarget{show: item.Show, name: item.Show})
			} else if needsBackdrop(saved) {
				targets = append(targets, videoMatchTarget{show: item.Show, name: item.Show, saved: &saved})
			}
			continue
		}
		saved, matched := m.videoMetadata.Get(libraryID, item.ID)
		if !matched {
			targets = append(targets, videoMatchTarget{id: item.ID, name: item.Title})
		} else if needsBackdrop(saved) {
			targets = append(targets, videoMatchTarget{id: item.ID, name: item.Title, saved: &saved})
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].name < targets[j].name })
	return targets
}

// needsBackdrop marks TMDB matches saved before backdrops were kept.
func needsBackdrop(saved video.MetadataOverride) bool {
	return saved.Provider == "TMDB" && saved.BackdropURL == "" && saved.Title != ""
}
