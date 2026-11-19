package podcast

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mochil/internal/jobs"
)

// eofGateBody deliberately ignores request cancellation. It exposes the point
// after the final bytes have been copied but before io.Copy returns EOF.
type eofGateBody struct {
	read    bool
	atEOF   chan struct{}
	release chan struct{}
}

func (b *eofGateBody) Read(p []byte) (int, error) {
	if !b.read {
		b.read = true
		return copy(p, "episode"), nil
	}
	close(b.atEOF)
	<-b.release
	return 0, io.EOF
}

func (b *eofGateBody) Close() error { return nil }

type downloadRoundTripper func(*http.Request) (*http.Response, error)

func (f downloadRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDownloaderDeleteDuringFinalCopyDoesNotRepublish(t *testing.T) {
	body := &eofGateBody{atEOF: make(chan struct{}), release: make(chan struct{})}
	client := &http.Client{Transport: downloadRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: -1, Header: make(http.Header)}, nil
	})}
	d := NewDownloader(t.TempDir(), client, 1024)
	defer d.Close()
	d.Start(Episode{ID: "episode", MediaURL: "http://example.test/episode"})
	select {
	case <-body.atEOF:
	case <-time.After(5 * time.Second):
		close(body.release)
		t.Fatal("download did not reach EOF")
	}
	if err := d.Delete("episode"); err != nil {
		close(body.release)
		t.Fatal(err)
	}
	close(body.release)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = d.pool.Wait(ctx, "episode")
	if ctx.Err() != nil {
		t.Fatal("download did not finish")
	}
	if _, ok := d.Path("episode"); ok {
		t.Fatal("deleted episode was republished after copy completed")
	}
	if state := d.Status("episode"); state.Status != "" {
		t.Fatalf("deleted episode state = %+v", state)
	}
}

func TestDownloaderRejectsOversizedEpisodeWithoutEvictingExistingCache(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("123456789"))
	}))
	defer server.Close()
	d := NewDownloader(root, server.Client(), 8)
	defer d.Close()
	if err := os.WriteFile(filepath.Join(root, "existing.media"), []byte("saved"), 0o600); err != nil {
		t.Fatal(err)
	}
	d.Start(Episode{ID: "oversized", MediaURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.pool.Wait(ctx, "oversized"); err == nil {
		t.Fatal("oversized download unexpectedly succeeded")
	}
	if ctx.Err() != nil {
		t.Fatal("download did not finish")
	}
	if _, ok := d.Path("oversized"); ok {
		t.Fatal("oversized episode was cached")
	}
	if _, ok := d.Path("existing"); !ok {
		t.Fatal("existing download was evicted by an episode that cannot fit")
	}
	if state := d.Status("oversized"); state.Status != "failed" || state.Error == "" {
		t.Fatalf("oversized state = %+v", state)
	}
	if stats := d.Stats(); stats.Bytes != 5 || stats.Files != 1 {
		t.Fatalf("cache stats = %+v", stats)
	}
}

func TestDownloaderDeleteThenImmediateRestartKeepsDeletedState(t *testing.T) {
	body := &eofGateBody{atEOF: make(chan struct{}), release: make(chan struct{})}
	client := &http.Client{Transport: downloadRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: -1, Header: make(http.Header)}, nil
	})}
	d := NewDownloader(t.TempDir(), client, 1024)
	defer d.Close()
	episode := Episode{ID: "episode", MediaURL: "http://example.test/episode"}
	d.Start(episode)
	select {
	case <-body.atEOF:
	case <-time.After(5 * time.Second):
		close(body.release)
		t.Fatal("download did not reach EOF")
	}
	if err := d.Delete(episode.ID); err != nil {
		close(body.release)
		t.Fatal(err)
	}
	state := d.Start(episode)
	if state.Status != "failed" {
		close(body.release)
		t.Fatalf("immediate restart status = %+v, want a retryable admission failure", state)
	}
	close(body.release)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = d.pool.Wait(ctx, episode.ID)
	if ctx.Err() != nil {
		t.Fatal("download did not finish")
	}
	if current := d.Status(episode.ID); current.Status != "failed" || current.Error != state.Error || !strings.Contains(strings.ToLower(current.Error), "try again") {
		t.Fatalf("canceled earlier job overwrote retryable restart result: before=%+v after=%+v", state, current)
	}
	if _, ok := d.Path(episode.ID); ok {
		t.Fatal("canceled earlier job republished episode")
	}
}

