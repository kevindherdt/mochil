package web

import (
	"testing"

	"mochil/internal/podcast"
)

func TestPodcastFeedSummaryOmitsEpisodes(t *testing.T) {
	summaries := summarizePodcastFeeds([]podcast.Feed{{ID: "feed", Title: "Show", Episodes: []podcast.Episode{{ID: "one"}, {ID: "two"}}}})
	if len(summaries) != 1 || summaries[0].EpisodeCount != 2 || summaries[0].Title != "Show" {
		t.Fatalf("unexpected summaries: %#v", summaries)
	}
}

func TestPodcastFeedSummaryUsesSameOriginArtworkProxy(t *testing.T) {
	summaries := summarizePodcastFeeds([]podcast.Feed{{
		ID: "feed", ArtworkURL: "https://cdn.example/cover.jpg",
		Episodes: []podcast.Episode{{ID: "episode/one", ArtworkURL: "https://cdn.example/cover.jpg"}},
	}})
	if got, want := summaries[0].ArtworkURL, "/api/podcast-artwork/feed/feed"; got != want {
		t.Fatalf("artwork URL = %q, want %q", got, want)
	}
}
