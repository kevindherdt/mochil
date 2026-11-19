package video

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type blockingProber struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int64
}

func (p *blockingProber) Probe(string) (ProbeResult, error) {
	if p.calls.Add(1) == 1 {
		close(p.started)
	}
	<-p.release
	return ProbeResult{Duration: 1, Video: []VideoStream{{Index: 0, Codec: "h264"}}}, nil
}

func cachedCount(cache string) int {
	store := NewStore("movies", filepath.Join(filepath.Dir(cache), "missing-root"), cache)
	_, _ = store.Scan()
	return store.Count()
}

func TestStoreCacheRequiresOneExactBoundedJSONDocument(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "video.json")
	valid := `{"version":2,"items":[{"id":"one","library_id":"movies","path":"one.mp4","title":"One","container":"mp4","media_type":"video/mp4","direct_play":true,"size":1}]}`
	if err := os.WriteFile(cache, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	if count := cachedCount(cache); count != 1 {
		t.Fatalf("valid cache count = %d", count)
	}
	if err := os.WriteFile(cache, []byte(valid+` {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if count := cachedCount(cache); count != 0 {
		t.Fatalf("cache with trailing JSON count = %d", count)
	}
}

func TestStorePublishesVideoSnapshotAndReusesUnchangedItems(t *testing.T) {
	root, cache := t.TempDir(), filepath.Join(t.TempDir(), "movies.json")
	for _, name := range []string{"Movie 10.mp4", "Movie 2.mkv", "ignore.epub"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore("movies", root, cache)
	stats, err := store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Items != 2 || len(store.Items()) != 2 {
		t.Fatalf("stats/items = %#v/%d", stats, len(store.Items()))
	}
	if store.Items()[0].Title != "Movie 2" || store.Items()[1].Title != "Movie 10" {
		t.Fatalf("items are not naturally ordered: %#v", store.Items())
	}
	firstVersion := store.Version()
	stats, err = store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Reused != 2 || store.Version() <= firstVersion {
		t.Fatalf("rescan = %#v version=%d", stats, store.Version())
	}
}

func TestStoreSnapshotsDoNotExposeStreamStorage(t *testing.T) {
	store := NewStore("movies", t.TempDir(), "")
	store.publish([]Item{{
		ID:        "one",
		Video:     []VideoStream{{Codec: "h264"}},
		Audio:     []AudioStream{{Language: "eng"}},
		Subtitles: []SubtitleStream{{Language: "eng"}},
	}})

	items := store.Items()
	item, ok := store.Item("one")
	if !ok {
		t.Fatal("item not found")
	}
	items[0].Video[0].Codec = "mutated"
	items[0].Audio[0].Language = "mutated"
	item.Subtitles[0].Language = "mutated"

	got, _ := store.Item("one")
	if got.Video[0].Codec != "h264" || got.Audio[0].Language != "eng" || got.Subtitles[0].Language != "eng" {
		t.Fatalf("stored streams were mutated through a snapshot: %+v", got)
	}
}

func TestStoreSerializesConcurrentScans(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Film.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	prober := &blockingProber{started: make(chan struct{}), release: make(chan struct{})}
	store := NewStore("movies", root, filepath.Join(t.TempDir(), "movies.json"))
	store.SetProber(prober)
	results := make(chan error, 2)
	go func() { _, err := store.Scan(); results <- err }()
	<-prober.started
	go func() { _, err := store.Scan(); results <- err }()
	time.Sleep(20 * time.Millisecond)
	if calls := prober.calls.Load(); calls != 1 {
		t.Fatalf("probe calls while first scan active = %d", calls)
	}
	close(prober.release)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if calls := prober.calls.Load(); calls != 1 {
		t.Fatalf("unchanged item was reprobed after serialized scan: %d calls", calls)
	}
}

func TestStoreRetainsLastKnownGoodSnapshotWhenRootDisappears(t *testing.T) {
	parent, cache := t.TempDir(), filepath.Join(t.TempDir(), "movies.json")
	root := filepath.Join(parent, "movies")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Film.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("movies", root, cache)
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, root+".offline"); err != nil {
		t.Fatal(err)
	}
	stats, err := store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Reused != 1 || stats.Errors != 1 || len(store.Items()) != 1 {
		t.Fatalf("last-known-good snapshot was not retained: %#v %#v", stats, store.Items())
	}
}

func TestStoreLoadsPersistentSnapshotWhenSourceStartsOffline(t *testing.T) {
	parent, cache := t.TempDir(), filepath.Join(t.TempDir(), "movies.json")
	root := filepath.Join(parent, "movies")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Film.webm"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := NewStore("movies", root, cache)
	if _, err := writer.Scan(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, root+".offline"); err != nil {
		t.Fatal(err)
	}
	reloaded := NewStore("movies", root, cache)
	stats, err := reloaded.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Reused != 1 || len(reloaded.Items()) != 1 {
		t.Fatalf("cached startup = %#v %#v", stats, reloaded.Items())
	}
}

func TestStoreDoesNotPublishPartialSnapshotAfterWalkFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission failure cannot be asserted as root")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Film.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("movies", root, filepath.Join(t.TempDir(), "movies.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(blocked, 0o700)
	if _, err := store.Scan(); err == nil {
		t.Fatal("scan unexpectedly succeeded through unreadable directory")
	}
	if len(store.Items()) != 1 {
		t.Fatal("failed scan replaced last-known-good snapshot")
	}
}

// A single-item lookup cannot use Items(): that clones every record. It needs the
// keys only, so IDs() has to list all of them without touching item data.
func TestIDsListsEveryItemWithoutCopyingRecords(t *testing.T) {
	store := NewStore("movies", t.TempDir(), "")
	store.publish([]Item{
		{ID: "one", Title: "One", Video: []VideoStream{{Codec: "h264"}}},
		{ID: "two", Title: "Two", Video: []VideoStream{{Codec: "hevc"}}},
	})
	ids := store.IDs()
	if len(ids) != store.Count() {
		t.Fatalf("IDs() = %d entries, Count() = %d", len(ids), store.Count())
	}
	seen := map[string]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	if !seen["one"] || !seen["two"] {
		t.Fatalf("IDs() = %#v", ids)
	}
	// The listing is a copy: mutating it must not disturb the store.
	ids[0] = "changed"
	if _, ok := store.Item("changed"); ok {
		t.Fatal("mutating the id list changed the store")
	}
}
