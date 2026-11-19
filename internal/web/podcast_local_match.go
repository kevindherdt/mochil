package web

import (
	"strings"
	"unicode"

	"mochil/internal/audio"
	"mochil/internal/podcast"
)

// matchLocalPodcastEpisodes links RSS metadata to existing server-local files
// without modifying either catalogue. Only a unique show/title match is safe;
// ambiguous files remain visible as ordinary local podcast media.
func matchLocalPodcastEpisodes(items []audio.Item, episodes []podcast.Episode) map[string]audio.Item {
	byIdentity := make(map[string][]audio.Item)
	for _, item := range items {
		show := item.Album
		if strings.TrimSpace(show) == "" {
			show = item.Artist
		}
		key := podcastMatchText(show) + "\x00" + podcastMatchText(item.Title)
		if key != "\x00" {
			byIdentity[key] = append(byIdentity[key], item)
		}
	}
	matched := make(map[string]audio.Item)
	for _, episode := range episodes {
		candidates := byIdentity[podcastMatchText(episode.FeedTitle)+"\x00"+podcastMatchText(episode.Title)]
		if len(candidates) == 1 {
			matched[episode.ID] = candidates[0]
		}
	}
	return matched
}

func podcastMatchText(value string) string {
	var out strings.Builder
	space := true
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			out.WriteRune(character)
			space = false
		} else if !space {
			out.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(out.String())
}
