package web

import (
	"testing"

	"mochil/internal/audio"
)

func TestMatchesPlayedFilter(t *testing.T) {
	played := &audio.Progress{Finished: true}
	inProgress := &audio.Progress{Position: 42, Duration: 100}
	tests := []struct {
		name     string
		filter   string
		progress *audio.Progress
		want     bool
	}{
		{"all without progress", "all", nil, true},
		{"unplayed without progress", "unplayed", nil, true},
		{"unplayed in progress", "unplayed", inProgress, true},
		{"unplayed finished", "unplayed", played, false},
		{"played finished", "played", played, true},
		{"played in progress", "played", inProgress, false},
		{"continue in progress", "continue", inProgress, true},
		{"continue finished", "continue", played, false},
		{"continue without progress", "continue", nil, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := matchesPlayedFilter(test.filter, test.progress); got != test.want {
				t.Fatalf("matchesPlayedFilter(%q, %+v) = %v, want %v", test.filter, test.progress, got, test.want)
			}
		})
	}
}
