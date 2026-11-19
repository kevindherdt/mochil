package audio

import "testing"

func TestPlaylistStorePersistsNamedLists(t *testing.T) {
	path := t.TempDir() + "/playlists.json"
	store, err := OpenPlaylistStore(path)
	if err != nil {
		t.Fatal(err)
	}
	entries := []QueueEntry{{LibraryID: "music", ItemID: "one"}, {LibraryID: "music", ItemID: "two"}}
	if err = store.Put("reader", "Evening", entries); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenPlaylistStore(path)
	if err != nil {
		t.Fatal(err)
	}
	lists := reopened.List("reader")
	if len(lists) != 1 || lists[0].Name != "Evening" || len(lists[0].Entries) != 2 {
		t.Fatalf("unexpected playlists: %#v", lists)
	}
	if err = reopened.Remove("reader", "evening"); err != nil {
		t.Fatal(err)
	}
	if len(reopened.List("reader")) != 0 {
		t.Fatal("playlist was not removed")
	}
}

func TestPlaylistStoreFailedPersistenceDoesNotChangeMemory(t *testing.T) {
	store := &PlaylistStore{path: "/dev/null/playlists.json", data: playlistData{Version: 1, Accounts: map[string][]Playlist{
		"reader": {{Name: "Saved", Entries: []QueueEntry{{LibraryID: "music", ItemID: "one"}}}},
	}}}
	if err := store.Put("reader", "Saved", []QueueEntry{{LibraryID: "music", ItemID: "two"}}); err == nil {
		t.Fatal("expected persistence error")
	}
	lists := store.List("reader")
	if len(lists) != 1 || lists[0].Entries[0].ItemID != "one" {
		t.Fatalf("playlist changed: %#v", lists)
	}
}

func TestPlaylistStoreValidatesEntries(t *testing.T) {
	store, err := OpenPlaylistStore(t.TempDir() + "/playlists.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("reader", "Broken", []QueueEntry{{LibraryID: "", ItemID: "one"}}); err == nil {
		t.Fatal("invalid entry accepted")
	}
}

func TestPlaylistStoreAddsAtomicallyWithoutDuplicates(t *testing.T) {
	store, err := OpenPlaylistStore(t.TempDir() + "/playlists.json")
	if err != nil {
		t.Fatal(err)
	}
	first := QueueEntry{LibraryID: "music", ItemID: "one"}
	second := QueueEntry{LibraryID: "music", ItemID: "two"}
	if err = store.Add("reader", "Evening", first); err != nil {
		t.Fatal(err)
	}
	if err = store.Add("reader", "evening", second); err != nil {
		t.Fatal(err)
	}
	if err = store.Add("reader", "Evening", first); err != nil {
		t.Fatal(err)
	}
	lists := store.List("reader")
	if len(lists) != 1 || len(lists[0].Entries) != 2 || lists[0].Entries[1] != second {
		t.Fatalf("unexpected playlist: %#v", lists)
	}
}

func TestPlaylistStoreRenamesWithoutLosingEntries(t *testing.T) {
	store, err := OpenPlaylistStore(t.TempDir() + "/playlists.json")
	if err != nil {
		t.Fatal(err)
	}
	entry := QueueEntry{LibraryID: "music", ItemID: "one"}
	if err = store.Add("reader", "Old", entry); err != nil {
		t.Fatal(err)
	}
	if err = store.Rename("reader", "old", "New"); err != nil {
		t.Fatal(err)
	}
	lists := store.List("reader")
	if len(lists) != 1 || lists[0].Name != "New" || len(lists[0].Entries) != 1 || lists[0].Entries[0] != entry {
		t.Fatalf("unexpected renamed playlist: %#v", lists)
	}
}
