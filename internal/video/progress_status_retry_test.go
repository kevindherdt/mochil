package video

import "testing"

func TestRepeatedStatusDoesNotConflictWithActivePlayback(t *testing.T) {
	store, _ := OpenProgressStore("")
	dismissed := true
	first, err := store.PatchStatus("alice", "movies", "movie", 600, nil, &dismissed)
	if err != nil {
		t.Fatal(err)
	}
	store.path = t.TempDir() // A no-op must not attempt persistence.
	repeated, err := store.PatchStatus("alice", "movies", "movie", 600, nil, &dismissed)
	if err != nil || repeated != first {
		t.Fatalf("retry changed state: first=%+v repeated=%+v error=%v", first, repeated, err)
	}
	checkpoint := first
	store.path = ""
	checkpoint.Position = 30
	checkpoint.DeviceID = "active-tv"
	if err := store.PutIfRevision("alice", checkpoint, first.Revision); err != nil {
		t.Fatalf("status retry invalidated playback revision: %v", err)
	}
}
