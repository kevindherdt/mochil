package main

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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mochil/internal/appconfig"
	"mochil/internal/catalog"
	"mochil/internal/jobs"
	"mochil/internal/web"
)

type blockingCloseBackend struct {
	started chan struct{}
	release chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func (b *blockingCloseBackend) Scan() (catalog.Stats, error) {
	b.once.Do(func() { close(b.started) })
	<-b.release
	return catalog.Stats{}, nil
}
func (*blockingCloseBackend) Books() []catalog.Book            { return nil }
func (*blockingCloseBackend) Book(string) (catalog.Book, bool) { return catalog.Book{}, false }
func (*blockingCloseBackend) Acquire(context.Context, catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{}, context.Canceled
}
func (b *blockingCloseBackend) Close() { close(b.closed) }

func TestSupportsPageStreamingOnlyForComicLibraries(t *testing.T) {
	for _, kind := range []string{"comics", "manga", "mixed"} {
		if !supportsPageStreaming(kind) {
			t.Errorf("%s should support page streaming", kind)
		}
	}
	for _, kind := range []string{"books", "movies", "tv", "anime", "audiobooks", "podcasts", "music"} {
		if supportsPageStreaming(kind) {
			t.Errorf("%s should not support page streaming", kind)
		}
	}
}

func TestScanSchedulerWaitsForPreviousCallback(t *testing.T) {
	var scheduler scanScheduler
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	scheduler.resetPlans(context.Background(), []scanPlan{{time.Millisecond, func() {
		once.Do(func() { close(started) })
		<-release
	}}})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("scheduled callback did not start")
	}
	reset := make(chan struct{})
	go func() {
		scheduler.resetPlans(context.Background(), []scanPlan{{0, func() {}}})
		close(reset)
	}()
	select {
	case <-reset:
		t.Fatal("scheduler reset returned before callback stopped")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-reset:
	case <-time.After(time.Second):
		t.Fatal("scheduler reset did not finish")
	}
	scheduler.close()
}

func TestReplacingLibraryDrainsItsActiveScanBeforeClose(t *testing.T) {
	backend := &blockingCloseBackend{started: make(chan struct{}), release: make(chan struct{}), closed: make(chan struct{})}
	library := web.Library{ID: "remote", Store: backend}
	application := &liveApplication{
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		scanJobs: jobs.New(1, 2),
	}
	defer application.scanJobs.Close()
	application.scheduleLibraryScan(library, "old")
	<-backend.started
	done := make(chan struct{})
	go func() {
		application.closeReplacedLibraries(
			runtimeConfig{libraries: []web.Library{library}, libraryFingerprints: map[string]string{"remote": "old"}},
			runtimeConfig{libraryFingerprints: map[string]string{}},
		)
		close(done)
	}()
	select {
	case <-backend.closed:
		t.Fatal("backend closed while its scan was active")
	case <-time.After(20 * time.Millisecond):
	}
	close(backend.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("replacement did not finish")
	}
	select {
	case <-backend.closed:
	default:
		t.Fatal("backend was not closed after its scan drained")
	}
}

func TestSupportsVideoLibraries(t *testing.T) {
	for _, kind := range []string{"movies", "tv", "anime"} {
		if !supportsVideoLibrary(kind) {
			t.Errorf("%s should use the video library", kind)
		}
	}
	for _, kind := range []string{"books", "comics", "manga", "mixed", "audiobooks", "podcasts", "music"} {
		if supportsVideoLibrary(kind) {
			t.Errorf("%s should not use the video library", kind)
		}
	}
}

