package health

import (
	"testing"
	"time"
)

func TestReportIsSortedCopiedAndDegraded(t *testing.T) {
	services := map[string]bool{"wiki": true}
	s := New(func() Snapshot {
		return Snapshot{Shelves: []Shelf{{ID: "z", Ready: true, Items: 2}, {ID: "a", Ready: false}}, Services: services, Jobs: map[string]JobQueue{"scan": {Running: 1}}}
	})
	s.started = time.Unix(100, 0)
	s.now = func() time.Time { return time.Unix(112, 0) }
	r := s.Report()
	if r.Status != "degraded" || r.UptimeSeconds != 12 || r.Shelves[0].ID != "a" {
		t.Fatalf("report: %#v", r)
	}
	r.Services["wiki"] = false
	if !services["wiki"] {
		t.Fatal("report leaked service map")
	}
	r.Jobs["scan"] = JobQueue{}
	if s.Report().Jobs["scan"].Running != 1 {
		t.Fatal("report leaked job map")
	}
}
