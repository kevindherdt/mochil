package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// Pairing a TV is the one setup step a fresh install cannot do from the TV itself, so it must work
// without any configuration: the address comes from the request that reached the server.
func TestTVPairingWorksWithoutAPublicURL(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/android/v1/pairing", strings.NewReader(`{"device_name":"Living room TV"}`))
	request.Host = "books.local:8099"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("pairing request = %d %s", response.Code, response.Body.String())
	}
	var pairing struct {
		ID     string `json:"pairing_id"`
		Code   string `json:"user_code"`
		Verify string `json:"verification_uri"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &pairing); err != nil {
		t.Fatal(err)
	}
	if pairing.Code == "" || pairing.ID == "" {
		t.Fatalf("pairing = %+v", pairing)
	}
	if want := "http://books.local:8099/pair?code=" + pairing.Code; pairing.Verify != want {
		t.Fatalf("verification_uri = %q, want %q", pairing.Verify, want)
	}
}

// A configured public_url still wins, because that is the address that works from outside the LAN.
func TestTVPairingPrefersTheConfiguredPublicURL(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, PublicURL: "https://library.example.com/",
		ReadingState: filepath.Join(t.TempDir(), "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/android/v1/pairing", strings.NewReader(`{"device_name":"TV"}`))
	request.Host = "books.local:8099"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("pairing request = %d %s", response.Code, response.Body.String())
	}
	var pairing struct {
		Verify string `json:"verification_uri"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &pairing)
	if !strings.HasPrefix(pairing.Verify, "https://library.example.com/pair?code=") {
		t.Fatalf("verification_uri = %q", pairing.Verify)
	}
}