func TestDownloaderEvictionClearsReadyState(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("123456"))
	}))
	defer server.Close()
	d := NewDownloader(root, server.Client(), 8)
	defer d.Close()
	for _, id := range []string{"first", "second"} {
		d.Start(Episode{ID: id, MediaURL: server.URL})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := d.pool.Wait(ctx, id)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := d.Path("first"); ok {
		t.Fatal("older episode was not evicted")
	}
	if state := d.Status("first"); state.Status == "ready" {
		t.Fatalf("evicted episode still reports ready: %+v", state)
	}
	if stats := d.Stats(); stats.Bytes != 6 || stats.Files != 1 {
		t.Fatalf("cache stats = %+v", stats)
	}
}

func TestDownloaderRejectsUnknownLengthOversizedEpisodeWithoutEviction(t *testing.T) {
	root := t.TempDir()
	client := &http.Client{Transport: downloadRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("123456789")), ContentLength: -1, Header: make(http.Header)}, nil
	})}
	d := NewDownloader(root, client, 8)
	defer d.Close()
	if err := os.WriteFile(filepath.Join(root, "existing.media"), []byte("saved"), 0o600); err != nil {
		t.Fatal(err)
	}
	d.Start(Episode{ID: "oversized", MediaURL: "http://example.test/episode"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.pool.Wait(ctx, "oversized"); err == nil || !strings.Contains(err.Error(), "cache limit") {
		t.Fatalf("oversized download result = %v", err)
	}
	if _, ok := d.Path("oversized"); ok {
		t.Fatal("oversized episode was cached")
	}
	if _, ok := d.Path("existing"); !ok {
		t.Fatal("existing download was evicted")
	}
	if entries, err := os.ReadDir(root); err != nil {
		t.Fatal(err)
	} else {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".podcast-") {
				t.Fatalf("temporary download left behind: %s", entry.Name())
			}
		}
	}
}

func TestDownloaderAcceptsEpisodeExactlyAtCacheLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("12345678"))
	}))
	defer server.Close()
	d := NewDownloader(t.TempDir(), server.Client(), 8)
	defer d.Close()
	d.Start(Episode{ID: "exact", MediaURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.pool.Wait(ctx, "exact"); err != nil {
		t.Fatal(err)
	}
	if stats := d.Stats(); stats.Bytes != 8 || stats.Files != 1 {
		t.Fatalf("cache stats = %+v", stats)
	}
	if state := d.Status("exact"); state.Status != "ready" {
		t.Fatalf("exact-limit episode = %+v", state)
	}
}

func TestDownloaderConcurrentCompletionsRespectCacheLimit(t *testing.T) {
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		arrived <- struct{}{}
		<-release
		_, _ = w.Write([]byte("123456"))
	}))
	defer server.Close()
	d := NewDownloader(t.TempDir(), server.Client(), 8)
	defer d.Close()
	for _, id := range []string{"first", "second"} {
		d.Start(Episode{ID: id, MediaURL: server.URL})
	}
	for range 2 {
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			close(release)
			t.Fatal("downloads did not overlap")
		}
	}
	close(release)
	for _, id := range []string{"first", "second"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := d.pool.Wait(ctx, id)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
	}
	if stats := d.Stats(); stats.Bytes != 6 || stats.Files != 1 {
		t.Fatalf("concurrent cache stats = %+v", stats)
	}
	for _, id := range []string{"first", "second"} {
		_, exists := d.Path(id)
		if ready := d.Status(id).Status == "ready"; ready != exists {
			t.Fatalf("%s: ready=%v exists=%v", id, ready, exists)
		}
	}
}

