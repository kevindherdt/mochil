package web

import "testing"

// An update queued while a digest pass is finishing was never replayed against
// the table it is now looking at, so the pass must ask for another one instead
// of leaving it for the next minute.
func TestKoSyncRefreshAsksForAnotherPassWhenAnUpdateArrivesDuringReplay(t *testing.T) {
	server := &multiServer{}
	var queued bool
	koSyncReplayHook = func() {
		if queued {
			return
		}
		queued = true
		server.noteUnmatchedKoSync("alice", koSyncProgress{Document: "0123456789abcdef0123456789abcdef", Progress: "10"})
	}
	defer func() { koSyncReplayHook = nil }()

	server.refreshKoSyncDigests()

	server.koSyncContent.mu.Lock()
	refreshed := server.koSyncContent.refreshed
	waiting := len(server.koSyncContent.waiting)
	server.koSyncContent.mu.Unlock()
	if waiting != 1 {
		t.Fatalf("waiting = %d, want the update queued during the pass", waiting)
	}
	if !refreshed.IsZero() {
		t.Fatal("refreshed was left current, so the queued update waits a full refresh interval")
	}

	// The follow-up pass must not schedule itself again: retrying forever is the
	// failure mode this counter exists to avoid.
	server.refreshKoSyncDigests()
	server.koSyncContent.mu.Lock()
	refreshed = server.koSyncContent.refreshed
	server.koSyncContent.mu.Unlock()
	if refreshed.IsZero() {
		t.Fatal("the follow-up pass asked for yet another pass, so every unmatched update would loop")
	}
}