func TestDocumentFilesystemLibraryUsesIndependentStore(t *testing.T) {
	root := t.TempDir()
	documents := filepath.Join(root, "notes")
	if err := os.MkdirAll(documents, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.json")
	cfg := appconfig.Config{
		Listen: ":8080", ScanEvery: "0", CacheDirectory: filepath.Join(root, "data"),
		Libraries: []appconfig.Library{{
			ID: "notes", Name: "Notes", Kind: "documents",
			Source: appconfig.Source{Type: "filesystem", Path: documents},
		}},
	}
	data, _ := json.Marshal(cfg)
	if err := saveConfig(configPath, data); err != nil {
		t.Fatal(err)
	}
	runtime, err := configure(configPath, web.Config{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := runtime.libraries[0]
	if got.Documents == nil || got.Store != nil || got.Video != nil || got.Audio != nil {
		t.Fatalf("documents used the wrong backend: %#v", got)
	}
}

func TestSaveConfigRetainsPreviousValidConfiguration(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	first := []byte(`{"scan_every":"10m","libraries":[]}`)
	second := []byte(`{"scan_every":"5m","libraries":[]}`)
	if err := saveConfig(path, first); err != nil {
		t.Fatal(err)
	}
	if err := saveConfig(path, second); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".previous")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(backup)) != string(first) {
		t.Fatalf("backup = %q, want %q", backup, first)
	}
}

func TestLibraryFingerprintIncludesContentKind(t *testing.T) {
	library := appconfig.Library{Source: appconfig.Source{Type: "filesystem", Path: "/media"}}
	library.Kind = "books"
	books := libraryFingerprint("/data", library)
	library.Kind = "anime"
	if anime := libraryFingerprint("/data", library); anime == books {
		t.Fatal("changing content kind must replace the library backend")
	}
}

func TestAllLocalVideoKindsUseVideoStore(t *testing.T) {
	for _, kind := range []string{"movies", "tv", "anime"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			media := filepath.Join(root, kind)
			if err := os.MkdirAll(media, 0o755); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(root, "config.json")
			cfg := appconfig.Config{
				Listen: ":8080", ScanEvery: "0", CacheDirectory: filepath.Join(root, "data"),
				Libraries: []appconfig.Library{{
					ID: kind, Name: kind, Kind: kind,
					Source: appconfig.Source{Type: "filesystem", Path: media},
				}},
			}
			data, _ := json.Marshal(cfg)
			if err := saveConfig(configPath, data); err != nil {
				t.Fatal(err)
			}
			runtime, err := configure(configPath, web.Config{}, 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := runtime.libraries[0]; got.Video == nil || got.Store != nil {
				t.Fatalf("%s used the publication backend: %#v", kind, got)
			}
		})
	}
}

