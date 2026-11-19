package web

import (
	"net/http"
	"sort"
)

const androidHomeSectionLimit = 5

type androidHomeSnapshot struct {
	Reading   []readingView `json:"reading"`
	Listening []audioView   `json:"listening"`
	Watching  []videoView   `json:"watching"`
}

// androidHome returns the three bounded continuation lanes in one request.
// This keeps reverse-proxy and authenticated connection overhead out of the
// critical Home path without coupling the underlying progress stores.
func (m *multiServer) androidHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	account := m.readingAccount(r)
	snapshot := androidHomeSnapshot{
		Reading:   []readingView{},
		Listening: []audioView{},
		Watching:  []videoView{},
	}
	if m.reading != nil {
		snapshot.Reading = m.readingViews(account, false, "", androidHomeSectionLimit)
	}
	if m.audioProgress != nil {
		progress := m.audioProgress.List(account)
		sort.Slice(progress, func(i, j int) bool { return progress[i].UpdatedAt > progress[j].UpdatedAt })
		snapshot.Listening = m.audioProgressViews(account, progress, "", 0, true, androidHomeSectionLimit)
	}
	if m.videoProgress != nil {
		progress := m.videoProgress.List(account)
		sort.Slice(progress, func(i, j int) bool { return progress[i].UpdatedAt > progress[j].UpdatedAt })
		snapshot.Watching = m.videoProgressViews(account, progress, true, androidHomeSectionLimit)
	}
	writeJSON(w, http.StatusOK, snapshot)
}
