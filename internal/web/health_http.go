package web

import (
	"net/http"

	"mochil/internal/catalog"
	"mochil/internal/health"
	"mochil/internal/jobs"
)

// multiServer.bookCounts (library id -> bookCountAt) remembers how many books each library had at a catalogue version. Counting by asking for the books
// copies the whole list (5,000+ records), so the health report only recounts after the catalogue changed.
type bookCountAt struct {
	version uint64
	count   int
}

func (m *multiServer) bookCount(id string, store catalog.Backend) int {
	versioned, ok := store.(catalog.VersionedBackend)
	if !ok {
		return len(store.Books())
	}
	version := versioned.CatalogVersion()
	if value, found := m.bookCounts.Load(id); found && value.(bookCountAt).version == version {
		return value.(bookCountAt).count
	}
	count := len(store.Books())
	m.bookCounts.Store(id, bookCountAt{version: version, count: count})
	return count
}

func (m *multiServer) healthSnapshot() health.Snapshot {
	shelves := make([]health.Shelf, 0, len(m.libraries))
	for _, library := range m.libraries {
		count, ready := 0, false
		switch {
		case library.Store != nil:
			count, ready = m.bookCount(library.ID, library.Store), true
		case library.Video != nil:
			count, ready = library.Video.Count(), true
		case library.Audio != nil:
			count, ready = library.Audio.Count(), true
		case library.Documents != nil:
			count, ready = library.Documents.Count(), true
		}
		shelves = append(shelves, health.Shelf{ID: library.ID, Kind: library.Kind, Items: count, Ready: ready})
	}
	queues := map[string]health.JobQueue{}
	if m.backgroundJobs != nil {
		queues["background"] = summarizeJobs(m.backgroundJobs.Snapshot())
	}
	if m.podcastDownloads != nil {
		queues["podcast-downloads"] = summarizeJobs(m.podcastDownloads.Jobs())
	}
	if m.hls != nil {
		queues["video-transcoding"] = summarizeJobs(m.hls.Jobs())
	}
	if m.audioHLS != nil {
		queues["audio-transcoding"] = summarizeJobs(m.audioHLS.Jobs())
	}
	if m.audioAnalysis != nil {
		queues["audio-analysis"] = summarizeJobs(m.audioAnalysis.Jobs())
	}
	if m.runtimeJobs != nil {
		queues["library-scans"] = summarizeJobs(m.runtimeJobs.Snapshot())
	}
	return health.Snapshot{Shelves: shelves, Services: map[string]bool{"settings": m.settingsService.Available(), "wiki": m.wiki != nil, "kosync": m.kosync != nil || m.kosyncRemote != nil, "transcoding": m.hls != nil || m.audioHLS != nil, "podcasts": m.podcasts != nil, "background_jobs": m.backgroundJobs != nil}, Jobs: queues}
}

func summarizeJobs(states []jobs.State) health.JobQueue {
	var queue health.JobQueue
	for _, state := range states {
		switch state.Status {
		case "queued":
			queue.Queued++
		case "running":
			queue.Running++
		case "ready":
			queue.Ready++
		case "failed":
			queue.Failed++
		case "canceled":
			queue.Canceled++
		}
	}
	return queue
}

func (m *multiServer) apiHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, m.health.Report())
}
