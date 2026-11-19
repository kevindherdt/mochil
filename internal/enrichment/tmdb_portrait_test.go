package enrichment

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPortraitVersionOneFallbackThenRetry(t *testing.T) {
	path := t.TempDir() + "/details.json"
	old := `{"movie/42":{"created":"` + time.Now().Format(time.RFC3339) + `","version":1,"details":{"cast":[{"name":"Old"}]}}}`
	if err := os.WriteFile(path, []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return &http.Response{StatusCode: 503, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"credits":{"cast":[{"name":"New","profile_path":"/abc.jpg"}]}}`))}, nil
	})}
	tmdb := NewTMDB("token", client)
	tmdb.PersistDetails(path)
	first, err := tmdb.Details(context.Background(), "movie", "42")
	if err != nil || len(first.Cast) != 1 || first.Cast[0].Name != "Old" {
		t.Fatalf("fallback=%+v err=%v", first, err)
	}
	second, err := tmdb.Details(context.Background(), "movie", "42")
	if err != nil || len(second.Cast) != 1 || second.Cast[0].ProfilePath != "/abc.jpg" || calls != 2 {
		t.Fatalf("retry=%+v calls=%d err=%v", second, calls, err)
	}
	_, err = tmdb.Details(context.Background(), "movie", "42")
	if err != nil || calls != 2 {
		t.Fatalf("cached calls=%d err=%v", calls, err)
	}
}

func TestPortraitRefreshesVersionOneAndRejectsUnsafePaths(t *testing.T) {
	path := t.TempDir() + "/details.json"
	old := `{"movie/42":{"created":"` + time.Now().Format(time.RFC3339) + `","version":1,"details":{"cast":[{"name":"Old"}]}}}`
	if err := os.WriteFile(path, []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"credits":{"cast":[{"name":"A","profile_path":"/abc123.jpg"},{"name":"B","profile_path":"/../evil.jpg"}]}}`))}, nil
	})}
	tmdb := NewTMDB("token", client)
	tmdb.PersistDetails(path)
	got, err := tmdb.Details(context.Background(), "movie", "42")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(got.Cast) != 2 || got.Cast[0].ProfilePath != "/abc123.jpg" || got.Cast[1].ProfilePath != "" {
		t.Fatalf("calls=%d cast=%+v", calls, got.Cast)
	}
}
