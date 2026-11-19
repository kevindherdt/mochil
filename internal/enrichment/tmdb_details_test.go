package enrichment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDetailsReadGenresRuntimeScoreAndPreferTheUSRating(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/3/tv/1396" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected request %s", r.URL)
		}
		body := `{"genres":[{"name":"Drama"},{"name":"Crime"}],"episode_run_time":[47],"vote_average":8.912,"content_ratings":{"results":[{"iso_3166_1":"DE","rating":"16"},{"iso_3166_1":"US","rating":"TV-MA"}]}}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	tmdb := NewTMDB("secret", client)
	path := t.TempDir() + "/tmdb-details.json"
	tmdb.PersistDetails(path)
	for range 2 {
		got, err := tmdb.Details(context.Background(), "tv", "1396")
		if err != nil || strings.Join(got.Genres, ",") != "Drama,Crime" || got.RuntimeMinutes != 47 || got.Rating != 8.9 || got.Certification != "TV-MA" {
			t.Fatalf("details = %#v, %v", got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("TMDB was asked %d times, want 1", calls)
	}
	again := NewTMDB("secret", client)
	again.PersistDetails(path)
	if got, err := again.Details(context.Background(), "tv", "1396"); err != nil || got.Certification != "TV-MA" || calls != 1 {
		t.Fatalf("after a restart: %#v, %v, calls %d", got, err, calls)
	}
	if _, err := tmdb.Details(context.Background(), "tv", "../x"); err == nil {
		t.Fatal("a non-numeric id was accepted")
	}
}

func TestLegacyDetailsSurviveFailedRefreshAndRestart(t *testing.T) {
	for _, failure := range []string{"http", "transport", "decode"} {
		t.Run(failure, func(t *testing.T) {
			path := t.TempDir() + "/details.json"
			legacy := `{"tv/42":{"created":"` + time.Now().Format(time.RFC3339Nano) + `","details":{"genres":["Drama"],"runtime_minutes":47,"rating":8.9,"certification":"TV-MA"}}}`
			if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			fail := true
			client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/3/tv/42" {
					calls++
					if fail {
						switch failure {
						case "transport":
							return nil, fmt.Errorf("offline")
						case "http":
							return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
						case "decode":
							return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{")), Header: http.Header{}}, nil
						}
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"genres":[{"name":"Updated"}],"credits":{"cast":[{"name":"Actor"}]}}`)), Header: http.Header{}}, nil
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: http.Header{}}, nil
			})}
			tmdb := NewTMDB("secret", client)
			tmdb.PersistDetails(path)
			checkLegacy := func(instance *TMDB) {
				t.Helper()
				got, err := instance.Details(context.Background(), "tv", "42")
				if err != nil || strings.Join(got.Genres, ",") != "Drama" || got.RuntimeMinutes != 47 || got.Rating != 8.9 || got.Certification != "TV-MA" {
					t.Fatalf("legacy fallback = %#v, %v", got, err)
				}
			}
			checkLegacy(tmdb)
			// A separate successful write must not erase the legacy entry on disk.
			if _, err := tmdb.Details(context.Background(), "movie", "1"); err != nil {
				t.Fatal(err)
			}
			restarted := NewTMDB("secret", client)
			restarted.PersistDetails(path)
			checkLegacy(restarted)
			if calls != 2 {
				t.Fatalf("failed refreshes = %d, want 2", calls)
			}
			fail = false
			got, err := restarted.Details(context.Background(), "tv", "42")
			if err != nil || calls != 3 || strings.Join(got.Genres, ",") != "Updated" || len(got.Cast) != 1 {
				t.Fatalf("retry = %#v, %v, calls %d", got, err, calls)
			}
			restartedAgain := NewTMDB("secret", client)
			restartedAgain.PersistDetails(path)
			if got, err = restartedAgain.Details(context.Background(), "tv", "42"); err != nil || calls != 3 || len(got.Cast) != 1 {
				t.Fatalf("upgraded cache = %#v, %v, calls %d", got, err, calls)
			}
		})
	}
}

func TestDetailsUseTheMovieReleaseCertification(t *testing.T) {
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		body := `{"genres":[{"name":"Sci-Fi"}],"runtime":116,"release_dates":{"results":[{"iso_3166_1":"US","release_dates":[{"certification":""},{"certification":"R"}]}]}}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	got, err := NewTMDB("secret", client).Details(context.Background(), "movie", "78")
	if err != nil || got.RuntimeMinutes != 116 || got.Genres[0] != "Sci-Fi" || got.Certification != "R" {
		t.Fatalf("details = %#v, %v", got, err)
	}
}
