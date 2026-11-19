package web

import (
	"testing"

	"mochil/internal/audio"
	"mochil/internal/podcast"
)

func TestMatchLocalPodcastEpisodesUsesUniqueNormalizedShowAndTitle(t *testing.T) {
	items := []audio.Item{{ID: "local", Album: "The History of Egypt Podcast", Title: "06c - Khufu & the Magicians (Westcar Papyrus)"}}
	episodes := []podcast.Episode{{ID: "rss", FeedTitle: "The History of Egypt Podcast", Title: "06c: Khufu & the Magicians (Westcar Papyrus)"}}
	matched := matchLocalPodcastEpisodes(items, episodes)
	if matched["rss"].ID != "local" {
		t.Fatalf("matched=%v", matched)
	}
}

func TestMatchLocalPodcastEpisodesRejectsAmbiguousAndWrongShowFiles(t *testing.T) {
	items := []audio.Item{
		{ID: "one", Album: "Show", Title: "Episode 1"},
		{ID: "two", Album: "Show", Title: "Episode 1"},
		{ID: "other", Album: "Other Show", Title: "Episode 2"},
	}
	episodes := []podcast.Episode{{ID: "rss-one", FeedTitle: "Show", Title: "Episode 1"}, {ID: "rss-two", FeedTitle: "Show", Title: "Episode 2"}}
	if matched := matchLocalPodcastEpisodes(items, episodes); len(matched) != 0 {
		t.Fatalf("matched=%v", matched)
	}
}
