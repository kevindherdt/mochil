package audio

import (
	"path/filepath"
	"testing"
)

func TestRemapCarriesQueueFavouritesPlaylistsAndProgressToTheNewId(t *testing.T) {
	dir := t.TempDir()
	queue, _ := OpenQueueStore(filepath.Join(dir, "q.json"))
	lists, _ := OpenPlaylistStore(filepath.Join(dir, "p.json"))
	progress, _ := OpenProgressStore(filepath.Join(dir, "g.json"))
	old := QueueEntry{LibraryID: "music", ItemID: "old"}
	other := QueueEntry{LibraryID: "music", ItemID: "other"}
	elsewhere := QueueEntry{LibraryID: "audiobooks", ItemID: "old"}
	if err := queue.Put("kevin", []QueueEntry{other, old, elsewhere}); err != nil {
		t.Fatal(err)
	}
	if err := lists.Put("kevin", "Mix", []QueueEntry{old, other}); err != nil {
		t.Fatal(err)
	}
	if err := progress.Put("kevin", Progress{LibraryID: "music", ItemID: "old", Position: 42, Duration: 200, UpdatedAt: 5}); err != nil {
		t.Fatal(err)
	}
	ids := Remap{"old": "new"}
	for _, err := range []error{queue.Remap("music", ids), lists.Remap("music", ids), progress.Remap("music", ids)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := queue.Get("kevin"); got[0] != other || got[1].ItemID != "new" || got[2] != elsewhere {
		t.Fatalf("queue = %+v (another library's entry with the same id must stay)", got)
	}
	if list, _ := lists.Get("kevin", "Mix"); list.Entries[0].ItemID != "new" || list.Entries[1] != other {
		t.Fatalf("playlist = %+v", list)
	}
	if p, ok := progress.Get("kevin", "music", "new"); !ok || p.Position != 42 || p.ItemID != "new" {
		t.Fatalf("progress = %+v %v", p, ok)
	}
	if _, ok := progress.Get("kevin", "music", "old"); ok {
		t.Fatal("the old progress must be gone")
	}
	// undo restores everything, and the saved files agree after a reopen
	back := ids.Reverse()
	_ = queue.Remap("music", back)
	_ = lists.Remap("music", back)
	_ = progress.Remap("music", back)
	reopened, _ := OpenQueueStore(filepath.Join(dir, "q.json"))
	if got := reopened.Get("kevin"); got[1] != old {
		t.Fatalf("undo = %+v", got)
	}
	if p, ok := progress.Get("kevin", "music", "old"); !ok || p.Position != 42 {
		t.Fatal("undo must bring the progress back")
	}
}

func TestRemapKeepsTheMoreRecentProgressWhenTheNewIdAlreadyHasSome(t *testing.T) {
	progress, _ := OpenProgressStore(filepath.Join(t.TempDir(), "g.json"))
	for _, p := range []Progress{{LibraryID: "music", ItemID: "old", Position: 10, Duration: 100, UpdatedAt: 1}, {LibraryID: "music", ItemID: "new", Position: 99, Duration: 100, UpdatedAt: 9}} {
		if err := progress.Put("kevin", p); err != nil {
			t.Fatal(err)
		}
	}
	if err := progress.Remap("music", Remap{"old": "new"}); err != nil {
		t.Fatal(err)
	}
	if p, _ := progress.Get("kevin", "music", "new"); p.Position != 99 {
		t.Fatalf("%+v", p)
	}
}