func TestDownloaderSuccessfulRetryAfterCanceledJobExits(t *testing.T) {
	body := &eofGateBody{atEOF: make(chan struct{}), release: make(chan struct{})}
	var requests atomic.Int32
	client := &http.Client{Transport: downloadRoundTripper(func(*http.Request) (*http.Response, error) {
		var responseBody io.ReadCloser = body
		if requests.Add(1) > 1 {
			responseBody = io.NopCloser(strings.NewReader("new"))
		}
		return &http.Response{StatusCode: http.StatusOK, Body: responseBody, ContentLength: -1, Header: make(http.Header)}, nil
	})}
	d := NewDownloader(t.TempDir(), client, 1024)
	defer d.Close()
	episode := Episode{ID: "episode", MediaURL: "http://example.test/episode"}
	d.Start(episode)
	select {
	case <-body.atEOF:
	case <-time.After(5 * time.Second):
		close(body.release)
		t.Fatal("download did not reach EOF")
	}
	if err := d.Delete(episode.ID); err != nil {
		close(body.release)
		t.Fatal(err)
	}
	close(body.release)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.pool.Wait(ctx, episode.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("old download result = %v", err)
	}
	if state := d.Start(episode); state.Status != "queued" {
		t.Fatalf("retry admission = %+v", state)
	}
	if err := d.pool.Wait(ctx, episode.ID); err != nil {
		t.Fatal(err)
	}
	path, ok := d.Path(episode.ID)
	if !ok {
		t.Fatal("retry did not publish")
	}
	value, err := os.ReadFile(path)
	if err != nil || string(value) != "new" {
		t.Fatalf("retry content = %q, %v", value, err)
	}
}

func TestDownloaderTrimsExistingOverLimitCacheOnAdmission(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"older", "newer"} {
		if err := os.WriteFile(filepath.Join(root, id+".media"), []byte("123456"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(filepath.Join(root, "older.media"), time.Unix(1, 0), time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(root, "newer.media"), time.Unix(2, 0), time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("1234"))
	}))
	defer server.Close()
	d := NewDownloader(root, server.Client(), 8)
	defer d.Close()
	if stats := d.Stats(); stats.Bytes != 12 || stats.Files != 2 {
		t.Fatalf("startup changed existing cache: %+v", stats)
	}
	d.Start(Episode{ID: "new", MediaURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.pool.Wait(ctx, "new"); err != nil {
		t.Fatal(err)
	}
	if stats := d.Stats(); stats.Bytes != 4 || stats.Files != 1 {
		t.Fatalf("post-admission cache = %+v", stats)
	}
	if _, ok := d.Path("new"); !ok {
		t.Fatal("newly admitted episode missing")
	}
}

func TestDownloaderRejectsOversizedEpisodeWithoutTouchingExistingOverLimitCache(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "existing.media"), []byte("123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("123456789"))
	}))
	defer server.Close()
	d := NewDownloader(root, server.Client(), 8)
	defer d.Close()
	d.Start(Episode{ID: "oversized", MediaURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.pool.Wait(ctx, "oversized"); err == nil {
		t.Fatal("oversized download succeeded")
	}
	if stats := d.Stats(); stats.Bytes != 9 || stats.Files != 1 {
		t.Fatalf("existing cache changed: %+v", stats)
	}
	if _, ok := d.Path("existing"); !ok {
		t.Fatal("existing over-limit episode was removed")
	}
}

func TestDownloaderReleasesRejectedWorkBeforeRetry(t *testing.T) {
	body := &eofGateBody{atEOF: make(chan struct{}), release: make(chan struct{})}
	client := &http.Client{Transport: downloadRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: -1, Header: make(http.Header)}, nil
	})}
	d := NewDownloader(t.TempDir(), client, 1024)
	defer d.Close()
	episode := Episode{ID: "episode", MediaURL: "http://example.test/episode"}
	d.Start(episode)
	select {
	case <-body.atEOF:
	case <-time.After(5 * time.Second):
		close(body.release)
		t.Fatal("download did not reach EOF")
	}
	if err := d.Delete(episode.ID); err != nil {
		close(body.release)
		t.Fatal(err)
	}
	if state := d.Start(episode); state.Status != "failed" {
		close(body.release)
		t.Fatalf("first restart = %+v", state)
	}
	if err := d.Delete(episode.ID); err != nil {
		close(body.release)
		t.Fatal(err)
	}
	if state := d.Start(episode); state.Status != "failed" {
		close(body.release)
		t.Fatalf("second restart = %+v", state)
	}
	close(body.release)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = d.pool.Wait(ctx, episode.ID)
	if ctx.Err() != nil {
		t.Fatal("old work did not exit")
	}
	state := d.Start(Episode{ID: episode.ID, MediaURL: "://invalid"})
	if state.Status != "queued" {
		t.Fatalf("retry after old work exited = %+v", state)
	}
	if err := d.pool.Wait(ctx, episode.ID); err == nil || !strings.Contains(err.Error(), "missing protocol scheme") {
		t.Fatalf("retry did not execute independently: %v", err)
	}
}

