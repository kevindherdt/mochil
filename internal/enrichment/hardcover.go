package enrichment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const hardcoverEndpoint = "https://api.hardcover.app/v1/graphql"

type Hardcover struct {
	token  string
	client *http.Client
	now    func() time.Time
	gate   chan struct{}
	mu     sync.Mutex
	cache  map[string]cacheEntry
}

func NewHardcover(token string, client *http.Client) *Hardcover {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Hardcover{token: strings.TrimSpace(token), client: client, now: time.Now, gate: make(chan struct{}, 1), cache: map[string]cacheEntry{}}
}

func (h *Hardcover) Check(ctx context.Context) error {
	if h.token == "" {
		return ErrProviderNotConnected
	}
	var payload struct {
		Me []struct {
			ID int64 `json:"id"`
		} `json:"me"`
	}
	if err := h.graphql(ctx, `query ConnectionStatus { me(limit: 1) { id } }`, nil, &payload); err != nil {
		if errors.Is(err, ErrProviderRejected) || errors.Is(err, ErrProviderRateLimited) {
			return err
		}
		return ErrProviderUnavailable
	}
	if len(payload.Me) != 1 || payload.Me[0].ID <= 0 {
		return ErrProviderRejected
	}
	return nil
}

func (h *Hardcover) Search(ctx context.Context, query Query) ([]Suggestion, error) {
	title, author := strings.TrimSpace(query.Title), strings.TrimSpace(query.Author)
	if h.token == "" {
		return nil, errors.New("Hardcover is not connected")
	}
	if title == "" || len(title) > 500 || len(author) > 500 {
		return nil, errors.New("Hardcover lookup requires a bounded title")
	}
	searchText := title
	if author != "" {
		searchText += " " + author
	}
	key := strings.ToLower(searchText)
	if results, ok := h.cached(key); ok {
		return results, nil
	}
	select {
	case h.gate <- struct{}{}:
		defer func() { <-h.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if results, ok := h.cached(key); ok {
		return results, nil
	}
	var searchPayload struct {
		Search struct {
			IDs []int64 `json:"ids"`
		} `json:"search"`
	}
	if err := h.graphql(ctx, `query SearchBooks($query: String!) { search(query: $query, query_type: "Book", per_page: 5, page: 1) { ids } }`, map[string]any{"query": searchText}, &searchPayload); err != nil {
		return nil, err
	}
	ids := searchPayload.Search.IDs
	if len(ids) > maximumResults {
		ids = ids[:maximumResults]
	}
	if len(ids) == 0 {
		h.store(key, nil)
		return nil, nil
	}
	var details struct {
		Books []struct {
			ID            int64  `json:"id"`
			Title         string `json:"title"`
			Description   string `json:"description"`
			ReleaseYear   int    `json:"release_year"`
			Slug          string `json:"slug"`
			CachedImage   any    `json:"cached_image"`
			Contributions []struct {
				Author struct {
					Name string `json:"name"`
				} `json:"author"`
			} `json:"contributions"`
		} `json:"books"`
	}
	if err := h.graphql(ctx, `query BookDetails($ids: [Int!]!) { books(where: {id: {_in: $ids}}) { id title description release_year slug cached_image contributions { author { name } } } }`, map[string]any{"ids": ids}, &details); err != nil {
		return nil, err
	}
	order := make(map[int64]int, len(ids))
	for index, id := range ids {
		order[id] = index
	}
	sort.SliceStable(details.Books, func(i, j int) bool { return order[details.Books[i].ID] < order[details.Books[j].ID] })
	results := make([]Suggestion, 0, len(details.Books))
	for _, book := range details.Books {
		if book.ID <= 0 || strings.TrimSpace(book.Title) == "" {
			continue
		}
		authors := make([]string, 0, len(book.Contributions))
		for _, contribution := range book.Contributions {
			if name := strings.TrimSpace(contribution.Author.Name); name != "" {
				authors = append(authors, name)
			}
		}
		id := strconv.FormatInt(book.ID, 10)
		slug := safeSlug(book.Slug)
		sourceURL := "https://hardcover.app/books/" + slug
		if slug == "" {
			sourceURL = "https://hardcover.app/search?query=" + url.QueryEscape(book.Title)
		}
		results = append(results, Suggestion{Title: book.Title, Authors: authors, Year: book.ReleaseYear, Overview: boundedText(book.Description, 2000), CoverURL: hardcoverImageURL(book.CachedImage), Provider: "Hardcover", ProviderID: id, SourceURL: sourceURL})
	}
	h.store(key, results)
	return append([]Suggestion(nil), results...), nil
}

func (h *Hardcover) graphql(ctx context.Context, query string, variables map[string]any, target any) error {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, hardcoverEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+h.token)
	response, err := h.client.Do(request)
	if err != nil {
		return fmt.Errorf("Hardcover request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return ErrProviderRejected
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return ErrProviderRateLimited
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Hardcover returned HTTP %d", response.StatusCode)
	}
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 1<<20)).Decode(&envelope); err != nil {
		return fmt.Errorf("decode Hardcover response: %w", err)
	}
	if len(envelope.Errors) != 0 {
		return ErrProviderRejected
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return errors.New("Hardcover returned no data")
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		return fmt.Errorf("decode Hardcover data: %w", err)
	}
	return nil
}

func (h *Hardcover) cached(key string) ([]Suggestion, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry, ok := h.cache[key]
	if !ok || h.now().Sub(entry.created) >= 24*time.Hour {
		delete(h.cache, key)
		return nil, false
	}
	return append([]Suggestion(nil), entry.results...), true
}
func (h *Hardcover) store(key string, results []Suggestion) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.cache) >= maximumCache {
		var oldestKey string
		var oldest time.Time
		for candidate, entry := range h.cache {
			if oldest.IsZero() || entry.created.Before(oldest) {
				oldestKey, oldest = candidate, entry.created
			}
		}
		delete(h.cache, oldestKey)
	}
	h.cache[key] = cacheEntry{results: append([]Suggestion(nil), results...), created: h.now()}
}

func safeSlug(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	for _, part := range strings.Split(value, "-") {
		if part == "" {
			continue
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
				return ""
			}
		}
	}
	return value
}
func hardcoverImageURL(value any) string {
	var walk func(any) string
	walk = func(candidate any) string {
		switch typed := candidate.(type) {
		case map[string]any:
			if raw, ok := typed["url"].(string); ok && strings.HasPrefix(raw, "https://") {
				return raw
			}
			for _, nested := range typed {
				if found := walk(nested); found != "" {
					return found
				}
			}
		case []any:
			for _, nested := range typed {
				if found := walk(nested); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return walk(value)
}
