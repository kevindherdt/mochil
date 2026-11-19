package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mochil/internal/jobs"
)

func TestRestartRequestsCoalesceBehindBackgroundQueue(t *testing.T) {
	pool := jobs.New(1, 4)
	defer pool.Close()
	m := &multiServer{cfg: MultiConfig{Restart: func() { t.Error("restart should be canceled by test cleanup") }}, backgroundJobs: pool}
	for range 2 {
		response := httptest.NewRecorder()
		m.restart(response)
		if response.Code != http.StatusAccepted {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
	}
	states := pool.Snapshot()
	if len(states) != 1 || states[0].Key != "server:restart" {
		t.Fatalf("restart jobs: %#v", states)
	}
}
