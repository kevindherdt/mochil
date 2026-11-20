package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"mochil/internal/jobs"
	settingssvc "mochil/internal/settings"
	"strings"
)

func (m *multiServer) settings(w http.ResponseWriter, r *http.Request) {
	if !m.settingsService.Available() {
		http.Error(w, "Managed settings require a config file", http.StatusNotImplemented)
		return
	}
	if r.Method == http.MethodGet {
		data, err := m.settingsService.Read()
		if err != nil {
			// Read() opens the config file, so the error carries a server path and
			// permission detail. Log it; do not hand it to the browser.
			if m.root != nil && m.root.log != nil {
				m.root.log.Error("read managed settings", "error", err)
			}
			http.Error(w, "Settings could not be read", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// This document names library paths, the sync user and the public address. Every other JSON
		// response here is no-store; this one was written by hand and was the only one cacheable.
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, settingssvc.MaximumBytes+1))
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	result, err := m.settingsService.Update(data)
	if err != nil {
		if errors.Is(err, settingssvc.ErrUnavailable) {
			http.Error(w, err.Error(), http.StatusNotImplemented)
			return
		}
		// The read path deliberately keeps server paths out of the browser; the write path used to hand
		// them over, because a failed save bubbles up appconfig's decode error, which quotes the file it
		// was reading. Say what went wrong without saying where the server keeps its files.
		if m.root != nil && m.root.log != nil {
			m.root.log.Error("save managed settings", "error", err)
		}
		http.Error(w, settingsSaveFailure(err), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (m *multiServer) restart(w http.ResponseWriter) {
	if m.cfg.Restart == nil || m.backgroundJobs == nil {
		http.Error(w, "Restart is unavailable", http.StatusNotImplemented)
		return
	}
	err := m.backgroundJobs.Submit("server:restart", func(ctx context.Context) error {
		timer := time.NewTimer(350 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			m.cfg.Restart()
			return nil
		}
	})
	if err != nil && !errors.Is(err, jobs.ErrDuplicate) {
		http.Error(w, "Restart queue is unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"restarting": true})
}

// settingsSaveFailure turns a save error into something safe to show. Decode errors quote the file the
// server was reading and permission errors quote its directory, and neither is the browser's business;
// the reason a person can act on is whether it was their JSON or the server's own file.
func settingsSaveFailure(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid settings JSON"), strings.Contains(message, "settings exceed"):
		return message
	case strings.Contains(message, "decode config"), strings.Contains(message, "unknown field"):
		return "Mochil could not read the settings it was asked to save. Nothing was changed."
	case strings.Contains(message, "permission"), strings.Contains(message, "no such file"):
		return "Mochil could not write its settings file. Nothing was changed."
	default:
		return "Mochil could not save these settings. Nothing was changed."
	}
}
