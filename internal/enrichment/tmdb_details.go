package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// detailsCacheLife is how long a title's facts are trusted. Genres, runtime and ratings move slowly.
const detailsCacheLife = 14 * 24 * time.Hour

// TitleDetails is what TMDB knows about a whole show or movie that the files do not: genres, how long it runs, its age rating and its score.
type TitleDetails struct {
	Genres         []string     `json:"genres,omitempty"`
	RuntimeMinutes int          `json:"runtime_minutes,omitempty"`
	Rating         float64      `json:"rating,omitempty"`
	Certification  string       `json:"certification,omitempty"`
	Cast           []CastMember `json:"cast,omitempty"`
}

type CastMember struct {
	Name        string `json:"name"`
	Character   string `json:"character,omitempty"`
	ProfilePath string `json:"profile_path,omitempty"`
}

type storedDetails struct {
	Created time.Time    `json:"created"`
	Details TitleDetails `json:"details"`
	Version int          `json:"version"`
}

// Details returns the genres, runtime, age rating and score of one TMDB show or movie. Results are kept for two weeks, on disk too.
func (t *TMDB) Details(ctx context.Context, mediaType, id string) (TitleDetails, error) {
	if t.token == "" {
		return TitleDetails{}, ErrProviderNotConnected
	}
	if mediaType != "tv" && mediaType != "movie" {
		return TitleDetails{}, fmt.Errorf("TMDB details need a movie or television title")
	}
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		return TitleDetails{}, fmt.Errorf("TMDB id must be numeric")
	}
	key := mediaType + "/" + id
	if details, ok := t.cachedDetails(key); ok {
		return details, nil
	}
	select {
	case t.gate <- struct{}{}:
		defer func() { <-t.gate }()
	case <-ctx.Done():
		return TitleDetails{}, ctx.Err()
	}
	if details, ok := t.cachedDetails(key); ok {
		return details, nil
	}
	extra := "content_ratings"
	if mediaType == "movie" {
		extra = "release_dates"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.themoviedb.org/3/"+mediaType+"/"+id+"?language=en-US&append_to_response="+extra+",credits", nil)
	if err != nil {
		return TitleDetails{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+t.token)
	response, err := t.client.Do(request)
	if err != nil {
		return t.detailsRefreshError(ctx, key, fmt.Errorf("TMDB request failed: %w", err))
	}
	defer response.Body.Close()
	var details TitleDetails
	switch response.StatusCode {
	case http.StatusOK:
		var payload struct {
			Genres         []struct{ Name string } `json:"genres"`
			Runtime        int                     `json:"runtime"`
			EpisodeRuntime []int                   `json:"episode_run_time"`
			VoteAverage    float64                 `json:"vote_average"`
			Credits        struct {
				Cast []struct {
					Name        string `json:"name"`
					Character   string `json:"character"`
					ProfilePath string `json:"profile_path"`
				} `json:"cast"`
			} `json:"credits"`
			ContentRatings struct {
				Results []struct {
					Country string `json:"iso_3166_1"`
					Rating  string `json:"rating"`
				} `json:"results"`
			} `json:"content_ratings"`
			ReleaseDates struct {
				Results []struct {
					Country string `json:"iso_3166_1"`
					Dates   []struct {
						Certification string `json:"certification"`
					} `json:"release_dates"`
				} `json:"results"`
			} `json:"release_dates"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 2<<20)).Decode(&payload); err != nil {
			return t.detailsRefreshError(ctx, key, fmt.Errorf("decode TMDB details: %w", err))
		}
		for _, genre := range payload.Genres {
			if name := strings.TrimSpace(genre.Name); name != "" && len(details.Genres) < 4 {
				details.Genres = append(details.Genres, boundedText(name, 40))
			}
		}
		for _, member := range payload.Credits.Cast {
			if name := strings.TrimSpace(member.Name); name != "" && len(details.Cast) < 10 {
				profile := member.ProfilePath
				if !ValidTMDBProfilePath(profile) {
					profile = ""
				}
				details.Cast = append(details.Cast, CastMember{Name: boundedText(name, 100), Character: boundedText(strings.TrimSpace(member.Character), 100), ProfilePath: profile})
			}
		}
		details.RuntimeMinutes = payload.Runtime
		if details.RuntimeMinutes <= 0 && len(payload.EpisodeRuntime) > 0 {
			details.RuntimeMinutes = payload.EpisodeRuntime[0]
		}
		if payload.VoteAverage > 0 {
			details.Rating = math.Round(payload.VoteAverage*10) / 10
		}
		var found []string
		for _, row := range payload.ContentRatings.Results {
			if row.Rating != "" {
				found = append(found, row.Country+"\x00"+row.Rating)
			}
		}
		for _, row := range payload.ReleaseDates.Results {
			for _, date := range row.Dates {
				if date.Certification != "" {
					found = append(found, row.Country+"\x00"+date.Certification)
					break
				}
			}
		}
		details.Certification = pickCertification(found)
	case http.StatusNotFound:
		// TMDB has no such title; remember that so the page does not ask again.
	default:
		return t.detailsRefreshError(ctx, key, fmt.Errorf("TMDB returned HTTP %d", response.StatusCode))
	}
	t.mu.Lock()
	if t.details == nil {
		t.details = map[string]storedDetails{}
	}
	if len(t.details) >= 2000 {
		t.details = map[string]storedDetails{}
	}
	t.details[key] = storedDetails{Created: t.now(), Details: details, Version: 2}
	t.saveDetailsLocked()
	t.mu.Unlock()
	return details, nil
}

// pickCertification prefers the US rating, then the first one TMDB lists. Rows are "COUNTRY\x00rating".
func pickCertification(rows []string) string {
	first := ""
	for _, row := range rows {
		country, rating, _ := strings.Cut(row, "\x00")
		rating = boundedText(rating, 12)
		if country == "US" {
			return rating
		}
		if first == "" {
			first = rating
		}
	}
	return first
}

// PersistDetails keeps the title cache in a file and loads what an earlier run saved.
func (t *TMDB) PersistDetails(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.detailsPath = path
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var stored map[string]storedDetails
	if json.Unmarshal(raw, &stored) != nil {
		return
	}
	if t.details == nil {
		t.details = map[string]storedDetails{}
	}
	for key, entry := range stored {
		if t.now().Sub(entry.Created) < detailsCacheLife {
			t.details[key] = entry
		}
	}
}

func (t *TMDB) saveDetailsLocked() {
	if t.detailsPath == "" {
		return
	}
	raw, err := json.Marshal(t.details)
	if err != nil || os.MkdirAll(filepath.Dir(t.detailsPath), 0o755) != nil {
		return
	}
	temporary := t.detailsPath + ".tmp"
	if os.WriteFile(temporary, raw, 0o600) == nil {
		_ = os.Rename(temporary, t.detailsPath)
	}
}

func (t *TMDB) cachedDetails(key string) (TitleDetails, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.details[key]
	if !ok || t.now().Sub(entry.Created) >= detailsCacheLife {
		delete(t.details, key)
		return TitleDetails{}, false
	}
	if entry.Version != 2 {
		return TitleDetails{}, false
	}
	return entry.Details, true
}

// A legacy entry remains usable until its original expiry, but must be retried
// on the next request rather than being recorded as a successful upgrade.
func (t *TMDB) detailsRefreshError(ctx context.Context, key string, err error) (TitleDetails, error) {
	if ctx.Err() != nil {
		return TitleDetails{}, err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.details[key]
	if ok && entry.Version >= 0 && entry.Version < 2 && t.now().Sub(entry.Created) < detailsCacheLife {
		return entry.Details, nil
	}
	return TitleDetails{}, err
}
