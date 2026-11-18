package web

import (
	"net/http"
	"time"

	"mochil/internal/jobs"
)

type jobView struct {
	Queue           string     `json:"queue"`
	Key             string     `json:"key"`
	Status          string     `json:"status"`
	QueuedAt        time.Time  `json:"queued_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	ProgressCurrent int64      `json:"progress_current,omitempty"`
	ProgressTotal   int64      `json:"progress_total,omitempty"`
	ProgressMessage string     `json:"progress_message,omitempty"`
	Error           string     `json:"error,omitempty"`
}

func (m *multiServer) apiJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := make([]jobView, 0)
	if m.backgroundJobs != nil {
		result = appendJobViews(result, "background", m.backgroundJobs.Snapshot())
	}
	if m.podcastDownloads != nil {
		result = appendJobViews(result, "podcast-downloads", m.podcastDownloads.Jobs())
	}
	if m.hls != nil {
		result = appendJobViews(result, "video-transcoding", m.hls.Jobs())
	}
	if m.audioHLS != nil {
		result = appendJobViews(result, "audio-transcoding", m.audioHLS.Jobs())
	}
	if m.audioAnalysis != nil {
		result = appendJobViews(result, "audio-analysis", m.audioAnalysis.Jobs())
	}
	if m.runtimeJobs != nil {
		result = appendJobViews(result, "library-scans", m.runtimeJobs.Snapshot())
	}
	writeJSON(w, http.StatusOK, result)
}

func appendJobViews(target []jobView, queue string, states []jobs.State) []jobView {
	for _, state := range states {
		target = append(target, jobView{
			Queue: queue, Key: state.Key, Status: state.Status,
			QueuedAt: state.QueuedAt, StartedAt: state.StartedAt,
			FinishedAt:      state.FinishedAt,
			ProgressCurrent: state.ProgressCurrent, ProgressTotal: state.ProgressTotal,
			ProgressMessage: state.ProgressMessage,
			Error:           state.Error,
		})
	}
	return target
}
