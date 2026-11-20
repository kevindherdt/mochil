package audio

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeProber struct{ calls int }

func (p *fakeProber) Probe(string) (Metadata, error) {
	p.calls++
	return Metadata{Title: "Tagged title", Artist: "Author", Album: "Series", Track: 2, Duration: 90}, nil
}

func TestStoreScansTagsAndReusesRevision(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Series", "02.mp3")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("audio"), 0o640); err != nil {
		t.Fatal(err)
	}
	probe := &fakeProber{}
	store := NewStore("audio", root, filepath.Join(t.TempDir(), "audio.json"))
	store.SetProber(probe)
	stats, err := store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Items != 1 || probe.calls != 1 {
		t.Fatalf("stats=%+v calls=%d", stats, probe.calls)
	}
	item := store.Items()[0]
	if item.Title != "Tagged title" || item.Artist != "Author" || item.Track != 2 {
		t.Fatalf("item=%+v", item)
	}
	if _, _, err = store.Open(item.ID); err != nil {
		t.Fatal(err)
	}
	stats, err = store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Reused != 1 || probe.calls != 1 {
		t.Fatalf("reuse=%+v calls=%d", stats, probe.calls)
	}
}

func TestStoreSnapshotsDoNotExposeChapterStorage(t *testing.T) {
	store := NewStore("audio", t.TempDir(), "")
	store.publish([]Item{{ID: "one", Chapters: []Chapter{{Title: "Original"}}}})

	items := store.Items()
	snapshot, _ := store.Snapshot()
	item, ok := store.Item("one")
	if !ok {
		t.Fatal("item not found")
	}
	items[0].Chapters[0].Title = "Items mutation"
	snapshot[0].Chapters[0].Title = "Snapshot mutation"
	item.Chapters[0].Title = "Item mutation"

	got, _ := store.Item("one")
	if got.Chapters[0].Title != "Original" {
		t.Fatalf("stored chapters were mutated through a snapshot: %+v", got.Chapters)
	}
}

func TestIdentifyRejectsTraversalAndSupportsAudiobook(t *testing.T) {
	if _, ok := Identify("a", "../escape.mp3", 1, time.Now()); ok {
		t.Fatal("accepted traversal")
	}
	item, ok := Identify("a", "Novel.m4b", 1, time.Now())
	if !ok || item.MediaType != "audio/mp4" {
		t.Fatalf("item=%+v ok=%v", item, ok)
	}
	if _, ok = Identify("a", "Series/._Novel.m4b", 4096, time.Now()); ok {
		t.Fatal("accepted AppleDouble resource fork as an audiobook")
	}
	for path, mediaType := range map[string]string{"speech.oga": "audio/ogg", "chapter.weba": "audio/webm", "chapter.webm": "audio/webm", "archive.wave": "audio/wav"} {
		item, supported := Identify("a", path, 1, time.Now())
		if !supported || item.MediaType != mediaType {
			t.Errorf("Identify(%q)=%+v, %v", path, item, supported)
		}
	}
}

func TestInferBookFolderMetadataUsesExplicitAudiobookLayout(t *testing.T) {
	item := Item{Path: "Rifujin na Magonote/Mushoku Tensei/Book 03.0 Mushoku Tensei Vol. 03/title.m4b"}
	inferBookFolderMetadata(&item)
	if item.Series != "Mushoku Tensei" || item.Artist != "Rifujin na Magonote" || item.Track != 3 || item.SeriesIndex != 3 {
		t.Fatalf("inferred item = %#v", item)
	}
	music := Item{Path: "Artist/Album/03 Song.mp3"}
	inferBookFolderMetadata(&music)
	if music.Series != "" || music.Track != 0 {
		t.Fatalf("ordinary music path was inferred as audiobook: %#v", music)
	}
}

