package audio

import "testing"

func TestIdenticalCheckpointDoesNotPersist(t *testing.T) {
	for _, mode := range []string{"unconditional", "conditional", "explicit"} {
		t.Run(mode, func(t *testing.T) {
			store, _ := OpenProgressStore("")
			initial := Progress{LibraryID: "library", ItemID: "item", Position: 30, Duration: 600, DeviceID: "phone"}
			if err := store.Put("alice", initial); err != nil {
				t.Fatal(err)
			}
			first, _ := store.Get("alice", "library", "item")
			version := store.Version()
			store.path = t.TempDir() // Replacing a directory would fail if persistence were attempted.
			retry := initial
			retry.UpdatedAt, retry.Revision = -1, 999
			var err error
			switch mode {
			case "unconditional":
				err = store.Put("alice", retry)
			case "conditional":
				err = store.PutIfRevision("alice", retry, first.Revision)
			case "explicit":
				err = store.PutIfRevisionExplicit("alice", retry, first.Revision)
			}
			if err != nil {
				t.Fatal(err)
			}
			got, _ := store.Get("alice", "library", "item")
			if got != first {
				t.Fatalf("retry changed state: %+v -> %+v", first, got)
			}
			if store.Version() != version {
				t.Fatal("retry invalidated cache version")
			}
		})
	}
}

func TestCheckpointStateChangesStillPersist(t *testing.T) {
	changes := map[string]func(*Progress){
		"position":  func(p *Progress) { p.Position += .01 },
		"duration":  func(p *Progress) { p.Duration++ },
		"device":    func(p *Progress) { p.DeviceID = "tv" },
		"dismissed": func(p *Progress) { p.Dismissed = true },
		"finished":  func(p *Progress) { p.Finished = true },
		"rate":      func(p *Progress) { p.PlaybackRate = 1.5 },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			store, _ := OpenProgressStore("")
			initial := Progress{LibraryID: "library", ItemID: "item", Position: 30, Duration: 600, DeviceID: "phone"}
			if err := store.Put("alice", initial); err != nil {
				t.Fatal(err)
			}
			first, _ := store.Get("alice", "library", "item")
			next := first
			change(&next)
			if err := store.PutIfRevision("alice", next, first.Revision); err != nil {
				t.Fatal(err)
			}
			got, _ := store.Get("alice", "library", "item")
			next.Revision, next.UpdatedAt = got.Revision, got.UpdatedAt
			if got != next || got.Revision != first.Revision+1 {
				t.Fatalf("state change not saved: %+v", got)
			}
		})
	}
}
