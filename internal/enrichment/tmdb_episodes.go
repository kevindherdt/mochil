package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// episodeCacheLife is how long a season's episode list is trusted. Episode names and stills rarely change, so it is
// long, and it is kept on disk so a restart does not ask TMDB for every season again.
const episodeCacheLife = 14 * 24 * time.Hour

type storedEpisodeEntry struct {
	Created  time.Time             `json:"created"`
	Episodes map[int]EpisodeDetail `json:"episodes"`
}

// EpisodeDetail is what TMDB knows about one episode: its name, story and a still frame.
type EpisodeDetail struct {
	Number   int    `json:"number"`
	Name     string `json:"name,omitempty"`
	Overview string `json:"overview,omitempty"`
	StillURL string `json:"still_url,omitempty"`
}

type episodeCacheEntry struct {
	episodes map[int]EpisodeDetail
	created  time.Time
}

// SeasonEpisodes lists a season's episodes by number. Results are kept for two weeks; an unknown season is an empty map.
func (t *TMDB) SeasonEpisodes(ctx context.Context, showID string, season int) (map[int]EpisodeDetail, error) {
	if t.token == "" {
		return nil, ErrProviderNotConnected
	}
	if season < 0 || season > 200 || strings.TrimSpace(showID) == "" {
		return nil, fmt.Errorf("TMDB season lookup needs a show id and season")
	}
	if _, err := strconv.ParseInt(showID, 10, 64); err != nil {
		return nil, fmt.Errorf("TMDB show id must be numeric")
	}
	key := showID + "/" + strconv.Itoa(season)
	if episodes, ok := t.cachedEpisodes(key); ok {
		return episodes, nil
	}
	select {
	case t.gate <- struct{}{}:
		defer func() { <-t.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if episodes, ok := t.cachedEpisodes(key); ok {
		return episodes, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.themoviedb.org/3/tv/"+showID+"/season/"+strconv.Itoa(season)+"?language=en-US", nil)
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
	episodes := map[int]EpisodeDetail{}
	switch response.StatusCode {
	case http.StatusOK:
		var payload struct {
			Episodes []struct {
				Number    int    `json:"episode_number"`
				Name      string `json:"name"`
				Overview  string `json:"overview"`
				StillPath string `json:"still_path"`
			} `json:"episodes"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 2<<20)).Decode(&payload); err != nil {
			return nil, fmt.Errorf("decode TMDB season: %w", err)
		}
		for _, item := range payload.Episodes {
			if item.Number <= 0 {
				continue
			}
			detail := EpisodeDetail{Number: item.Number, Name: boundedText(strings.TrimSpace(item.Name), 300), Overview: boundedText(item.Overview, 2000)}
			if strings.HasPrefix(item.StillPath, "/") && !strings.Contains(item.StillPath, "..") {
				detail.StillURL = "https://image.tmdb.org/t/p/w1280" + item.StillPath
			}
			episodes[item.Number] = detail
		}
	case http.StatusNotFound:
		// TMDB has no such season; remember that so the page does not ask again.
	default:
		return nil, fmt.Errorf("TMDB returned HTTP %d", response.StatusCode)
	}
	t.mu.Lock()
	if t.episodes == nil {
		t.episodes = map[string]episodeCacheEntry{}
	}
	if len(t.episodes) >= 500 {
		t.episodes = map[string]episodeCacheEntry{}
	}
	t.episodes[key] = episodeCacheEntry{episodes: episodes, created: t.now()}
	t.saveEpisodesLocked()
	t.mu.Unlock()
	return episodes, nil
}

// PersistEpisodes keeps the season cache in a file and loads what an earlier run saved.
func (t *TMDB) PersistEpisodes(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.episodePath = path
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var stored map[string]storedEpisodeEntry
	if json.Unmarshal(raw, &stored) != nil {
		return
	}
	if t.episodes == nil {
		t.episodes = map[string]episodeCacheEntry{}
	}
	for key, entry := range stored {
		if t.now().Sub(entry.Created) < episodeCacheLife && entry.Episodes != nil {
			t.episodes[key] = episodeCacheEntry{episodes: entry.Episodes, created: entry.Created}
		}
	}
}

func (t *TMDB) saveEpisodesLocked() {
	if t.episodePath == "" {
		return
	}
	stored := make(map[string]storedEpisodeEntry, len(t.episodes))
	for key, entry := range t.episodes {
		stored[key] = storedEpisodeEntry{Created: entry.created, Episodes: entry.episodes}
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(t.episodePath), 0o755) != nil {
		return
	}
	temporary := t.episodePath + ".tmp"
	if os.WriteFile(temporary, raw, 0o600) == nil {
		_ = os.Rename(temporary, t.episodePath)
	}
}

func (t *TMDB) cachedEpisodes(key string) (map[int]EpisodeDetail, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.episodes[key]
	if !ok || t.now().Sub(entry.created) >= episodeCacheLife {
		delete(t.episodes, key)
		return nil, false
	}
	return entry.episodes, true
}
