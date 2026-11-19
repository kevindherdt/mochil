package audio

import "testing"

// A passive checkpoint at or above the 98% completion threshold auto-marks the
// playback finished so "continue watching" stops surfacing it. The boundary is
// deliberately tight: a checkpoint just below it must NOT complete.
func TestPassiveCheckpointAt98PercentAutoCompletes(t *testing.T) {
	store, _ := OpenProgressStore("")
	if err := store.Put("alice", Progress{LibraryID: "podcasts", ItemID: "episode", Position: 0, Duration: 600, DeviceID: "phone"}); err != nil {
		t.Fatal(err)
	}
	// 588/600 == 0.98 exactly -> completes.
	if err := store.Put("alice", Progress{LibraryID: "podcasts", ItemID: "episode", Position: 588, Duration: 600, DeviceID: "phone"}); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get("alice", "podcasts", "episode")
	if !got.Finished {
		t.Fatalf("98%% checkpoint did not auto-complete: %+v", got)
	}
}

func TestPassiveCheckpointJustBelow98PercentStaysFinishedFalse(t *testing.T) {
	store, _ := OpenProgressStore("")
	if err := store.Put("alice", Progress{LibraryID: "podcasts", ItemID: "episode", Position: 0, Duration: 600, DeviceID: "phone"}); err != nil {
		t.Fatal(err)
	}
	// 587/600 == 0.9783... < 0.98 -> does not complete, but position still advances.
	if err := store.Put("alice", Progress{LibraryID: "podcasts", ItemID: "episode", Position: 587, Duration: 600, DeviceID: "phone"}); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get("alice", "podcasts", "episode")
	if got.Finished {
		t.Fatalf("sub-threshold checkpoint wrongly marked finished: %+v", got)
	}
	if got.Position != 587 {
		t.Fatalf("position did not advance to 587: %+v", got)
	}
}
