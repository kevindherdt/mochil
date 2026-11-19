// Package enrichment provides opt-in, review-only external metadata suggestions.
package enrichment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	maximumResults = 5
	maximumCache   = 1000
)

type Suggestion struct {
	Title       string   `json:"title"`
	Authors     []string `json:"authors,omitempty"`
	Year        int      `json:"year,omitempty"`
	Overview    string   `json:"overview,omitempty"`
	MediaType   string   `json:"media_type,omitempty"`
	CoverURL    string   `json:"cover_url,omitempty"`
	BackdropURL string   `json:"backdrop_url,omitempty"`
	SourceURL   string   `json:"source_url"`
	Provider    string   `json:"provider"`
	ProviderID  string   `json:"provider_id"`
}

type Query struct {
	Title  string
	Author string
}

type cacheEntry struct {
	results []Suggestion
	created time.Time
}

type OpenLibrary struct {
	client *http.Client
	now    func() time.Time
	mu     sync.Mutex
	cache  map[string]cacheEntry
	gate   chan struct{}
}

func NewOpenLibrary(client *http.Client) *OpenLibrary {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &OpenLibrary{client: client, now: time.Now, cache: make(map[string]cacheEntry), gate: make(chan struct{}, 1)}
}

func (o *OpenLibrary) Search(ctx context.Context, query Query) ([]Suggestion, error) {
	title := strings.TrimSpace(query.Title)
	author := strings.TrimSpace(query.Author)
	if title == "" || len(title) > 500 || len(author) > 500 {
		return nil, errors.New("metadata lookup requires a bounded title")
	}
	key := strings.ToLower(title + "\x00" + author)
	if results, ok := o.cached(key); ok {
		return results, nil
	}
	select {
	case o.gate <- struct{}{}:
		defer func() { <-o.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if results, ok := o.cached(key); ok {
		return results, nil
	}

	values := url.Values{"title": {title}, "limit": {"5"}, "fields": {"key,title,author_name,cover_i"}}
	if author != "" {
		values.Set("author", author)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openlibrary.org/search.json?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Mochil/1.0 (self-hosted metadata review)")
	response, err := o.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Open Library request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Open Library returned HTTP %d", response.StatusCode)
	}
	var payload struct {
		Docs []struct {
			Key     string   `json:"key"`
			Title   string   `json:"title"`
			Authors []string `json:"author_name"`
			CoverID int64    `json:"cover_i"`
		} `json:"docs"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 1<<20))
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Open Library response: %w", err)
	}
	results := make([]Suggestion, 0, min(len(payload.Docs), maximumResults))
	for _, item := range payload.Docs {
		if len(results) == maximumResults || strings.TrimSpace(item.Title) == "" || !strings.HasPrefix(item.Key, "/works/") {
			continue
		}
		id := strings.TrimPrefix(item.Key, "/works/")
		suggestion := Suggestion{Title: item.Title, Authors: append([]string(nil), item.Authors...), Provider: "Open Library", ProviderID: id, SourceURL: "https://openlibrary.org" + item.Key}
		if item.CoverID > 0 {
			suggestion.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg?default=false", item.CoverID)
		}
		results = append(results, suggestion)
	}
	o.store(key, results)
	return append([]Suggestion(nil), results...), nil
}

func (o *OpenLibrary) cached(key string) ([]Suggestion, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	entry, ok := o.cache[key]
	if !ok || o.now().Sub(entry.created) >= 24*time.Hour {
		delete(o.cache, key)
		return nil, false
	}
	return append([]Suggestion(nil), entry.results...), true
}

func (o *OpenLibrary) store(key string, results []Suggestion) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.cache) >= maximumCache {
		var oldestKey string
		var oldest time.Time
		for candidate, entry := range o.cache {
			if oldest.IsZero() || entry.created.Before(oldest) {
				oldestKey, oldest = candidate, entry.created
			}
		}
		delete(o.cache, oldestKey)
	}
	o.cache[key] = cacheEntry{results: append([]Suggestion(nil), results...), created: o.now()}
}
