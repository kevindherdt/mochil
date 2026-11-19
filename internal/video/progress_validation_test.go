package video

import (
	"math"
	"testing"
)

func TestProgressRejectsNonFiniteTimesWithoutChangingState(t *testing.T) {
	store, _ := OpenProgressStore("")
	original := Progress{LibraryID: "movies", ItemID: "movie", Position: 20, Duration: 100}
	if err := store.Put("alice", original); err != nil {
		t.Fatal(err)
	}
	saved, _ := store.Get("alice", "movies", "movie")
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
		if _, err := store.PatchStatus("alice", "movies", "movie", number, &finished, nil); err == nil {
			t.Fatal("accepted non-finite status duration")
		}
	}
	current, _ := store.Get("alice", "movies", "movie")
	if current != saved {
		t.Fatal("invalid input changed progress")
	}
}