func TestMovieFilesystemLibraryUsesIndependentVideoSnapshot(t *testing.T) {
	// These tests exercise the catalogue over HTTP, not the sign-in door.
	t.Setenv("MOCHIL_ALLOW_OPEN", "1")
	root := t.TempDir()
	movies := filepath.Join(root, "movies")
	if err := os.MkdirAll(movies, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(movies, "Movie 01.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.json")
	cfg := appconfig.Config{
		Listen: ":8080", ScanEvery: "0", CacheDirectory: filepath.Join(root, "data"),
		Libraries: []appconfig.Library{{
			ID: "movies", Name: "Movies", Kind: "movies",
			Source:    appconfig.Source{Type: "filesystem", Path: movies},
			Inference: appconfig.Inference{FolderLayout: "auto"},
		}},
	}
	data, _ := json.Marshal(cfg)
	if err := saveConfig(configPath, data); err != nil {
		t.Fatal(err)
	}
	runtime, err := configure(configPath, web.Config{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(runtime.libraries) != 1 || runtime.libraries[0].Video == nil || runtime.libraries[0].Store != nil {
		t.Fatalf("movie runtime did not preserve the video boundary: %#v", runtime.libraries)
	}
	if !scanLibrary(slog.New(slog.NewTextHandler(io.Discard, nil)), runtime.libraries[0]) {
		t.Fatal("movie scan failed")
	}
	handler, err := (&liveApplication{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}).build(runtime)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/libraries", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"movies"`) || !strings.Contains(response.Body.String(), `"count":1`) {
		t.Fatalf("movie summary = %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/videos?library=movies&q=movie", nil))
	if response.Code != http.StatusOK || response.Header().Get("X-Total-Count") != "1" || !strings.Contains(response.Body.String(), `"title":"Movie 01"`) {
		t.Fatalf("movie catalogue = %d %s", response.Code, response.Body.String())
	}
	var videos []struct {
		StreamURL string `json:"stream_url"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &videos); err != nil || len(videos) != 1 || videos[0].StreamURL == "" {
		t.Fatalf("movie stream URL = %#v, %v", videos, err)
	}
	response = httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, videos[0].StreamURL, nil)
	request.Header.Set("Range", "bytes=1-3")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent || response.Body.String() != "ide" || response.Header().Get("Content-Range") != "bytes 1-3/5" {
		t.Fatalf("movie range = %d %q %q", response.Code, response.Body.String(), response.Header().Get("Content-Range"))
	}
	progressURL := "/api/video-progress/movies/" + strings.Split(videos[0].StreamURL, "/")[3]
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, progressURL, strings.NewReader(`{"position_seconds":42,"duration_seconds":100,"device_id":"phone"}`))
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"position_seconds":42`) {
		t.Fatalf("save movie progress = %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, progressURL, nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"device_id":"phone"`) {
		t.Fatalf("load movie progress = %d %s", response.Code, response.Body.String())
	}
}

func TestLiveApplicationAtomicallyPublishesSavedLibraries(t *testing.T) {
	t.Setenv("MOCHIL_ALLOW_OPEN", "1")
	root := t.TempDir()
	firstPath, secondPath := filepath.Join(root, "first"), filepath.Join(root, "second")
	if err := os.MkdirAll(firstPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secondPath, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.json")
	base := appconfig.Config{
		Listen: ":8080", ScanEvery: "0", CacheDirectory: filepath.Join(root, "data"),
		Libraries: []appconfig.Library{{
			ID: "first", Name: "First", Kind: "books",
			Source: appconfig.Source{Type: "filesystem", Path: firstPath}, Inference: appconfig.Inference{FolderLayout: "auto"},
		}},
	}
	initialData, _ := json.Marshal(base)
	if err := saveConfig(configPath, initialData); err != nil {
		t.Fatal(err)
	}
	runtime, err := configure(configPath, web.Config{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	application := &liveApplication{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), configPath: configPath, current: runtime}
	handler, err := application.build(runtime)
	if err != nil {
		t.Fatal(err)
	}
	application.handler.Store(handler)
	base.Libraries[0].Name = "TV Series"
	base.Libraries = append(base.Libraries, appconfig.Library{ID: "second", Name: "Second", Kind: "books", Source: appconfig.Source{Type: "filesystem", Path: secondPath}, Inference: appconfig.Inference{FolderLayout: "auto"}})
	updatedData, _ := json.Marshal(base)
	result, err := application.apply(updatedData)
	if err != nil || !result.Applied || result.Libraries != 2 {
		t.Fatalf("live apply = %#v, %v", result, err)
	}
	response := httptest.NewRecorder()
	application.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/libraries", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"second"`) || !strings.Contains(response.Body.String(), `"name":"TV Series"`) {
		t.Fatalf("new runtime was not published: %d %s", response.Code, response.Body.String())
	}
	saved, err := os.ReadFile(configPath)
	if err != nil || !strings.Contains(string(saved), `"second"`) || !strings.Contains(string(saved), `"TV Series"`) {
		t.Fatalf("configuration was not persisted: %s %v", saved, err)
	}
}

func TestReuseUnchangedLibrariesOnlyScansChangedBackends(t *testing.T) {
	root := t.TempDir()
	firstPath, secondPath := filepath.Join(root, "first"), filepath.Join(root, "second")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	configPath := filepath.Join(root, "config.json")
	cfg := appconfig.Config{
		Listen: ":8080", ScanEvery: "0", CacheDirectory: filepath.Join(root, "data"),
		Libraries: []appconfig.Library{
			{ID: "first", Name: "First", Kind: "books", Source: appconfig.Source{Type: "filesystem", Path: firstPath}, Inference: appconfig.Inference{FolderLayout: "auto"}},
			{ID: "second", Name: "Second", Kind: "books", Source: appconfig.Source{Type: "filesystem", Path: secondPath}, Inference: appconfig.Inference{FolderLayout: "auto"}},
		},
	}
	writeRuntime := func(value appconfig.Config) runtimeConfig {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := saveConfig(configPath, data); err != nil {
			t.Fatal(err)
		}
		runtime, err := configure(configPath, web.Config{}, 0)
		if err != nil {
			t.Fatal(err)
		}
		return runtime
	}
	current := writeRuntime(cfg)
	candidate := writeRuntime(cfg)
	toScan, reused := reuseUnchangedLibraries(&candidate, current)
	if len(toScan) != 0 || reused != 2 {
		t.Fatalf("unchanged reconcile scanned %d and reused %d", len(toScan), reused)
	}
	for index := range current.libraries {
		if candidate.libraries[index].Store != current.libraries[index].Store {
			t.Fatalf("library %q did not retain its backend", current.libraries[index].ID)
		}
	}

	// Presentation-only changes do not invalidate the backend snapshot.
	cfg.Libraries[0].Name = "TV Series"
	candidate = writeRuntime(cfg)
	toScan, reused = reuseUnchangedLibraries(&candidate, current)
	if len(toScan) != 0 || reused != 2 || candidate.libraries[0].Name != "TV Series" {
		t.Fatalf("presentation reconcile scanned %d and reused %d", len(toScan), reused)
	}

	// A source change invalidates only that shelf.
	cfg.Libraries[1].Source.Path = firstPath
	candidate = writeRuntime(cfg)
	toScan, reused = reuseUnchangedLibraries(&candidate, current)
	if len(toScan) != 1 || toScan[0].ID != "second" || reused != 1 {
		t.Fatalf("backend reconcile = scan %#v, reused %d", toScan, reused)
	}
}

type panickingBackend struct{ blockingCloseBackend }

func (*panickingBackend) Scan() (catalog.Stats, error) { panic("malformed archive") }

func TestPanickingScanFailsOnlyThatLibrary(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if scanLibrary(logger, web.Library{ID: "broken", Store: &panickingBackend{}}) {
		t.Fatal("a library whose scan panicked must not be reported ready")
	}
	if !scanLibrary(logger, web.Library{ID: "fine", Store: &blockingCloseBackend{started: make(chan struct{}), release: closedChan()}}) {
		t.Fatal("the next library must still scan")
	}
}

func closedChan() chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}

func TestEachScanIntervalRunsOnItsOwn(t *testing.T) {
	var scheduler scanScheduler
	defer scheduler.close()
	var fast, never atomic.Int32
	scheduler.resetPlans(context.Background(), []scanPlan{
		{time.Millisecond, func() { fast.Add(1) }},
		{0, func() { never.Add(1) }},
	})
	deadline := time.After(time.Second)
	for fast.Load() < 3 {
		select {
		case <-deadline:
			t.Fatal("the interval plan never ran")
		case <-time.After(time.Millisecond):
		}
	}
	if never.Load() != 0 {
		t.Fatal("a library set to manual scans was scheduled")
	}
}

// A cache-directory change used to be published half-applied: the config, sessions and annotations moved while the reading
// store kept writing to the old path, so progress saved after the change went where the next boot does not look.
func TestLiveApplicationRefusesToMoveTheCacheDirectoryUnderAWriter(t *testing.T) {
	t.Setenv("MOCHIL_ALLOW_OPEN", "1")
	root := t.TempDir()
	libraryPath := filepath.Join(root, "books")
	if err := os.MkdirAll(libraryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.json")
	base := appconfig.Config{
		Listen: ":8080", ScanEvery: "0", CacheDirectory: filepath.Join(root, "data"),
		Libraries: []appconfig.Library{{
			ID: "books", Name: "Books", Kind: "books",
			Source: appconfig.Source{Type: "filesystem", Path: libraryPath}, Inference: appconfig.Inference{FolderLayout: "auto"},
		}},
	}
	initialData, _ := json.Marshal(base)
	if err := saveConfig(configPath, initialData); err != nil {
		t.Fatal(err)
	}
	runtime, err := configure(configPath, web.Config{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	application := &liveApplication{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), configPath: configPath, current: runtime}
	handler, err := application.build(runtime)
	if err != nil {
		t.Fatal(err)
	}
	application.handler.Store(handler)

	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	base.CacheDirectory = filepath.Join(root, "elsewhere")
	moved, _ := json.Marshal(base)
	if _, err := application.apply(moved); err == nil || !strings.Contains(err.Error(), "cache directory cannot change") {
		t.Fatalf("apply = %v, want a refusal naming the cache directory", err)
	}
	// Nothing may be written: a saved config the running process does not honour is the half-applied state itself.
	stored, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(stored) != string(before) {
		t.Fatalf("config was rewritten despite the refusal: %s", stored)
	}
	// And the running handler still answers from the old runtime.
	response := httptest.NewRecorder()
	application.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/libraries", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"books"`) {
		t.Fatalf("runtime changed anyway: %d %s", response.Code, response.Body.String())
	}
}
