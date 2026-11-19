package audio

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAudioStateOversizePreservesReadableFileAndCleansTemporary(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "state.json")
	initial := map[string]string{"value": "saved"}
	if err := saveAudioState(path, ".audio-test-*", maximumAudioStateBytes, initial); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveAudioState(path, ".audio-test-*", maximumAudioStateBytes, map[string]string{"value": strings.Repeat("x", maximumAudioStateBytes)}); err == nil {
		t.Fatal("accepted state larger than decoder limit")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("rejected save replaced existing state")
	}
	var decoded map[string]string
	if err := decodeAudioState(path, &decoded); err != nil || decoded["value"] != "saved" {
		t.Fatalf("saved state not reloadable: %v, %v", decoded, err)
	}
	files, err := os.ReadDir(directory)
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary file leaked: %v, %v", files, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("private state permissions: %v", info.Mode())
	}
}

func TestOversizeAudioCollectionUpdatesDoNotChangeMemory(t *testing.T) {
	entry := QueueEntry{"music", "one"}
	oversize := []QueueEntry{{"music", strings.Repeat("x", maximumAudioStateBytes)}}
	queue, _ := OpenQueueStore(filepath.Join(t.TempDir(), "queue.json"))
	if err := queue.Put("alice", []QueueEntry{entry}); err != nil {
		t.Fatal(err)
	}
	if err := queue.Put("alice", oversize); err == nil {
		t.Fatal("queue accepted oversized state")
	}
	if got := queue.Get("alice"); len(got) != 1 || got[0] != entry {
		t.Fatal("queue memory changed")
	}
	playlists, _ := OpenPlaylistStore(filepath.Join(t.TempDir(), "playlists.json"))
	if err := playlists.Put("alice", "Favorites", []QueueEntry{entry}); err != nil {
		t.Fatal(err)
	}
	if err := playlists.Put("alice", "Favorites", oversize); err == nil {
		t.Fatal("playlist accepted oversized state")
	}
	if got := playlists.List("alice"); len(got) != 1 || len(got[0].Entries) != 1 || got[0].Entries[0] != entry {
		t.Fatal("playlist memory changed")
	}
}

func TestProgressSaveRetainsItsLargerByteBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	store, err := OpenProgressStore(path)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the actual progress writer above the collection-specific limit.
	value := Progress{LibraryID: "music", ItemID: "one", DeviceID: strings.Repeat("x", maximumAudioStateBytes)}
	if err := store.Put("alice", value); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenProgressStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Get("alice", "music", "one")
	if !ok || got.DeviceID != value.DeviceID {
		t.Fatal("progress writer used the smaller collection budget")
	}
}
