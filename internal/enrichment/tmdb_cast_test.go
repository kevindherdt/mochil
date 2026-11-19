package enrichment

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDetailsRefreshLegacyCacheAndReturnBoundedCast(t *testing.T) {
	path := t.TempDir() + "/details.json"
	old := `{"tv/42":{"created":"` + time.Now().Format(time.RFC3339) + `","details":{"genres":["Drama"]}}}`
	if err := os.WriteFile(path, []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if !strings.Contains(r.URL.Query().Get("append_to_response"), "credits") {
			t.Error("credits not requested")
		}
		cast := `{"name":" Actor ","character":" Hero "},` + strings.Repeat(`{"name":"Another","character":"Role"},`, 20) + `{"name":"Last"}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"credits":{"cast":[` + cast + `]}}`)), Header: http.Header{}}, nil
	})}
	tmdb := NewTMDB("secret", client)
	tmdb.PersistDetails(path)
	got, err := tmdb.Details(context.Background(), "tv", "42")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Cast []struct {
			Name      string `json:"name"`
			Character string `json:"character"`
		} `json:"cast"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(payload.Cast) != 10 || payload.Cast[0].Name != "Actor" || payload.Cast[0].Character != "Hero" {
		t.Fatalf("calls=%d cast=%+v", calls, payload.Cast)
	}
	again := NewTMDB("secret", client)
	again.PersistDetails(path)
	if _, err := again.Details(context.Background(), "tv", "42"); err != nil || calls != 1 {
		t.Fatalf("cache not reused: calls=%d err=%v", calls, err)
	}
}
