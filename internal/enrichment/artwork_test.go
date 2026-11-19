package enrichment

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestArtworkCacheOnlyFetchesBoundedTMDBImages(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"image/jpeg"}}, Body: io.NopCloser(strings.NewReader("jpeg")), ContentLength: 4}, nil
	})}
	cache := NewArtworkCache(t.TempDir(), 1024, client)
	for range 2 {
		path, contentType, err := cache.Fetch(context.Background(), "https://image.tmdb.org/t/p/w342/poster.jpg")
		if err != nil || contentType != "image/jpeg" {
			t.Fatalf("path=%q type=%q err=%v", path, contentType, err)
		}
		if data, err := os.ReadFile(path); err != nil || string(data) != "jpeg" {
			t.Fatalf("cached data=%q err=%v", data, err)
		}
	}
	if calls != 1 {
		t.Fatalf("requests=%d, want cached single request", calls)
	}
	if _, _, err := cache.Fetch(context.Background(), "http://127.0.0.1/private.jpg"); err == nil {
		t.Fatal("private artwork URL was accepted")
	}
}

func TestProviderArtworkURLValidation(t *testing.T) {
	for _, value := range []string{"https://image.tmdb.org/t/p/w342/poster.jpg", "https://assets.hardcover.app/books/cover.webp", "https://covers.openlibrary.org/b/id/12345-M.jpg?default=false"} {
		if !ValidProviderArtworkURL(value) {
			t.Fatalf("approved artwork URL rejected: %s", value)
		}
	}
	for _, value := range []string{"http://assets.hardcover.app/cover.jpg", "https://assets.hardcover.app.evil.test/cover.jpg", "https://assets.hardcover.app:443/cover.jpg", "https://assets.hardcover.app/cover.jpg?redirect=x", "https://127.0.0.1/cover.jpg", "https://covers.openlibrary.org/b/id/not-a-number-M.jpg?default=false", "https://covers.openlibrary.org/b/id/12345-L.jpg?default=false", "https://covers.openlibrary.org/b/id/12345-M.jpg", "https://covers.openlibrary.org/b/id/12345-M.jpg?default=true", "https://covers.openlibrary.org.evil.test/b/id/12345-M.jpg?default=false"} {
		if ValidProviderArtworkURL(value) {
			t.Fatalf("unsafe artwork URL accepted: %s", value)
		}
	}
}

func TestSlowArtworkDownloadDoesNotBlockCachedOrDuplicateRequests(t *testing.T) {
	release := make(chan struct{})
	var started sync.WaitGroup
	started.Add(1)
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if strings.Contains(request.URL.Path, "slow") {
			calls.Add(1)
			started.Done()
			<-release
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"image/jpeg"}}, Body: io.NopCloser(strings.NewReader("jpeg")), ContentLength: 4}, nil
	})}
	cache := NewArtworkCache(t.TempDir(), 1<<20, client)
	if _, _, err := cache.Fetch(context.Background(), "https://image.tmdb.org/t/p/w342/fast.jpg"); err != nil {
		t.Fatal(err)
	}
	slow := "https://image.tmdb.org/t/p/w342/slow.jpg"
	results := make(chan error, 2)
	go func() { _, _, err := cache.Fetch(context.Background(), slow); results <- err }()
	started.Wait()
	// A cache hit and a different image are served while the slow one is still in flight.
	done := make(chan error, 1)
	go func() {
		_, _, err := cache.Fetch(context.Background(), "https://image.tmdb.org/t/p/w342/fast.jpg")
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a cached image waited behind a slow download")
	}
	// A duplicate request shares the download, and a cancelled waiter returns promptly.
	go func() { _, _, err := cache.Fetch(context.Background(), slow); results <- err }()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, _, err := cache.Fetch(ctx, slow); err == nil {
		t.Fatal("a cancelled waiter should return its context error")
	}
	close(release)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("slow image downloaded %d times, want one shared download", calls.Load())
	}
}

func TestTMDBBackdropURLValidation(t *testing.T) {
	if !ValidTMDBBackdropURL("https://image.tmdb.org/t/p/w1280/wide.jpg") || !ValidProviderArtworkURL("https://image.tmdb.org/t/p/w1280/wide.jpg") {
		t.Fatal("TMDB backdrop rejected")
	}
	for _, value := range []string{"https://image.tmdb.org/t/p/original/wide.jpg", "http://image.tmdb.org/t/p/w1280/wide.jpg", "https://image.tmdb.org/t/p/w1280/wide.jpg?x=1", "https://evil.test/t/p/w1280/wide.jpg"} {
		if ValidTMDBBackdropURL(value) {
			t.Fatalf("unsafe backdrop accepted: %s", value)
		}
	}
}
