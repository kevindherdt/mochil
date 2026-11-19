package web

import (
	"encoding/json"
	"testing"

	"mochil/internal/podcast"
)

func TestPodcastViewInvalidDurationRemainsJSONEncodable(t *testing.T) {
	m := &multiServer{}
	for _, duration := range []string{"NaN", "Inf", "1e308:00", "-60"} {
		view := m.podcastView("fixture", Library{ID: "podcasts", Kind: "podcasts"},
			podcast.Episode{ID: "episode", Title: "Episode", Duration: duration})
		if view.Duration != 0 {
			t.Fatalf("invalid feed duration %q produced %v", duration, view.Duration)
		}
		if _, err := json.Marshal([]audioView{view}); err != nil {
			t.Fatalf("feed duration %q broke episode JSON: %v", duration, err)
		}
	}
}
