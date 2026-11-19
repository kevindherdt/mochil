package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenProgressStoreRejectsMalformedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"accounts":{}} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgressStore(path); err == nil {
		t.Fatal("audio progress with trailing JSON was accepted")
	}
}

func TestProgressStoreRollsBackFailedPersistence(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenProgressStore("")
	if err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "progress.json")
	progress := Progress{LibraryID: "music", ItemID: "track", Position: 5, Duration: 60}
	if err = store.Put("alice", progress); err == nil {
		t.Fatal("write through an invalid state path succeeded")
	}
	if _, exists := store.Get("alice", "music", "track"); exists {
		t.Fatal("failed write remained visible in memory")
	}
	if store.Version() != 1 {
		t.Fatalf("failed write advanced version to %d", store.Version())
	}
}

func TestPatchStatusPreservesNewerPlaybackPosition(t *testing.T) {
	store, _ := OpenProgressStore("")
	if err := store.Put("alice", Progress{LibraryID: "podcasts", ItemID: "episode", Position: 125, Duration: 600, DeviceID: "phone"}); err != nil {
		t.Fatal(err)
	}
	dismissed := true
	patched, err := store.PatchStatus("alice", "podcasts", "episode", 600, nil, &dismissed)
	if err != nil {
		t.Fatal(err)
	}
	if patched.Position != 125 || patched.DeviceID != "phone" || !patched.Dismissed || patched.Revision != 2 {
		t.Fatalf("patched progress = %+v", patched)
	}
}

func TestPatchPlayedCreatesAndResetsStatusWithoutStaleSnapshot(t *testing.T) {
	store, _ := OpenProgressStore("")
	played := true
	patched, err := store.PatchStatus("alice", "podcasts", "episode", 600, &played, nil)
	if err != nil || patched.Position != 600 || !patched.Finished {
		t.Fatalf("played = %+v, %v", patched, err)
	}
	played = false
	patched, err = store.PatchStatus("alice", "podcasts", "episode", 600, &played, nil)
	if err != nil || patched.Position != 0 || patched.Finished {
		t.Fatalf("unplayed = %+v, %v", patched, err)
	}
}
