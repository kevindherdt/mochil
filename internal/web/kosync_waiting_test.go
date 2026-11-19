package web

import "testing"

func TestPendingKoSyncKeepsOneUpdatePerUserAndDocument(t *testing.T) {
	server := &multiServer{}
	server.noteUnmatchedKoSync("alice", koSyncProgress{Document: "doc", Progress: "10"})
	server.noteUnmatchedKoSync("bob", koSyncProgress{Document: "doc", Progress: "20"})
	if len(server.koSyncContent.waiting) != 2 {
		t.Fatalf("waiting = %d, want 2: two users on one document must not overwrite each other", len(server.koSyncContent.waiting))
	}
	server.noteUnmatchedKoSync("alice", koSyncProgress{Document: "doc", Progress: "30"})
	if len(server.koSyncContent.waiting) != 2 {
		t.Fatalf("waiting = %d, want 2: a repeat from one user must replace that user's entry", len(server.koSyncContent.waiting))
	}
	for key, update := range server.koSyncContent.waiting {
		if update.username == "alice" && update.progress.Progress != "30" {
			t.Fatalf("alice entry %q kept %q, want the newer update", key, update.progress.Progress)
		}
	}
}
