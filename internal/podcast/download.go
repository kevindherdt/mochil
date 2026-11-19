package podcast

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"mochil/internal/jobs"
)

type Download struct {
	EpisodeID string `json:"episode_id"`
	Status    string `json:"status"`
	Bytes     int64  `json:"bytes"`
	Error     string `json:"error,omitempty"`
}
type DownloadStats struct {
	Bytes    int64 `json:"bytes"`
	MaxBytes int64 `json:"max_bytes"`
	Files    int   `json:"files"`
	Failed   int   `json:"failed,omitempty"`
}
type Downloader struct {
	root                      string
	client                    *http.Client
	maxBytes, maxEpisodeBytes int64
	pool                      *jobs.Pool
	mu                        sync.RWMutex
	state                     map[string]Download
	deleted                   map[string]bool
}

// PlaybackLookup reports when an episode was last marked played. It keeps
// account-owned playback state outside the shared podcast download package.
type PlaybackLookup func(episodeID string) (time.Time, bool)

// NewDownloader bounds completed downloads by maxBytes. An episode larger
// than that limit fails without evicting existing downloads; zero or negative
// maxBytes selects the default limit. Existing over-limit caches are left
// untouched at startup and trimmed oldest-first on the next successful
// admission. Failed removals may leave the cache over its configured limit.
func NewDownloader(root string, client *http.Client, maxBytes int64) *Downloader {
	if client == nil {
		client = NewPublicHTTPClient(0)
	}
	if maxBytes <= 0 {
		maxBytes = 20 << 30
	}
	_ = os.MkdirAll(root, 0o750)
	pool, err := jobs.NewPersistent(2, 256, filepath.Join(root, "jobs.json"))
	if err != nil {
		pool = jobs.New(2, 256)
	}
	state := make(map[string]Download)
	for _, job := range pool.Snapshot() {
		if job.Status == "failed" {
			state[job.Key] = Download{EpisodeID: job.Key, Status: "failed", Bytes: job.ProgressCurrent, Error: job.Error}
		}
	}
	return &Downloader{root: root, client: client, maxBytes: maxBytes, maxEpisodeBytes: 2 << 30, pool: pool, state: state, deleted: map[string]bool{}}
}
func (d *Downloader) Path(id string) (string, bool) {
	path := filepath.Join(d.root, id+".media")
	info, err := os.Stat(path)
	return path, err == nil && info.Mode().IsRegular()
}
func (d *Downloader) Status(id string) Download {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if _, ok := d.Path(id); ok {
		return Download{EpisodeID: id, Status: "ready"}
	}
	return d.state[id]
}

// Start returns the current download state, preserving the existing caller contract.
func (d *Downloader) Start(episode Episode) Download {
	state, _ := d.StartWithError(episode)
	return state
}

