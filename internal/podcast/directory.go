package podcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SearchResult is a directory entry that can be subscribed to as an RSS feed.
type SearchResult struct {
	Title        string `json:"title"`
	Author       string `json:"author,omitempty"`
	ArtworkURL   string `json:"artwork_url,omitempty"`
	FeedURL      string `json:"feed_url"`
	EpisodeCount int    `json:"episode_count,omitempty"`
	Genre        string `json:"genre,omitempty"`
}

// SearchDirectory uses Apple's public podcast directory. It needs no account
// or API key and returns feed URLs; Mochil still owns subsequent RSS access.
func SearchDirectory(ctx context.Context, client *http.Client, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("a podcast name is required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	if limit < 1 || limit > 40 {
		limit = 20
	}
	params := url.Values{"term": {query}, "media": {"podcast"}, "entity": {"podcast"}, "country": {"us"}, "limit": {fmt.Sprint(limit)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://itunes.apple.com/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mochil/1 podcast directory")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("podcast directory returned %s", res.Status)
	}
	var payload struct {
		Results []struct {
			CollectionName string   `json:"collectionName"`
			TrackName      string   `json:"trackName"`
			ArtistName     string   `json:"artistName"`
			Artwork600     string   `json:"artworkUrl600"`
			Artwork100     string   `json:"artworkUrl100"`
			FeedURL        string   `json:"feedUrl"`
			TrackCount     int      `json:"trackCount"`
			Genres         []string `json:"genres"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(payload.Results))
	for _, item := range payload.Results {
		if strings.TrimSpace(item.FeedURL) == "" {
			continue
		}
		genre := ""
		if len(item.Genres) > 0 {
			genre = item.Genres[0]
		}
		title := strings.TrimSpace(item.CollectionName)
		if title == "" {
			title = strings.TrimSpace(item.TrackName)
		}
		artwork := item.Artwork600
		if artwork == "" {
			artwork = item.Artwork100
		}
		out = append(out, SearchResult{Title: title, Author: strings.TrimSpace(item.ArtistName), ArtworkURL: artwork, FeedURL: item.FeedURL, EpisodeCount: item.TrackCount, Genre: genre})
	}
	return out, nil
}
