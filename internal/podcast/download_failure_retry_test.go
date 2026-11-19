package podcast

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"mochil/internal/jobs"
)

type failingCloseGate struct {
	closing chan struct{}
	release chan struct{}
}

func (b *failingCloseGate) Read([]byte) (int, error) { return 0, io.EOF }
func (b *failingCloseGate) Close() error {
	close(b.closing)
	<-b.release
	return nil
}

func TestDownloaderRetryWhileFailedJobExitsPreservesFailure(t *testing.T) {
	body := &failingCloseGate{closing: make(chan struct{}), release: make(chan struct{})}
	client := &http.Client{Transport: downloadRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Body: body, Header: make(http.Header)}, nil
	})}
	d := NewDownloader(t.TempDir(), client, 1024)
	defer d.Close()
	defer close(body.release)
	episode := Episode{ID: "episode", MediaURL: "http://example.test/episode"}
	if state, err := d.StartWithError(episode); err != nil || state.Status != "queued" {
		t.Fatalf("initial admission = %+v, %v", state, err)
	}
	select {
	case <-body.closing:
	case <-time.After(5 * time.Second):
		t.Fatal("failed download did not reach response cleanup")
	}
	original := d.Status(episode.ID)
	if original.Status != "failed" || original.Error != "502 Bad Gateway" {
		t.Fatalf("original failure = %+v", original)
	}
	state, err := d.StartWithError(episode)
	if !errors.Is(err, jobs.ErrDuplicate) || state.Status != "failed" || !strings.Contains(state.Error, "still stopping") {
		t.Fatalf("retry admission = %+v, %v", state, err)
	}
	if current := d.Status(episode.ID); current != original {
		t.Fatalf("retry overwrote original failure: got %+v, want %+v", current, original)
	}
}

func TestDownloaderCanceledJobRetryReportsAdmissionFailure(t *testing.T) {
	body := &eofGateBody{atEOF: make(chan struct{}), release: make(chan struct{})}
	client := &http.Client{Transport: downloadRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: -1, Header: make(http.Header)}, nil
	})}
	d := NewDownloader(t.TempDir(), client, 1024)
	defer d.Close()
	released := false
	defer func() {
		if !released {
			close(body.release)
		}
	}()
	episode := Episode{ID: "episode", MediaURL: "http://example.test/episode"}
	d.Start(episode)
	select {
	case <-body.atEOF:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not reach EOF")
	}
	if err := d.Delete(episode.ID); err != nil {
		t.Fatal(err)
	}
	state, err := d.StartWithError(episode)
	if !errors.Is(err, jobs.ErrDuplicate) || state.Status != "failed" || !strings.Contains(state.Error, "still stopping") {
		t.Fatalf("retry admission = %+v, %v", state, err)
	}
	if current := d.Status(episode.ID); current != state {
		t.Fatalf("retry admission not retained: got %+v, want %+v", current, state)
	}
	close(body.release)
	released = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.pool.Wait(ctx, episode.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled job result = %v", err)
	}
	if current := d.Status(episode.ID); current != state {
		t.Fatalf("canceled job overwrote retry admission: got %+v, want %+v", current, state)
	}
}
