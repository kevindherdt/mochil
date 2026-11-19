package web

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/video"
)

type subtitleProbe struct{}

func (subtitleProbe) Probe(string) (video.ProbeResult, error) {
	return video.ProbeResult{Video: []video.VideoStream{{Index: 0}}, Subtitles: []video.SubtitleStream{{Index: 2, Codec: "subrip", Text: true}, {Index: 3, Codec: "hdmv_pgs_subtitle"}}}, nil
}

type previewProbe struct{}

func (previewProbe) Probe(string) (video.ProbeResult, error) {
	return video.ProbeResult{Duration: 125, Video: []video.VideoStream{{Index: 0}}}, nil
}

func TestVideoProgressDistinguishesPassiveRewindFromExplicitSeek(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Film.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, VideoState: filepath.Join(t.TempDir(), "progress.json")}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/video-progress/movies/" + store.Items()[0].ID
	for _, step := range []struct {
		body string
		want int
	}{
		{`{"position_seconds":80,"duration_seconds":100,"expected_revision":0,"device_id":"living-room-streamer"}`, http.StatusOK},
		{`{"position_seconds":20,"duration_seconds":100,"expected_revision":1,"device_id":"living-room-streamer"}`, http.StatusConflict},
		{`{"position_seconds":20,"duration_seconds":100,"expected_revision":1,"explicit_seek":true,"device_id":"living-room-streamer"}`, http.StatusOK},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, path, strings.NewReader(step.body)))
		if response.Code != step.want {
			t.Fatalf("body=%s response=%d %s", step.body, response.Code, response.Body.String())
		}
	}
	current := httptest.NewRecorder()
	handler.ServeHTTP(current, httptest.NewRequest(http.MethodGet, path, nil))
	if current.Code != http.StatusOK || !strings.Contains(current.Body.String(), `"device_id":"living-room-streamer"`) {
		t.Fatalf("video progress lost device attribution=%d %s", current.Code, current.Body.String())
	}
	continued := httptest.NewRecorder()
	handler.ServeHTTP(continued, httptest.NewRequest(http.MethodGet, "/api/video-progress?continue=1&limit=5", nil))
	var items []videoView
	if continued.Code != http.StatusOK || json.Unmarshal(continued.Body.Bytes(), &items) != nil || len(items) != 1 {
		t.Fatalf("continue list=%d %s", continued.Code, continued.Body.String())
	}
	finished := httptest.NewRecorder()
	handler.ServeHTTP(finished, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"finished":true}`)))
	continued = httptest.NewRecorder()
	handler.ServeHTTP(continued, httptest.NewRequest(http.MethodGet, "/api/video-progress?continue=1&limit=5", nil))
	if continued.Code != http.StatusOK || strings.TrimSpace(continued.Body.String()) != "[]" {
		t.Fatalf("finished continue list=%d %s", continued.Code, continued.Body.String())
	}
}

func TestVideoShowsGroupSeasonsAndKeepUnmatchedFiles(t *testing.T) {
	root := t.TempDir()
	files := []string{
		"Example Show [tvdbid=42]/Season 1/Example Show - S01E02 - Two.mkv",
		"Example Show [tvdbid=42]/Season 1/Example Show - S01E01 - One.mkv",
		"Example Show [tvdbid=42]/Season 2/Example Show - S02E01 - Return.mkv",
		"unmatched-video.mkv",
	}
	for _, name := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("video"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "Example Show [tvdbid=42]", "poster.jpg"), []byte("show-art"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Example Show [tvdbid=42]", "Season 1", "poster.jpg"), []byte("season-art"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("tv", root, filepath.Join(t.TempDir(), "cache.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "tv", Name: "TV", Kind: "tv", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/video-shows?library=tv", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("shows status = %d: %s", response.Code, response.Body.String())
	}
	var shows []videoShowView
	if err := json.Unmarshal(response.Body.Bytes(), &shows); err != nil {
		t.Fatal(err)
	}
	if len(shows) != 2 || shows[0].Name != "Example Show" || shows[0].Count != 3 || len(shows[0].Seasons) != 2 || !shows[1].Other || shows[1].Count != 1 {
		t.Fatalf("shows = %#v", shows)
	}
	paged := httptest.NewRecorder()
	handler.ServeHTTP(paged, httptest.NewRequest(http.MethodGet, "/api/video-shows?library=tv&limit=1&offset=1", nil))
	var pagedShows []videoShowView
	if paged.Code != http.StatusOK || paged.Header().Get("X-Total-Count") != "2" || json.Unmarshal(paged.Body.Bytes(), &pagedShows) != nil || len(pagedShows) != 1 || !pagedShows[0].Other {
		t.Fatalf("paged shows = status %d total %q body %s", paged.Code, paged.Header().Get("X-Total-Count"), paged.Body.String())
	}
	if !strings.Contains(shows[0].ArtworkURL, "scope=show") {
		t.Fatalf("show artwork URL = %q", shows[0].ArtworkURL)
	}
	if !strings.Contains(shows[0].Seasons[0].ArtworkURL, "scope=season") {
		t.Fatalf("season artwork URL = %q", shows[0].Seasons[0].ArtworkURL)
	}
	seasonArtwork := httptest.NewRecorder()
	handler.ServeHTTP(seasonArtwork, httptest.NewRequest(http.MethodGet, shows[0].Seasons[0].ArtworkURL, nil))
	if seasonArtwork.Code != http.StatusOK || seasonArtwork.Body.String() != "season-art" {
		t.Fatalf("season artwork = %d %q", seasonArtwork.Code, seasonArtwork.Body.String())
	}
	artwork := httptest.NewRecorder()
	handler.ServeHTTP(artwork, httptest.NewRequest(http.MethodGet, shows[0].ArtworkURL, nil))
	if artwork.Code != http.StatusOK || artwork.Body.String() != "show-art" {
		t.Fatalf("show artwork = %d %q", artwork.Code, artwork.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/videos?library=tv&show=Example+Show&season=1", nil))
	if response.Code != http.StatusOK || response.Header().Get("X-Total-Count") != "2" {
		t.Fatalf("season response = %d total=%q body=%s", response.Code, response.Header().Get("X-Total-Count"), response.Body.String())
	}
	var episodes []videoView
	if err := json.Unmarshal(response.Body.Bytes(), &episodes); err != nil {
		t.Fatal(err)
	}
	if len(episodes) == 0 || strings.Contains(episodes[0].ArtworkURL, "scope=season") {
		t.Fatalf("episode artwork = %#v", episodes)
	}
}

type subtitleExtractorStub struct{ index int }

func (s *subtitleExtractorStub) Extract(_ context.Context, _ *os.File, index int) ([]byte, error) {
	s.index = index
	return []byte("WEBVTT\n\n00:00.000 --> 00:01.000\nHello\n"), nil
}

func TestVideoSubtitleOnlyExtractsIndexedTextTracks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "episode.mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("anime", root, filepath.Join(t.TempDir(), "cache.json"))
	store.SetProber(subtitleProbe{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	extractor := &subtitleExtractorStub{index: -1}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, SubtitleExtractor: extractor}, []Library{{ID: "anime", Name: "Anime", Kind: "anime", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/video-subtitles/anime/"+item.ID+"/2.vtt", nil))
	if response.Code != http.StatusOK || extractor.index != 2 || !strings.Contains(response.Body.String(), "WEBVTT") || response.Header().Get("Content-Type") != "text/vtt; charset=utf-8" {
		t.Fatalf("subtitle response = %d, index %d, type %q, body %q", response.Code, extractor.index, response.Header().Get("Content-Type"), response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/video-subtitles/anime/"+item.ID+"/3.vtt", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("image subtitle status = %d", response.Code)
	}
}

func TestVideoPreviewGeneratesQuantizedFrame(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "movie.mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(root, "preview-command")
	script := "#!/bin/sh\nfor last do :; done\nprintf preview > \"$last\"\n"
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, filepath.Join(t.TempDir(), "cache.json"))
	store.SetProber(previewProbe{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	previews, err := video.NewPreviewStore(filepath.Join(t.TempDir(), "previews"), command, 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, VideoPreviews: previews}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	item := store.Items()[0]
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/video-previews/movies/"+item.ID+"/27.jpg", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/jpeg" || response.Header().Get("X-Video-Preview-Position") != "20" || response.Body.String() != "preview" {
		t.Fatalf("preview response = %d, type %q, position %q, body %q", response.Code, response.Header().Get("Content-Type"), response.Header().Get("X-Video-Preview-Position"), response.Body.String())
	}
}
