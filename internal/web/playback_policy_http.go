package web

import (
	"encoding/json"
	"net/http"
)

type playbackPolicyView struct {
	WiFiQuality   string `json:"wifi_quality"`
	MobileQuality string `json:"mobile_quality"`
}

func (m *multiServer) playbackPolicyHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	policy := playbackPolicyView{WiFiQuality: "maximum", MobileQuality: "high"}
	if m.cfg.ConfigJSON != nil {
		if data, err := m.cfg.ConfigJSON(); err == nil {
			var source struct {
				Playback playbackPolicyView `json:"playback"`
			}
			if json.Unmarshal(data, &source) == nil {
				policy.WiFiQuality = normalizedPlaybackQuality(source.Playback.WiFiQuality, policy.WiFiQuality)
				policy.MobileQuality = normalizedPlaybackQuality(source.Playback.MobileQuality, policy.MobileQuality)
			}
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, policy)
}

func normalizedPlaybackQuality(value, fallback string) string {
	switch value {
	case "maximum", "high", "medium", "low":
		return value
	default:
		return fallback
	}
}
