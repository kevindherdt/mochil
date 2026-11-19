package web

import "testing"

// The server stamps each save with its own clock, so a repeated or stepped-back
// clock must not make a save look older than the one already stored: the device
// would be told it saved while GET kept returning the previous position.
func TestKoSyncProgressStaysMonotonicWhenTheClockRepeats(t *testing.T) {
	store, err := newKoSyncStore("", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ensureUser("reader", "secret"); err != nil {
		t.Fatal(err)
	}
	document := "0123456789abcdef0123456789abcdef"
	first := koSyncProgress{Document: document, Progress: "/body[p1]", Percentage: 0.2, Timestamp: 1000}
	if _, err := store.setProgress("reader", first); err != nil {
		t.Fatal(err)
	}
	same := koSyncProgress{Document: document, Progress: "/body[p2]", Percentage: 0.4, Timestamp: 1000}
	stored, err := store.setProgress("reader", same)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.data.Users["reader"].Progress[document].Progress; got != "/body[p2]" {
		t.Fatalf("stored %q, want the newest save", got)
	}
	if stored.Timestamp <= first.Timestamp {
		t.Fatalf("timestamp %d did not advance past %d", stored.Timestamp, first.Timestamp)
	}
	if got := store.data.Users["reader"].Progress[document].Timestamp; got != stored.Timestamp {
		t.Fatalf("stored timestamp %d differs from the reported %d", got, stored.Timestamp)
	}

	// The same hazard when the clock steps backwards: the LAST save to arrive wins,
	// and it must still be stored rather than silently dropped behind the 200 reply.
	backwards := koSyncProgress{Document: document, Progress: "/body[p3]", Percentage: 0.6, Timestamp: 900}
	stored, err = store.setProgress("reader", backwards)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.data.Users["reader"].Progress[document].Progress; got != "/body[p3]" {
		t.Fatalf("stored %q after a backwards clock, want the newest save", got)
	}
	if stored.Timestamp <= 1000 {
		t.Fatalf("timestamp %d did not advance past the stored 1000", stored.Timestamp)
	}
}
