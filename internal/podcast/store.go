// Package podcast owns RSS subscriptions and episode state independently from
// local audio libraries and music queues.
package podcast

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const maximumStateBytes int64 = 128 << 20

type Episode struct {
	ID          string    `json:"id"`
	FeedID      string    `json:"feed_id"`
	FeedTitle   string    `json:"feed_title"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	MediaURL    string    `json:"media_url,omitempty"`
	Link        string    `json:"link,omitempty"`
	Published   time.Time `json:"published,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	MediaType   string    `json:"media_type,omitempty"`
	Size        int64     `json:"size,omitempty"`
	ArtworkURL  string    `json:"artwork_url,omitempty"`
	Downloaded  bool      `json:"downloaded,omitempty"`
}

type FeedPolicy struct {
	AutoDownload          bool `json:"auto_download,omitempty"`
	RetainDownloads       int  `json:"retain_downloads,omitempty"`
	DeletePlayedAfterDays int  `json:"delete_played_after_days,omitempty"`
}

type Feed struct {
	ID           string     `json:"id"`
	URL          string     `json:"url"`
	Title        string     `json:"title"`
	Description  string     `json:"description,omitempty"`
	ArtworkURL   string     `json:"artwork_url,omitempty"`
	ETag         string     `json:"-"`
	LastModified string     `json:"-"`
	Updated      time.Time  `json:"updated"`
	LastError    string     `json:"last_error,omitempty"`
	Policy       FeedPolicy `json:"policy,omitempty"`
	Episodes     []Episode  `json:"episodes"`
}
type dataFile struct {
	Version int    `json:"version"`
	Feeds   []Feed `json:"feeds"`
}
type Store struct {
	path       string
	client     *http.Client
	publicOnly bool
	mu         sync.RWMutex
	feeds      []Feed
	refreshing bool
	version    uint64
}

// RefreshStale conditionally refreshes old feeds while callers continue using
// the last good snapshot. At most one background refresh runs per process.
func (s *Store) RefreshStale(ctx context.Context, age time.Duration) {
	s.mu.Lock()
	if s.refreshing {
		s.mu.Unlock()
		return
	}
	ids := []string{}
	now := time.Now()
	for _, feed := range s.feeds {
		// Older state files did not persist enclosure URLs. Refresh those feeds
		// immediately after restart even when their timestamp is still fresh;
		// otherwise the catalogue is visible but its episodes cannot be played.
		missingMedia := false
		for _, episode := range feed.Episodes {
			if strings.TrimSpace(episode.MediaURL) == "" {
				missingMedia = true
				break
			}
		}
		if age <= 0 || missingMedia || now.Sub(feed.Updated) >= age {
			ids = append(ids, feed.ID)
		}
	}
	if len(ids) == 0 {
		s.mu.Unlock()
		return
	}
	s.refreshing = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.refreshing = false; s.mu.Unlock() }()
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		_, _ = s.Refresh(ctx, id)
	}
}

func Open(path string, client *http.Client) (*Store, error) {
	publicOnly := client == nil
	if client == nil {
		client = NewPublicHTTPClient(20 * time.Second)
	}
	s := &Store{path: path, client: client, publicOnly: publicOnly, version: 1}
	if path == "" {
		return s, nil
	}
	var data dataFile
	err := statefile.DecodeJSON(path, maximumStateBytes, &data)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if data.Version != 1 {
		return nil, errors.New("unsupported podcast store version")
	}
	s.feeds = data.Feeds
	for feedIndex := range s.feeds {
		s.feeds[feedIndex].Description = podcastPlainText(s.feeds[feedIndex].Description)
		for episodeIndex := range s.feeds[feedIndex].Episodes {
			s.feeds[feedIndex].Episodes[episodeIndex].Description = podcastPlainText(s.feeds[feedIndex].Episodes[episodeIndex].Description)
		}
	}
	return s, nil
}
func (s *Store) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}
func (s *Store) Feeds() []Feed {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Feed, len(s.feeds))
	for index := range s.feeds {
		out[index] = cloneFeed(s.feeds[index])
	}
	return out
}
func (s *Store) Feed(id string) (Feed, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, feed := range s.feeds {
		if feed.ID == id {
			return cloneFeed(feed), true
		}
	}
	return Feed{}, false
}
func (s *Store) Episodes() []Episode {
	s.mu.RLock()
	out := []Episode{}
	for _, feed := range s.feeds {
		out = append(out, feed.Episodes...)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Published.After(out[j].Published) })
	return out
}
func (s *Store) EpisodesSnapshot() ([]Episode, uint64) {
	s.mu.RLock()
	out := []Episode{}
	for _, feed := range s.feeds {
		out = append(out, feed.Episodes...)
	}
	version := s.version
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Published.After(out[j].Published) })
	return out, version
}
func (s *Store) Episode(id string) (Episode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, feed := range s.feeds {
		for _, episode := range feed.Episodes {
			if episode.ID == id {
				return episode, true
			}
		}
	}
	return Episode{}, false
}

