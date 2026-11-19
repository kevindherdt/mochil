package catalog

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSummaryDescribesThePublishedCatalogueWithoutCopyingIt(t *testing.T) {
	store := NewStore(t.TempDir(), filepath.Join(t.TempDir(), "cache.json"))
	if count, latest := store.Summary(); count != 0 || !latest.IsZero() {
		t.Fatalf("empty store summary = %d %v", count, latest)
	}
	older := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)
	store.replace([]Book{
		{ID: "a", Title: "A", Modified: older, Authors: []string{"Writer"}},
		{ID: "b", Title: "B", Modified: newer},
	})
	if count, latest := store.Summary(); count != 2 || !latest.Equal(newer) {
		t.Fatalf("summary = %d %v, want 2 %v", count, latest, newer)
	}

	// A published snapshot belongs to its generation: the next publish must
	// replace it, never edit it, or a handler reading it would see the
	// catalogue shift underneath.
	held := store.Snapshot()
	if len(held) != 2 {
		t.Fatalf("snapshot has %d books", len(held))
	}
	store.replace([]Book{{ID: "c", Title: "C", Modified: older}})
	if len(held) != 2 || held[0].ID != "a" {
		t.Fatalf("the held snapshot was modified in place: %+v", held)
	}
	if count, latest := store.Summary(); count != 1 || !latest.Equal(older) {
		t.Fatalf("summary after replace = %d %v", count, latest)
	}
	if len(store.Snapshot()) != 1 || store.Snapshot()[0].ID != "c" {
		t.Fatalf("snapshot does not describe the new generation: %+v", store.Snapshot())
	}
}
