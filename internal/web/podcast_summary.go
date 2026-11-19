package web

import (
	"net/url"
	"time"

	"mochil/internal/podcast"
)

// podcastFeedSummary keeps the subscription screen independent from the
// potentially large episode history. Episodes are paged through /api/audio.
type podcastFeedSummary struct {
	ID           string             `json:"id"`
	URL          string             `json:"url"`
	Title        string             `json:"title"`
	Description  string             `json:"description,omitempty"`
	ArtworkURL   string             `json:"artwork_url,omitempty"`
	Updated      time.Time          `json:"updated"`
	LastError    string             `json:"last_error,omitempty"`
	Policy       podcast.FeedPolicy `json:"policy,omitempty"`
	EpisodeCount int                `json:"episode_count"`
}

func summarizePodcastFeeds(feeds []podcast.Feed) []podcastFeedSummary {
	summaries := make([]podcastFeedSummary, len(feeds))
	for i, feed := range feeds {
		artworkURL := ""
		if feed.ArtworkURL != "" {
			artworkURL = "/api/podcast-artwork/feed/" + url.PathEscape(feed.ID)
		}
		summaries[i] = podcastFeedSummary{
			ID:           feed.ID,
			URL:          feed.URL,
			Title:        feed.Title,
			Description:  feed.Description,
			ArtworkURL:   artworkURL,
			Updated:      feed.Updated,
			LastError:    feed.LastError,
			Policy:       feed.Policy,
			EpisodeCount: len(feed.Episodes),
		}
	}
	return summaries
}