func (s *Store) Subscribe(ctx context.Context, rawURL string) (Feed, error) {
	parsed, err := validatePodcastURL(rawURL, s.publicOnly)
	if err != nil {
		return Feed{}, err
	}
	idSum := sha256.Sum256([]byte(parsed.String()))
	id := hex.EncodeToString(idSum[:12])
	s.mu.RLock()
	for _, feed := range s.feeds {
		if feed.ID == id {
			s.mu.RUnlock()
			return cloneFeed(feed), nil
		}
	}
	s.mu.RUnlock()
	feed := Feed{ID: id, URL: parsed.String()}
	feed, err = s.fetch(ctx, feed)
	if err != nil {
		return Feed{}, err
	}
	s.mu.Lock()
	for _, existing := range s.feeds {
		if existing.ID == id { // a second subscribe of the same address finished first
			s.mu.Unlock()
			return cloneFeed(existing), nil
		}
	}
	s.feeds = append(s.feeds, feed)
	err = s.saveLocked()
	if err != nil {
		// Keeping a feed that never reached the disk made the retry answer
		// "already subscribed" and it vanished on restart: undo the append.
		s.feeds = s.feeds[:len(s.feeds)-1]
	}
	if err == nil {
		s.version++
	}
	s.mu.Unlock()
	return cloneFeed(feed), err
}
func (s *Store) Refresh(ctx context.Context, id string) (Feed, error) {
	current, ok := s.Feed(id)
	if !ok {
		return Feed{}, os.ErrNotExist
	}
	fresh, err := s.fetch(ctx, current)
	s.mu.Lock()
	defer s.mu.Unlock()
	// The fetch takes seconds; the feed may have been removed or changed meanwhile, so find it again
	// and keep the policy as it is now rather than as it was when the fetch began.
	index := -1
	for i := range s.feeds {
		if s.feeds[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return Feed{}, os.ErrNotExist
	}
	if err != nil {
		current = cloneFeed(s.feeds[index])
		current.LastError = err.Error()
		current.Updated = time.Now().UTC()
		s.feeds[index] = current
		if saveErr := s.saveLocked(); saveErr == nil {
			s.version++
		}
		return cloneFeed(current), err
	}
	fresh.Policy = s.feeds[index].Policy
	previous := s.feeds[index]
	s.feeds[index] = fresh
	if err = s.saveLocked(); err != nil {
		s.feeds[index] = previous
		return cloneFeed(previous), err
	}
	s.version++
	return cloneFeed(fresh), nil
}
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, feed := range s.feeds {
		if feed.ID == id {
			s.feeds = append(s.feeds[:i], s.feeds[i+1:]...)
			if err := s.saveLocked(); err != nil {
				s.feeds = append(s.feeds, Feed{})
				copy(s.feeds[i+1:], s.feeds[i:])
				s.feeds[i] = feed
				return err
			}
			s.version++
			return nil
		}
	}
	return os.ErrNotExist
}

func (s *Store) SetPolicy(id string, policy FeedPolicy) (Feed, error) {
	if policy.RetainDownloads < 0 || policy.RetainDownloads > 50 {
		return Feed{}, errors.New("retained downloads must be between 0 and 50")
	}
	if policy.AutoDownload && policy.RetainDownloads == 0 {
		return Feed{}, errors.New("automatic downloads require a retention count")
	}
	if policy.DeletePlayedAfterDays < 0 || policy.DeletePlayedAfterDays > 3650 {
		return Feed{}, errors.New("played download retention must be between 0 and 3650 days")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.feeds {
		if s.feeds[i].ID != id {
			continue
		}
		previous := s.feeds[i].Policy
		s.feeds[i].Policy = policy
		if err := s.saveLocked(); err != nil {
			s.feeds[i].Policy = previous
			return Feed{}, err
		}
		s.version++
		return cloneFeed(s.feeds[i]), nil
	}
	return Feed{}, os.ErrNotExist
}

func cloneFeed(feed Feed) Feed {
	feed.Episodes = append([]Episode(nil), feed.Episodes...)
	return feed
}

func (s *Store) fetch(ctx context.Context, current Feed) (Feed, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, current.URL, nil)
	if err != nil {
		return current, err
	}
	request.Header.Set("User-Agent", "Mochil/1 podcast refresh")
	if current.ETag != "" {
		request.Header.Set("If-None-Match", current.ETag)
	}
	if current.LastModified != "" {
		request.Header.Set("If-Modified-Since", current.LastModified)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return current, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotModified {
		current.Updated = time.Now().UTC()
		current.LastError = ""
		return current, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return current, fmt.Errorf("feed returned %s", response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (10<<20)+1))
	if err != nil {
		return current, err
	}
	if len(body) > 10<<20 {
		return current, errors.New("podcast feed exceeds 10 MiB")
	}
	parsed, err := parseRSS(body, current.ID)
	if err != nil {
		return current, err
	}
	parsed.ID, parsed.URL, parsed.ETag, parsed.LastModified, parsed.Policy = current.ID, current.URL, response.Header.Get("ETag"), response.Header.Get("Last-Modified"), current.Policy
	parsed.Updated = time.Now().UTC()
	return parsed, nil
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".podcasts-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = json.NewEncoder(temporary).Encode(dataFile{1, s.feeds}); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
