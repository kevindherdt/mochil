package health

import (
	"sort"
	"time"
)

type Shelf struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Items int    `json:"items"`
	Ready bool   `json:"ready"`
}

type Snapshot struct {
	Shelves  []Shelf
	Services map[string]bool
	Jobs     map[string]JobQueue
}

type JobQueue struct {
	Queued   int `json:"queued"`
	Running  int `json:"running"`
	Ready    int `json:"ready"`
	Failed   int `json:"failed"`
	Canceled int `json:"canceled"`
}

type Report struct {
	Status        string              `json:"status"`
	GeneratedAt   time.Time           `json:"generated_at"`
	UptimeSeconds int64               `json:"uptime_seconds"`
	Shelves       []Shelf             `json:"shelves"`
	Services      map[string]bool     `json:"services"`
	Jobs          map[string]JobQueue `json:"jobs"`
}

type Service struct {
	started  time.Time
	snapshot func() Snapshot
	now      func() time.Time
}

func New(snapshot func() Snapshot) *Service {
	return &Service{started: time.Now().UTC(), snapshot: snapshot, now: time.Now}
}

func (s *Service) Report() Report {
	now := s.now().UTC()
	snap := Snapshot{Services: map[string]bool{}}
	if s.snapshot != nil {
		snap = s.snapshot()
		if snap.Services == nil {
			snap.Services = map[string]bool{}
		}
	}
	shelves := append([]Shelf(nil), snap.Shelves...)
	sort.Slice(shelves, func(i, j int) bool { return shelves[i].ID < shelves[j].ID })
	status := "ok"
	for _, shelf := range shelves {
		if !shelf.Ready {
			status = "degraded"
			break
		}
	}
	return Report{Status: status, GeneratedAt: now, UptimeSeconds: max(0, int64(now.Sub(s.started).Seconds())), Shelves: shelves, Services: clone(snap.Services), Jobs: cloneJobs(snap.Jobs)}
}

func cloneJobs(input map[string]JobQueue) map[string]JobQueue {
	out := make(map[string]JobQueue, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func clone(input map[string]bool) map[string]bool {
	out := make(map[string]bool, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
