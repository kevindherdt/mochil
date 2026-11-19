package enrichment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TMDBQuery struct {
	Title string
	Kind  string
}

type TMDB struct {
	token       string
	client      *http.Client
	now         func() time.Time
	gate        chan struct{}
	mu          sync.Mutex
	cache       map[string]cacheEntry
	episodes    map[string]episodeCacheEntry
	episodePath string
	details     map[string]storedDetails
	detailsPath string
}

func NewTMDB(token string, client *http.Client) *TMDB {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &TMDB{token: strings.TrimSpace(token), client: client, now: time.Now, gate: make(chan struct{}, 1), cache: make(map[string]cacheEntry)}
}

func (t *TMDB) Check(ctx context.Context) error {
	if t.token == "" {
		return ErrProviderNotConnected
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.themoviedb.org/3/authentication", nil)
	if err != nil {
		return ErrProviderUnavailable
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+t.token)
	response, err := t.client.Do(request)
	if err != nil {
		return ErrProviderUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return ErrProviderRejected
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return ErrProviderRateLimited
	}
	if response.StatusCode != http.StatusOK {
		return ErrProviderUnavailable
	}
	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 64<<10)).Decode(&payload); err != nil || !payload.Success {
		return ErrProviderRejected
	}
	return nil
}

func (t *TMDB) Search(ctx context.Context, query TMDBQuery) ([]Suggestion, error) {
	title := strings.TrimSpace(query.Title)
	mediaType := "movie"
	if query.Kind == "tv" || query.Kind == "anime" {
		mediaType = "tv"
	} else if query.Kind != "movies" {
		return nil, errors.New("TMDB lookup requires a movie, television, or anime library")
	}
	if t.token == "" {
		return nil, errors.New("TMDB is not connected")
	}
	if title == "" || len(title) > 500 {
		return nil, errors.New("TMDB lookup requires a bounded title")
	}
	key := mediaType + "\x00" + strings.ToLower(title)
	if results, ok := t.cached(key); ok {
		return results, nil
	}
	select {
	case t.gate <- struct{}{}:
		defer func() { <-t.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if results, ok := t.cached(key); ok {
		return results, nil
	}

	values := url.Values{"query": {title}, "include_adult": {"false"}, "language": {"en-US"}, "page": {"1"}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.themoviedb.org/3/search/"+mediaType+"?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+t.token)
	response, err := t.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("TMDB request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned HTTP %d", response.StatusCode)
	}
	var payload struct {
		Results []struct {
			ID           int64  `json:"id"`
			Title        string `json:"title"`
			Name         string `json:"name"`
			ReleaseDate  string `json:"release_date"`
			FirstAir     string `json:"first_air_date"`
			Overview     string `json:"overview"`
			PosterPath   string `json:"poster_path"`
			BackdropPath string `json:"backdrop_path"`
		} `json:"results"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode TMDB response: %w", err)
	}
	results := make([]Suggestion, 0, min(len(payload.Results), maximumResults))
	for _, item := range payload.Results {
		if len(results) == maximumResults || item.ID <= 0 {
			continue
		}
		name, date := strings.TrimSpace(item.Title), item.ReleaseDate
		if mediaType == "tv" {
			name, date = strings.TrimSpace(item.Name), item.FirstAir
		}
		if name == "" {
			continue
		}
		id := strconv.FormatInt(item.ID, 10)
		suggestion := Suggestion{Title: name, Year: year(date), Overview: boundedText(item.Overview, 2000), MediaType: mediaType, Provider: "TMDB", ProviderID: id, SourceURL: "https://www.themoviedb.org/" + mediaType + "/" + id}
		if strings.HasPrefix(item.PosterPath, "/") && !strings.Contains(item.PosterPath, "..") {
			suggestion.CoverURL = "https://image.tmdb.org/t/p/w342" + item.PosterPath
		}
		if strings.HasPrefix(item.BackdropPath, "/") && !strings.Contains(item.BackdropPath, "..") {
			suggestion.BackdropURL = "https://image.tmdb.org/t/p/w1280" + item.BackdropPath
		}
		results = append(results, suggestion)
	}
	t.store(key, results)
	return append([]Suggestion(nil), results...), nil
}

func year(date string) int {
	if len(date) < 4 {
		return 0
	}
	value, _ := strconv.Atoi(date[:4])
	return value
}

func boundedText(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}

func (t *TMDB) cached(key string) ([]Suggestion, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.cache[key]
	if !ok || t.now().Sub(entry.created) >= 24*time.Hour {
		delete(t.cache, key)
		return nil, false
	}
	return append([]Suggestion(nil), entry.results...), true
}

func (t *TMDB) store(key string, results []Suggestion) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.cache) >= maximumCache {
		var oldestKey string
		var oldest time.Time
		for candidate, entry := range t.cache {
			if oldest.IsZero() || entry.created.Before(oldest) {
				oldestKey, oldest = candidate, entry.created
			}
		}
		delete(t.cache, oldestKey)
	}
	t.cache[key] = cacheEntry{results: append([]Suggestion(nil), results...), created: t.now()}
}
