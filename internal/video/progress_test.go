package video

import (
	"errors"
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
		t.Fatal("video progress with trailing JSON was accepted")
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
	progress := Progress{LibraryID: "movies", ItemID: "film", Position: 5, Duration: 60}
	if err = store.Put("alice", progress); err == nil {
		t.Fatal("write through an invalid state path succeeded")
	}
	if _, exists := store.Get("alice", "movies", "film"); exists {
		t.Fatal("failed write remained visible in memory")
	}
}

func TestProgressPersistenceAndAccountIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	store, err := OpenProgressStore(path)
	if err != nil {
		t.Fatal(err)
	}
	progress := Progress{LibraryID: "movies", ItemID: "film", Position: 125, Duration: 1000, DeviceID: "phone", Dismissed: true}
	if err := store.Put("alice", progress); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get("bob", "movies", "film"); ok {
		t.Fatal("progress leaked across accounts")
	}
	reopened, err := OpenProgressStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Get("alice", "movies", "film")
	if !ok || got.Position != 125 || got.DeviceID != "phone" || !got.Dismissed {
		t.Fatalf("persisted progress = %#v, %v", got, ok)
	}
	if err := reopened.Delete("alice", "movies", "film"); err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Get("alice", "movies", "film"); ok {
		t.Fatal("deleted progress remained")
	}
}

func TestVideoProgressConditionalWriteRejectsStaleDevice(t *testing.T) {
	store, _ := OpenProgressStore("")
	progress := Progress{LibraryID: "movies", ItemID: "film", Position: 10, Duration: 100, DeviceID: "phone"}
	if err := store.PutIfRevision("alice", progress, 0); err != nil {
		t.Fatal(err)
	}
	progress.Position, progress.DeviceID = 25, "tablet"
	if err := store.PutIfRevision("alice", progress, 0); !errors.Is(err, ErrProgressConflict) {
		t.Fatalf("stale write error=%v", err)
	}
	current, _ := store.Get("alice", "movies", "film")
	if current.Position != 10 || current.Revision != 1 {
		t.Fatalf("current=%+v", current)
	}
}

func TestVideoProgressEquivalentStaleCheckpointIsIdempotent(t *testing.T) {
	store, _ := OpenProgressStore("")
	current := Progress{LibraryID: "movies", ItemID: "film", Position: 120.2, Duration: 1000, DeviceID: "tv"}
	if err := store.PutIfRevision("alice", current, 0); err != nil {
		t.Fatal(err)
	}
	stale := current
	stale.Position, stale.DeviceID = 120.0, "phone"
	if err := store.PutIfRevision("alice", stale, 0); err != nil {
		t.Fatalf("equivalent stale checkpoint should be acknowledged: %v", err)
	}
	stored, _ := store.Get("alice", "movies", "film")
	if stored.Position != 120.2 || stored.DeviceID != "tv" || stored.Revision != 1 {
		t.Fatalf("idempotent checkpoint replaced authoritative state: %+v", stored)
	}
}

func TestVideoProgressEquivalentPositionCannotChangeFinishedState(t *testing.T) {
	store, _ := OpenProgressStore("")
	current := Progress{LibraryID: "anime", ItemID: "episode", Position: 50, Duration: 100, Finished: true}
	if err := store.PutIfRevision("alice", current, 0); err != nil {
		t.Fatal(err)
	}
	stale := current
	stale.Finished = false
	if err := store.PutIfRevision("alice", stale, 0); !errors.Is(err, ErrProgressConflict) {
		t.Fatalf("finished-state change must still conflict: %v", err)
	}
}

func TestVideoProgressRejectsPassiveRewindButAcceptsExplicitSeek(t *testing.T) {
	store, _ := OpenProgressStore("")
	progress := Progress{LibraryID: "movies", ItemID: "film", Position: 800, Duration: 1000, DeviceID: "tv"}
	if err := store.PutIfRevision("alice", progress, 0); err != nil {
		t.Fatal(err)
	}
	progress.Position, progress.DeviceID = 200, "phone"
	if err := store.PutIfRevision("alice", progress, 1); !errors.Is(err, ErrProgressConflict) {
		t.Fatalf("passive rewind error=%v", err)
	}
	if err := store.PutIfRevisionExplicit("alice", progress, 1); err != nil {
		t.Fatalf("explicit seek error=%v", err)
	}
	current, _ := store.Get("alice", "movies", "film")
	if current.Position != 200 || current.DeviceID != "phone" || current.Revision != 2 {
		t.Fatalf("current=%+v", current)
	}
}

func TestVideoProgressPassiveWriteCannotReopenFinishedItem(t *testing.T) {
	store, _ := OpenProgressStore("")
	progress := Progress{LibraryID: "anime", ItemID: "episode", Position: 100, Duration: 100, Finished: true}
	if err := store.PutIfRevision("alice", progress, 0); err != nil {
		t.Fatal(err)
	}
	progress.Position, progress.Finished = 99, false
	if err := store.PutIfRevision("alice", progress, 1); !errors.Is(err, ErrProgressConflict) {
		t.Fatalf("passive reopen error=%v", err)
	}
}

func TestProgressAutomaticallyFinishesAtNinetyFivePercent(t *testing.T) {
	store, _ := OpenProgressStore("")
	if err := store.Put("alice", Progress{LibraryID: "movies", ItemID: "film", Position: 95, Duration: 100}); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get("alice", "movies", "film")
	if !got.Finished {
		t.Fatal("expected completed progress")
	}
}

func TestPatchStatusPreservesNewerPlaybackPosition(t *testing.T) {
	store, _ := OpenProgressStore("")
	if err := store.Put("alice", Progress{LibraryID: "anime", ItemID: "episode", Position: 900, Duration: 1500, DeviceID: "tv"}); err != nil {
		t.Fatal(err)
	}
	dismissed := true
	patched, err := store.PatchStatus("alice", "anime", "episode", 1500, nil, &dismissed)
	if err != nil {
		t.Fatal(err)
	}
	if patched.Position != 900 || patched.DeviceID != "tv" || !patched.Dismissed || patched.Revision != 2 {
		t.Fatalf("patched progress = %+v", patched)
	}
}

func TestPatchWatchedCreatesAtomicCompletedProgress(t *testing.T) {
	store, _ := OpenProgressStore("")
	watched := true
	patched, err := store.PatchStatus("alice", "movies", "film", 7200, &watched, nil)
	if err != nil || patched.Position != 7200 || !patched.Finished || patched.Revision != 1 {
		t.Fatalf("watched progress = %+v, %v", patched, err)
	}
}
