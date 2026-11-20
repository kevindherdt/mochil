package web

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	settingssvc "mochil/internal/settings"
)

// The settings document names library paths, the sync user and the public address, so it must not be
// storable by a browser or a proxy — every other JSON route here says no-store, and this one is the
// only one that writes its body by hand.
func TestTheSettingsDocumentIsNotCached(t *testing.T) {
	server := &multiServer{settingsService: liveSettings([]byte(`{"libraries":[]}`))}
	response := httptest.NewRecorder()
	server.settings(response, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

// A failed save used to answer with whatever the decoder said, which quotes the file the server was
// reading. The read path above it already refuses to hand that over.
func TestAFailedSaveDoesNotRevealServerPaths(t *testing.T) {
	service := liveSettings([]byte(`{"libraries":[]}`))
	service.ApplySource = func([]byte) (settingssvc.ApplyResult, error) {
		return settingssvc.ApplyResult{}, errors.New(`/mnt/data/containers/mochil/config/config.json: permission denied`)
	}
	server := &multiServer{settingsService: service, root: &Server{log: slog.New(slog.NewTextHandler(io.Discard, nil))}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"libraries":[]}`))
	server.settings(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if body := response.Body.String(); strings.Contains(body, "config.json") || strings.Contains(body, "/mnt/data") {
		t.Fatalf("the failure told the browser where the config lives: %s", body)
	}
	if !strings.Contains(response.Body.String(), "Nothing was changed") {
		t.Fatalf("the failure did not say the save had no effect: %s", response.Body.String())
	}
}

// liveSettings is a settings service over a byte slice the test owns, so a save can be inspected.
func liveSettings(document []byte) settingssvc.Service {
	return settingssvc.Service{
		ReadSource: func() ([]byte, error) { return document, nil },
		SaveSource: func(data []byte) error { copy(document, data); return nil },
	}
}