// StartWithError reports admission failures separately from download state.
func (d *Downloader) StartWithError(episode Episode) (Download, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.Path(episode.ID); ok {
		return Download{EpisodeID: episode.ID, Status: "ready"}, nil
	}
	if state := d.state[episode.ID]; state.Status == "queued" || state.Status == "downloading" {
		return state, nil
	}
	previous := d.state[episode.ID]
	state := Download{EpisodeID: episode.ID, Status: "queued"}
	d.state[episode.ID] = state
	if err := d.pool.Submit(episode.ID, func(ctx context.Context) error { return d.download(ctx, episode) }); err != nil {
		message := "The server could not start this download. Try again later."
		switch {
		case errors.Is(err, jobs.ErrDuplicate):
			message = "The previous download is still stopping. Please try again shortly."
		case errors.Is(err, jobs.ErrQueueFull):
			message = "The download queue is full. Please try again shortly."
		case errors.Is(err, jobs.ErrClosed):
			message = "Server downloads are unavailable. Please try again later."
		}
		state = Download{EpisodeID: episode.ID, Status: "failed", Error: message}
		if errors.Is(err, jobs.ErrDuplicate) && previous.Status == "failed" && !d.deleted[episode.ID] {
			// The failed worker can still own its pool key during cleanup.
			// Report this admission error without losing the original failure.
			d.state[episode.ID] = previous
		} else {
			d.state[episode.ID] = state
		}
		return state, err
	}
	delete(d.deleted, episode.ID)
	return state, nil
}
func (d *Downloader) Delete(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pool.Cancel(id)
	d.deleted[id] = true
	delete(d.state, id)
	err := os.Remove(filepath.Join(d.root, id+".media"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func (d *Downloader) Close() { d.pool.Close() }

// Jobs returns a point-in-time copy of the bounded download queue. It is
// intentionally separate from download state so operational diagnostics do not
// become part of the podcast catalogue contract.
func (d *Downloader) Jobs() []jobs.State { return d.pool.Snapshot() }

func (d *Downloader) Stats() DownloadStats {
	stats := DownloadStats{MaxBytes: d.maxBytes}
	d.mu.RLock()
	for _, state := range d.state {
		if state.Status == "failed" {
			stats.Failed++
		}
	}
	d.mu.RUnlock()
	entries, _ := os.ReadDir(d.root)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".media" {
			continue
		}
		if info, err := entry.Info(); err == nil {
			stats.Bytes += info.Size()
			stats.Files++
		}
	}
	return stats
}

// RetryFailed re-admits only failed episodes that still exist in the
// subscription snapshot. The existing bounded pool remains the admission
// authority.
func (d *Downloader) RetryFailed(episodes []Episode) int {
	failed := make(map[string]bool)
	d.mu.RLock()
	for id, state := range d.state {
		failed[id] = state.Status == "failed"
	}
	d.mu.RUnlock()
	retried := 0
	for _, episode := range episodes {
		if failed[episode.ID] {
			state := d.Start(episode)
			if state.Status == "queued" || state.Status == "downloading" {
				retried++
			}
		}
	}
	return retried
}

// Reconcile applies one feed's opt-in policy. It never removes feed metadata;
// it only bounds the local download cache for episodes from this feed.
func (d *Downloader) Reconcile(feed Feed, playedAt PlaybackLookup) {
	episodes := append([]Episode(nil), feed.Episodes...)
	sort.SliceStable(episodes, func(i, j int) bool { return episodes[i].Published.After(episodes[j].Published) })
	keep := feed.Policy.RetainDownloads
	if keep > len(episodes) {
		keep = len(episodes)
	}
	remove := make(map[string]bool)
	if feed.Policy.RetainDownloads > 0 {
		for _, episode := range episodes[keep:] {
			remove[episode.ID] = true
		}
	}
	if feed.Policy.DeletePlayedAfterDays > 0 && playedAt != nil {
		cutoff := time.Now().AddDate(0, 0, -feed.Policy.DeletePlayedAfterDays)
		for _, episode := range episodes {
			if played, ok := playedAt(episode.ID); ok && !played.After(cutoff) {
				remove[episode.ID] = true
			}
		}
	}
	for _, episode := range episodes {
		if remove[episode.ID] {
			_ = d.Delete(episode.ID)
		}
	}
	if feed.Policy.AutoDownload {
		for _, episode := range episodes[:keep] {
			if !remove[episode.ID] {
				d.Start(episode)
			}
		}
	}
}
func (d *Downloader) download(parent context.Context, episode Episode) error {
	d.set(Download{EpisodeID: episode.ID, Status: "downloading"})
	ctx, cancel := context.WithTimeout(parent, 6*time.Hour)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, episode.MediaURL, nil)
	if err != nil {
		d.fail(episode.ID, err)
		return err
	}
	response, err := d.client.Do(request)
	if err != nil {
		d.fail(episode.ID, err)
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		d.fail(episode.ID, errors.New(response.Status))
		return errors.New(response.Status)
	}
	if response.ContentLength > d.maxEpisodeBytes {
		d.fail(episode.ID, errors.New("episode exceeds 2 GiB"))
		return errors.New("episode exceeds 2 GiB")
	}
	if response.ContentLength > d.maxBytes {
		err := errors.New("episode exceeds download cache limit")
		d.fail(episode.ID, err)
		return err
	}
	temporary, err := os.CreateTemp(d.root, ".podcast-*")
	if err != nil {
		d.fail(episode.ID, err)
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	total := response.ContentLength
	if total < 0 {
		total = 0
	}
	progress := &downloadProgressWriter{writer: temporary, ctx: ctx, total: total}
	limit := min(d.maxEpisodeBytes, d.maxBytes)
	written, err := io.Copy(progress, io.LimitReader(response.Body, limit+1))
	if err == nil && written > d.maxEpisodeBytes {
		err = errors.New("episode exceeds 2 GiB")
	} else if err == nil && written > d.maxBytes {
		err = errors.New("episode exceeds download cache limit")
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		d.fail(episode.ID, err)
		return err
	}
	// Serialize publication with Delete; cancellation alone cannot stop a body
	// whose final Read succeeds after its context has been canceled.
	d.mu.Lock()
	if err = ctx.Err(); err == nil && d.deleted[episode.ID] {
		err = context.Canceled
	}
	if err == nil {
		err = os.Rename(name, filepath.Join(d.root, episode.ID+".media"))
	}
	if err == nil {
		d.state[episode.ID] = Download{EpisodeID: episode.ID, Status: "ready", Bytes: written}
		d.evict(episode.ID)
	}
	d.mu.Unlock()
	if err != nil {
		d.fail(episode.ID, err)
		return err
	}
	return nil
}

type downloadProgressWriter struct {
	writer                   io.Writer
	ctx                      context.Context
	total, written, reported int64
}

func (w *downloadProgressWriter) Write(value []byte) (int, error) {
	n, err := w.writer.Write(value)
	w.written += int64(n)
	if w.written-w.reported >= 256<<10 || (w.total > 0 && w.written == w.total) {
		jobs.Report(w.ctx, w.written, w.total, "Downloading episode")
		w.reported = w.written
	}
	return n, err
}
func (d *Downloader) set(value Download) {
	d.mu.Lock()
	if d.deleted[value.EpisodeID] {
		d.mu.Unlock()
		return
	}
	d.state[value.EpisodeID] = value
	d.mu.Unlock()
}
func (d *Downloader) fail(id string, err error) {
	d.set(Download{EpisodeID: id, Status: "failed", Error: err.Error()})
}

// evict runs under d.mu, serializing cache admission with deletion and other
// completions so one completion cannot evict another's newly published file.
func (d *Downloader) evict(keep string) {
	entries, _ := os.ReadDir(d.root)
	type file struct {
		path     string
		size     int64
		modified time.Time
	}
	files := []file{}
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".media" {
			continue
		}
		info, e := entry.Info()
		if e != nil {
			continue
		}
		files = append(files, file{filepath.Join(d.root, entry.Name()), info.Size(), info.ModTime()})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modified.Before(files[j].modified) })
	for _, file := range files {
		if total <= d.maxBytes {
			break
		}
		if filepath.Base(file.path) == keep+".media" {
			continue
		}
		if os.Remove(file.path) == nil {
			total -= file.size
			id := filepath.Base(file.path)
			id = id[:len(id)-len(".media")]
			if d.state[id].Status == "ready" {
				delete(d.state, id)
			}
		}
	}
}