func TestDownloaderPublishesStreamingJobProgress(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "524288")
		_, _ = w.Write(bytes.Repeat([]byte("a"), 262144))
		if flush, ok := w.(http.Flusher); ok {
			flush.Flush()
		}
		<-release
		_, _ = w.Write(bytes.Repeat([]byte("b"), 262144))
	}))
	defer server.Close()
	downloader := NewDownloader(t.TempDir(), server.Client(), 1<<20)
	defer downloader.Close()
	downloader.Start(Episode{ID: "progress", MediaURL: server.URL})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, state := range downloader.Jobs() {
			if state.Key == "progress" && state.ProgressCurrent >= 262144 {
				if state.ProgressTotal != 524288 || state.ProgressMessage != "Downloading episode" {
					t.Fatalf("progress = %#v", state)
				}
				close(release)
				if err := downloader.pool.Wait(context.Background(), "progress"); err != nil {
					t.Fatal(err)
				}
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(release)
	t.Fatal("download progress was not published")
}

func TestDownloaderReconcileRetentionAndStats(t *testing.T) {
	root := t.TempDir()
	downloader := NewDownloader(root, nil, 1024)
	defer downloader.Close()
	episodes := []Episode{
		{ID: "new", Published: time.Unix(30, 0)},
		{ID: "middle", Published: time.Unix(20, 0)},
		{ID: "old", Published: time.Unix(10, 0)},
	}
	for _, episode := range episodes {
		if err := os.WriteFile(filepath.Join(root, episode.ID+".media"), []byte("1234"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	downloader.Reconcile(Feed{Episodes: episodes, Policy: FeedPolicy{RetainDownloads: 2}}, nil)
	if _, ok := downloader.Path("old"); ok {
		t.Fatal("old download was not removed")
	}
	if _, ok := downloader.Path("new"); !ok {
		t.Fatal("new download was removed")
	}
	stats := downloader.Stats()
	if stats.Files != 2 || stats.Bytes != 8 || stats.MaxBytes != 1024 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestDownloaderReconcileStartsOptInDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("episode"))
	}))
	defer server.Close()
	downloader := NewDownloader(t.TempDir(), server.Client(), 1024)
	defer downloader.Close()
	downloader.Reconcile(Feed{
		Episodes: []Episode{{ID: "new", MediaURL: server.URL, Published: time.Now()}},
		Policy:   FeedPolicy{AutoDownload: true, RetainDownloads: 1},
	}, nil)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := downloader.Path("new"); ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("download did not complete: %+v", downloader.Status("new"))
}

