package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"mochil/internal/jobs"
	settingssvc "mochil/internal/settings"
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
		http.Error(w, err.Error(), http.StatusBadRequest)
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