func TestInferBookFolderMetadataPreservesEmbeddedValues(t *testing.T) {
	item := Item{Path: "Folder Author/Folder Series/Volume 08/title.m4b", Series: "Tagged Series", Artist: "Tagged Author", Track: 12}
	inferBookFolderMetadata(&item)
	if item.Series != "Tagged Series" || item.Artist != "Tagged Author" || item.Track != 12 {
		t.Fatalf("embedded metadata was replaced: %#v", item)
	}
}

func TestInferMusicFolderMetadataUsesExplicitLibraryFolders(t *testing.T) {
	album := Item{Path: "Albums/Paramore - Riot!/Paramore - Riot! - 01-01 For a Pessimist.flac"}
	inferMusicFolderMetadata(&album)
	if album.Artist != "Paramore" || album.AlbumArtist != "Paramore" || album.Album != "Riot!" {
		t.Fatalf("inferred album metadata = %#v", album)
	}
	track := Item{Path: "Tracks/Paramore - Decode.flac"}
	inferMusicFolderMetadata(&track)
	if track.Artist != "Paramore" {
		t.Fatalf("inferred track metadata = %#v", track)
	}
	ordinary := Item{Path: "Loose/Someone - Something.flac"}
	inferMusicFolderMetadata(&ordinary)
	if ordinary.Artist != "" || ordinary.Album != "" {
		t.Fatalf("unlabelled path was inferred: %#v", ordinary)
	}
}

func TestInferMusicFolderMetadataFindsNestedAlbumsAndCleansFallbackFilename(t *testing.T) {
	item := Item{
		Path:  "account/Albums/Paramore - All We Know Is Falling/Paramore - All We Know Is Falling - 01-10 My Heart.flac",
		Title: "Paramore - All We Know Is Falling - 01-10 My Heart",
	}
	inferMusicFolderMetadata(&item)
	if item.Artist != "Paramore" || item.AlbumArtist != "Paramore" || item.Album != "All We Know Is Falling" || item.Disc != 1 || item.Track != 10 || item.Title != "My Heart" {
		t.Fatalf("nested album metadata = %#v", item)
	}
}

func TestInferMusicFolderMetadataPreservesEmbeddedTags(t *testing.T) {
	item := Item{Path: "Albums/Folder Artist - Folder Album/Folder Artist - Folder Album - 01-01 Filename.flac", Title: "Tagged title", Artist: "Tagged Artist", AlbumArtist: "Tagged Album Artist", Album: "Tagged Album", Track: 7}
	inferMusicFolderMetadata(&item)
	if item.Title != "Tagged title" || item.Artist != "Tagged Artist" || item.AlbumArtist != "Tagged Album Artist" || item.Album != "Tagged Album" || item.Track != 7 {
		t.Fatalf("embedded metadata was replaced: %#v", item)
	}
}

func TestProgressPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	store, err := OpenProgressStore(path)
	if err != nil {
		t.Fatal(err)
	}
	value := Progress{LibraryID: "a", ItemID: "b", Position: 30, Duration: 100, PlaybackRate: 1.5}
	if err = store.Put("user", value); err != nil {
		t.Fatal(err)
	}
	reloaded, err := OpenProgressStore(path)
	if err != nil {
		t.Fatal(err)
	}
	saved, ok := reloaded.Get("user", "a", "b")
	if !ok || saved.Position != 30 || saved.PlaybackRate != 1.5 {
		t.Fatalf("saved=%+v ok=%v", saved, ok)
	}
}

func TestAudioProgressConditionalWriteRejectsStaleDevice(t *testing.T) {
	store, err := OpenProgressStore("")
	if err != nil {
		t.Fatal(err)
	}
	base := Progress{LibraryID: "a", ItemID: "book", Position: 10, Duration: 100, DeviceID: "phone"}
	if err = store.PutIfRevision("alice", base, 0); err != nil {
		t.Fatal(err)
	}
	saved, _ := store.Get("alice", "a", "book")
	if saved.Revision != 1 {
		t.Fatalf("revision=%d", saved.Revision)
	}
	base.Position, base.DeviceID = 20, "tablet"
	if err = store.PutIfRevision("alice", base, 0); !errors.Is(err, ErrProgressConflict) {
		t.Fatalf("stale write error=%v", err)
	}
	current, _ := store.Get("alice", "a", "book")
	if current.Position != 10 || current.DeviceID != "phone" {
		t.Fatalf("current=%+v", current)
	}
}

