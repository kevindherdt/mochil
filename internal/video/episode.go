package video

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	episodeMarker = regexp.MustCompile(`(?i)\bS0*([0-9]{1,3})E0*([0-9]{1,4})\b`)
	seasonFolder  = regexp.MustCompile(`(?i)^season[ ._-]*0*([0-9]{1,3})$`)
	providerTag   = regexp.MustCompile(`(?i)\s*\[(?:tvdbid|tmdbid|imdbid)=[^]]+\]\s*`)
	bracketed     = regexp.MustCompile(`\s*[\[(][^\])]*[\])]`)
	fansubEpisode = regexp.MustCompile(`(?i)(?:\bS0*([0-9]{1,3})\s*-\s*|\s-\s)0*([0-9]{1,4})(?:v[0-9])?\s*$`)
	releaseTail   = regexp.MustCompile(`(?i)\s+(?:WEB[- ]?DL|WEBRip|BluRay|BDRip|HDTV|DVDRip|Remux)(?:[- ._]*(?:2160|1080|720|576|480)p?)?.*$`)
)

// InferEpisode derives conservative episodic identity from common show folder
// layouts and SxxEyy filenames. It never guesses an episode number from an
// unrelated bare number such as a year or resolution.
func InferEpisode(item *Item) {
	if item == nil {
		return
	}
	parts := strings.Split(filepath.ToSlash(item.Path), "/")
	match := episodeMarker.FindStringSubmatch(item.Title)
	if len(match) != 3 {
		inferFansubEpisode(item, parts)
		return
	}
	item.Season, _ = strconv.Atoi(match[1])
	item.Episode, _ = strconv.Atoi(match[2])
	if len(parts) >= 2 {
		item.Show = strings.TrimSpace(providerTag.ReplaceAllString(parts[0], ""))
	}
	if len(parts) >= 3 {
		if folder := seasonFolder.FindStringSubmatch(parts[len(parts)-2]); len(folder) == 2 {
			if season, err := strconv.Atoi(folder[1]); err == nil {
				item.Season = season
			}
		}
	}
	end := episodeMarker.FindStringIndex(item.Title)
	if len(end) == 2 {
		title := strings.TrimSpace(strings.TrimLeft(item.Title[end[1]:], " -–—:._"))
		title = strings.TrimSpace(releaseTail.ReplaceAllString(title, ""))
		if title != "" {
			item.EpisodeTitle = title
		}
	}
}

// inferFansubEpisode reads release names such as "[Group] Show S2 - 01 (BD 576p) [CRC]" inside "Show/Season 02/", where
// the season folder vouches for the layout. Without that folder a trailing " - 01" is too easily a title.
func inferFansubEpisode(item *Item, parts []string) {
	if len(parts) < 3 {
		return
	}
	folder := seasonFolder.FindStringSubmatch(parts[len(parts)-2])
	if len(folder) != 2 {
		return
	}
	name := strings.TrimSpace(bracketed.ReplaceAllString(item.Title, ""))
	match := fansubEpisode.FindStringSubmatch(name)
	if len(match) != 3 {
		return
	}
	item.Season, _ = strconv.Atoi(folder[1])
	item.Episode, _ = strconv.Atoi(match[2])
	item.Show = strings.TrimSpace(providerTag.ReplaceAllString(parts[0], ""))
}
