package audio

import (
	"math"
	"testing"
)

func TestProgressRejectsNonFiniteTimesWithoutChangingState(t *testing.T) {
	store, _ := OpenProgressStore("")
	original := Progress{LibraryID: "podcasts", ItemID: "episode", Position: 20, Duration: 100}
	if err := store.Put("alice", original); err != nil {
		t.Fatal(err)
	}
	saved, _ := store.Get("alice", "podcasts", "episode")
	version := store.Version()
	for _, number := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, field := range []string{"position", "duration"} {
			invalid := original
			if field == "position" {
				invalid.Position = number
			} else {
				invalid.Duration = number
			}
			if err := store.Put("alice", invalid); err == nil {
				t.Fatalf("accepted non-finite %s", field)
			}
		}
		finished := true
		if _, err := store.PatchStatus("alice", "podcasts", "episode", number, &finished, nil); err == nil {
			t.Fatal("accepted non-finite status duration")
		}
	}
	current, _ := store.Get("alice", "podcasts", "episode")
	if current != saved || store.Version() != version {
		t.Fatal("invalid input changed progress")
	}
}

func TestNonFinitePlaybackRateUsesNormalSpeed(t *testing.T) {
	store, _ := OpenProgressStore("")
	if err := store.Put("alice", Progress{LibraryID: "music", ItemID: "track", Duration: 60, PlaybackRate: math.NaN()}); err != nil {
		t.Fatal(err)
	}
	saved, _ := store.Get("alice", "music", "track")
	if saved.PlaybackRate != 1 {
		t.Fatalf("rate = %v", saved.PlaybackRate)
	}
}