func TestAudioProgressAcknowledgesEquivalentStaleCheckpoint(t *testing.T) {
	store, err := OpenProgressStore("")
	if err != nil {
		t.Fatal(err)
	}
	current := Progress{LibraryID: "podcasts", ItemID: "episode", Position: 42, Duration: 600, DeviceID: "tv"}
	if err = store.PutIfRevision("alice", current, 0); err != nil {
		t.Fatal(err)
	}
	stale := current
	stale.Position, stale.DeviceID = 42.2, "phone"
	if err = store.PutIfRevision("alice", stale, 0); err != nil {
		t.Fatalf("equivalent stale checkpoint should be acknowledged: %v", err)
	}
	saved, _ := store.Get("alice", "podcasts", "episode")
	if saved.Revision != 1 || saved.Position != 42 || saved.DeviceID != "tv" {
		t.Fatalf("equivalent acknowledgement changed authoritative progress: %+v", saved)
	}
	stale.Position = 42.3
	if err = store.PutIfRevision("alice", stale, 0); !errors.Is(err, ErrProgressConflict) {
		t.Fatalf("materially different stale checkpoint should conflict: %v", err)
	}
}

func TestAudioProgressRejectsPassiveRewindButAcceptsExplicitSeek(t *testing.T) {
	store, _ := OpenProgressStore("")
	progress := Progress{LibraryID: "music", ItemID: "track", Position: 80, Duration: 100, DeviceID: "phone"}
	if err := store.PutIfRevision("alice", progress, 0); err != nil {
		t.Fatal(err)
	}
	progress.Position, progress.DeviceID = 20, "tablet"
	if err := store.PutIfRevision("alice", progress, 1); !errors.Is(err, ErrProgressConflict) {
		t.Fatalf("passive rewind error=%v", err)
	}
	if err := store.PutIfRevisionExplicit("alice", progress, 1); err != nil {
		t.Fatalf("explicit seek error=%v", err)
	}
	current, _ := store.Get("alice", "music", "track")
	if current.Position != 20 || current.DeviceID != "tablet" || current.Revision != 2 {
		t.Fatalf("current=%+v", current)
	}
}

func TestAudioProgressPassiveWriteCannotReopenFinishedItem(t *testing.T) {
	store, _ := OpenProgressStore("")
	progress := Progress{LibraryID: "podcasts", ItemID: "episode", Position: 100, Duration: 100, Finished: true}
	if err := store.PutIfRevision("alice", progress, 0); err != nil {
		t.Fatal(err)
	}
	progress.Position, progress.Finished = 99, false
	if err := store.PutIfRevision("alice", progress, 1); !errors.Is(err, ErrProgressConflict) {
		t.Fatalf("passive reopen error=%v", err)
	}
}

func TestOldCacheRemainsVisibleDuringProbeMigration(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "track.mp3")
	if err := os.WriteFile(media, []byte("audio"), 0o640); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(media)
	identified, _ := Identify("music", "track.mp3", info.Size(), info.ModTime())
	cache := filepath.Join(t.TempDir(), "audio.json")
	payload := cacheFile{Version: cacheVersion - 1, Items: []Item{identified}}
	file, err := os.Create(cache)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.NewEncoder(file).Encode(payload); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	store := NewStore("music", root, cache)
	if len(store.Items()) != 1 {
		t.Fatal("old snapshot was not published")
	}
	probe := &fakeProber{}
	store.SetProber(probe)
	if _, err = store.Scan(); err != nil {
		t.Fatal(err)
	}
	if probe.calls != 1 || store.Items()[0].ProbeVersion != probeVersion {
		t.Fatalf("calls=%d item=%+v", probe.calls, store.Items()[0])
	}
}
