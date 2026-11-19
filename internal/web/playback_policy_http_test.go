package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlaybackPolicyExposesOnlyBoundedQualityPreferences(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ConfigJSON: func() ([]byte, error) {
		return []byte(`{"playback":{"wifi_quality":"high","mobile_quality":"low"},"password":"secret"}`), nil
	}}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/playback-policy", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"wifi_quality":"high"`) || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("policy = %d %s", response.Code, response.Body.String())
	}
}
