package video

import (
	"os"
	"path/filepath"
	"testing"
)

// A probe's answers can be wrong or incomplete; when the probe is fixed, the items already in the cache have to be asked
// again. Reuse used to key on the file revision alone, so nothing was ever re-probed without touching the file itself.
func TestScanReprobesItemsStampedByAnOlderProbe(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Film.mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	prober := &countingProber{}
	store := NewStore("movies", root, filepath.Join(t.TempDir(), "movies.json"))
	store.SetProber(prober)
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	if prober.calls != 1 {
		t.Fatalf("first scan probed %d times, want 1", prober.calls)
	}
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	if prober.calls != 1 {
		t.Fatalf("a scan re-probed an item that was already current (%d calls)", prober.calls)
	}
	// An item stamped by an older probe version must be probed again even though its revision is unchanged.
	store.probeStaleForTest()
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	if prober.calls != 2 {
		t.Fatalf("an item with an old probe stamp was reused (%d calls, want 2)", prober.calls)
	}
}
