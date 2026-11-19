package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The shell cache must be named after the build, or an upgrade keeps serving the index.html (and the
// script it points at) from the previous release.
func TestServiceWorkerNamesItsShellAfterTheBuild(t *testing.T) {
	server := &multiServer{}
	recorder := httptest.NewRecorder()
	server.serveStatic(recorder, httptest.NewRequest(http.MethodGet, "/sw.js", nil), "static/sw.js")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d", recorder.Code)
	}
	body := recorder.Body.String()
	if strings.Contains(body, "__MOCHIL_SHELL_NAME__") {
		t.Fatal("the marker was served unstamped")
	}
	line := ""
	for _, candidate := range strings.Split(body, "\n") {
		if strings.Contains(candidate, "const SHELL =") {
			line = candidate
			break
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(line), `const SHELL = "mochil-shell-`) || strings.Contains(line, "mochil-shell-v1") {
		t.Fatalf("the shell cache is not named after this build: %q", line)
	}
}
