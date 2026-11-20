package video

import "time"

// Stop cancels key's active encoder immediately, if one is running, instead
// of waiting for the passive idle timeout (idleStopHLSSession, below) to
// notice nobody is fetching segments anymore. It is how a client's explicit
// "stop playback" signal reaches the ffmpeg process; without it, closing
// the player still left the encoder running (and burning CPU/disk) for up
// to idleAfter.
func (s *HLSStore) Stop(key string) bool {
	s.mu.Lock()
	session := s.sessions[key]
	s.mu.Unlock()
	if session == nil {
		return false
	}
	session.mu.Lock()
	jobKey := session.jobKey
	session.mu.Unlock()
	if jobKey == "" {
		return false
	}
	return s.pool.Cancel(jobKey)
}

// idleStopHLSSession is the passive counterpart to Stop: a safety net for
// when no explicit stop signal arrives (a tab closed without running its
// JS, a network hiccup dropped the request). It cancels jobKey once nobody
// has fetched a segment from session for idleAfter.
func idleStopHLSSession(idleAfter *time.Duration, cancel func(string) bool, session *hlsSession, jobKey string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		session.mu.Lock()
		active := session.jobKey == jobKey
		idle := time.Since(session.lastUsed)
		session.mu.Unlock()
		if !active {
			return
		}
		if idle > *idleAfter {
			cancel(jobKey)
			return
		}
	}
}
