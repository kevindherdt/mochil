package audio

import "testing"

func TestRepeatedStatusDoesNotConflictWithActivePlayback(t *testing.T) {
	store, _ := OpenProgressStore("")
	dismissed := true
	first, err := store.PatchStatus("alice", "podcasts", "episode", 600, nil, &dismissed)
	if err != nil {
		t.Fatal(err)
	}
	version := store.Version()
	store.path = t.TempDir() // A no-op must succeed without trying to replace this directory.
	repeated, err := store.PatchStatus("alice", "podcasts", "episode", 600, nil, &dismissed)
	if err != nil || repeated != first || store.Version() != version {
		t.Fatalf("retry changed state: first=%+v repeated=%+v error=%v", first, repeated, err)
	}
	checkpoint := first
	store.path = ""
	checkpoint.Position = 30
	checkpoint.DeviceID = "active-phone"
	if err := store.PutIfRevision("alice", checkpoint, first.Revision); err != nil {
		t.Fatalf("status retry invalidated playback revision: %v", err)
	}
}
