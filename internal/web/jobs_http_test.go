package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mochil/internal/jobs"
)

func TestJobsAPIReportsBoundedBackgroundWork(t *testing.T) {
	pool := jobs.New(1, 2)
	defer pool.Close()
	release := make(chan struct{})
	reported := make(chan struct{})
	if err := pool.Submit("podcasts:refresh-stale", func(ctx context.Context) error {
		jobs.Report(ctx, 2, 5, "Refreshing feeds")
		close(reported)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}); err != nil {
		t.Fatal(err)
	}
	<-reported
	m := &multiServer{backgroundJobs: pool}
	w := httptest.NewRecorder()
	m.apiJobs(w, httptest.NewRequest(http.MethodGet, "/api/jobs", nil))
	close(release)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var values []jobView
	if err := json.Unmarshal(w.Body.Bytes(), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].Queue != "background" || values[0].Key != "podcasts:refresh-stale" {
		t.Fatalf("jobs: %#v", values)
	}
	if values[0].Status != "queued" && values[0].Status != "running" {
		t.Fatalf("unexpected status: %q", values[0].Status)
	}
	if values[0].QueuedAt.Equal(time.Time{}) {
		t.Fatal("missing queue time")
	}
	if values[0].ProgressCurrent != 2 || values[0].ProgressTotal != 5 || values[0].ProgressMessage != "Refreshing feeds" {
		t.Fatalf("missing progress: %#v", values[0])
	}
}

func TestJobsAPIIsReadOnly(t *testing.T) {
	m := &multiServer{}
	w := httptest.NewRecorder()
	m.apiJobs(w, httptest.NewRequest(http.MethodPost, "/api/jobs", nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("status %d allow %q", w.Code, w.Header().Get("Allow"))
	}
}
