package web

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"mochil/internal/version"
)

// The About answer exists so "which build are you looking at?" has one number behind it instead of a theory about
// caches. These tests pin the three properties that make it useful: it names the running version, it is never
// cached, and the plain-text form answers before anyone signs in.
func TestVersionAPINamesTheRunningBuild(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&multiServer{}).versionAPI(recorder, httptest.NewRequest("GET", "/api/version", nil))

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store: a cached version number answers the wrong question", got)
	}
	var info aboutInfo
	if err := json.Unmarshal(recorder.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode: %v (%s)", err, recorder.Body.String())
	}
	if info.Product != "Mochil" || info.Version != version.Version {
		t.Fatalf("product/version = %q/%q, want Mochil/%q", info.Product, info.Version, version.Version)
	}
	if info.Started == "" || info.PID == 0 {
		t.Fatalf("started/pid missing: %+v", info)
	}
}

func TestAboutTextIsReadableFromACurl(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&multiServer{}).aboutText(recorder, httptest.NewRequest("GET", "/about", nil))

	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
	first := strings.SplitN(recorder.Body.String(), "\n", 2)[0]
	if !strings.HasPrefix(first, "Mochil "+version.Version) {
		t.Fatalf("first line = %q, want it to name Mochil and the running version", first)
	}
	if !strings.Contains(recorder.Body.String(), "uptime") {
		t.Fatalf("body has no uptime line: %q", recorder.Body.String())
	}
}

// The About answer must never grow into a report: libraries, paths and accounts are exactly the things that cannot
// be handed out unauthenticated, and this endpoint is reachable without a session by design.
func TestVersionAPICarriesNoServerDetails(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&multiServer{}).versionAPI(recorder, httptest.NewRequest("GET", "/api/version", nil))

	body := strings.ToLower(recorder.Body.String())
	for _, forbidden := range []string{"librar", "path", "account", "token", "config"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public version answer mentions %q: %s", forbidden, recorder.Body.String())
		}
	}
}
