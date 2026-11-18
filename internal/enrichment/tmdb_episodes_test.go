package enrichment

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSeasonEpisodesReadsNamesStoriesAndStillsOnceADay(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/3/tv/1396/season/2" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected request %s", r.URL)
		}
		body := `{"episodes":[{"episode_number":1,"name":"Seven Thirty-Seven","overview":"Walt returns.","still_path":"/a.jpg"},{"episode_number":2,"name":"Grilled","still_path":"../x"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	tmdb := NewTMDB("secret", client)
	for range 2 {
		episodes, err := tmdb.SeasonEpisodes(context.Background(), "1396", 2)
		if err != nil || episodes[1].Name != "Seven Thirty-Seven" || episodes[1].StillURL != "https://image.tmdb.org/t/p/w1280/a.jpg" || episodes[2].StillURL != "" {
			t.Fatalf("episodes = %#v, %v", episodes, err)
		}
	}
	if calls != 1 {
		t.Fatalf("TMDB was asked %d times, want 1", calls)
	}
	if _, err := tmdb.SeasonEpisodes(context.Background(), "../x", 1); err == nil {
		t.Fatal("a non-numeric show id was accepted")
	}
}

func TestSeasonEpisodesSurviveARestart(t *testing.T) {
	path := t.TempDir() + "/tmdb-episodes.json"
	calls := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"episodes":[{"episode_number":3,"name":"Three"}]}`)), Header: http.Header{}}, nil
	})}
	first := NewTMDB("secret", client)
	first.PersistEpisodes(path)
	if _, err := first.SeasonEpisodes(context.Background(), "5", 1); err != nil {
		t.Fatal(err)
	}
	second := NewTMDB("secret", client)
	second.PersistEpisodes(path)
	episodes, err := second.SeasonEpisodes(context.Background(), "5", 1)
	if err != nil || episodes[3].Name != "Three" || calls != 1 {
		t.Fatalf("episodes = %#v, %v after %d TMDB calls", episodes, err, calls)
	}
}