func TestDownloaderReconcileDeletesOnlyExpiredPlayedDownloads(t *testing.T) {
	root := t.TempDir()
	downloader := NewDownloader(root, nil, 1024)
	defer downloader.Close()
	episodes := []Episode{{ID: "old-played"}, {ID: "recent-played"}, {ID: "unplayed"}}
	for _, episode := range episodes {
		if err := os.WriteFile(filepath.Join(root, episode.ID+".media"), []byte("saved"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	downloader.Reconcile(Feed{Episodes: episodes, Policy: FeedPolicy{DeletePlayedAfterDays: 7}}, func(id string) (time.Time, bool) {
		switch id {
		case "old-played":
			return now.Add(-8 * 24 * time.Hour), true
		case "recent-played":
			return now.Add(-6 * 24 * time.Hour), true
		default:
			return time.Time{}, false
		}
	})
	if _, ok := downloader.Path("old-played"); ok {
		t.Fatal("expired played download was not removed")
	}
	for _, id := range []string{"recent-played", "unplayed"} {
		if _, ok := downloader.Path(id); !ok {
			t.Fatalf("%s download was removed", id)
		}
	}
}

func TestDownloaderDoesNotRedownloadExpiredPlayedEpisode(t *testing.T) {
	root := t.TempDir()
	downloader := NewDownloader(root, nil, 1024)
	defer downloader.Close()
	if err := os.WriteFile(filepath.Join(root, "played.media"), []byte("saved"), 0o600); err != nil {
		t.Fatal(err)
	}
	downloader.Reconcile(Feed{
		Episodes: []Episode{{ID: "played", MediaURL: "://must-not-download", Published: time.Now()}},
		Policy:   FeedPolicy{AutoDownload: true, RetainDownloads: 1, DeletePlayedAfterDays: 1},
	}, func(string) (time.Time, bool) { return time.Now().Add(-48 * time.Hour), true })
	if _, ok := downloader.Path("played"); ok {
		t.Fatal("expired played download was retained")
	}
	if state := downloader.Status("played"); state.Status != "" {
		t.Fatalf("expired played episode was queued again: %+v", state)
	}
}

func TestDownloaderReportsAndRetriesFailedEpisodes(t *testing.T) {
	downloader := NewDownloader(t.TempDir(), nil, 1024)
	downloader.state["failed"] = Download{EpisodeID: "failed", Status: "failed", Error: "offline"}
	downloader.state["removed"] = Download{EpisodeID: "removed", Status: "failed", Error: "offline"}
	if stats := downloader.Stats(); stats.Failed != 2 {
		t.Fatalf("failed stats = %+v", stats)
	}
	if retried := downloader.RetryFailed([]Episode{{ID: "failed", MediaURL: "://invalid"}}); retried != 1 {
		t.Fatalf("retried = %d", retried)
	}
	downloader.Close()
}

func TestDownloaderRestoresFailedJobProgressForRetry(t *testing.T) {
	root := t.TempDir()
	pool, err := jobs.NewPersistent(1, 2, filepath.Join(root, "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = pool.Do(context.Background(), "episode", func(ctx context.Context) error {
		jobs.Report(ctx, 123, 456, "Downloading episode")
		return errors.New("offline")
	})
	if err == nil {
		t.Fatal("failed job unexpectedly succeeded")
	}
	pool.Close()
	downloader := NewDownloader(root, nil, 1024)
	state := downloader.Status("episode")
	if state.Status != "failed" || state.Bytes != 123 {
		t.Fatalf("restored download = %#v", state)
	}
	if retried := downloader.RetryFailed([]Episode{{ID: "episode", MediaURL: "://invalid"}}); retried != 1 {
		t.Fatalf("retried = %d", retried)
	}
	downloader.Close()
}
