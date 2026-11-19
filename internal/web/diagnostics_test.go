package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiagnosticsKeepSlowRequestsAndFailuresWithoutQueries(t *testing.T) {
	d := &requestDiagnostics{started: time.Now()}
	fast := httptest.NewRequest(http.MethodGet, "/api/books?token=secret", nil)
	d.record(fast, 200, 20*time.Millisecond)
	d.record(httptest.NewRequest(http.MethodGet, "/api/books?token=secret", nil), 200, 2*time.Second)
	d.record(httptest.NewRequest(http.MethodGet, "/audio/abc", nil), 200, 9*time.Second)
	d.record(httptest.NewRequest(http.MethodPost, "/api/scan", nil), 500, 5*time.Millisecond)
	view := d.snapshot(2)
	if view.Requests != 4 || view.Failures != 1 || view.Libraries != 2 {
		t.Fatalf("counts: %+v", view)
	}
	if len(view.Slow) != 1 || view.Slow[0].Path != "/api/books" || view.Slow[0].DurationMS != 2000 {
		t.Fatalf("slow list must hold the slow API call only, without its query: %+v", view.Slow)
	}
	if len(view.Errors) != 1 || view.Errors[0].Status != 500 {
		t.Fatalf("errors: %+v", view.Errors)
	}
}

func TestObserveCapturesTheStatusTheHandlerWrote(t *testing.T) {
	handler := Observe(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", http.StatusBadGateway) }))
	before := diagnostics.snapshot(0).Failures
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	if diagnostics.snapshot(0).Failures != before+1 {
		t.Fatal("a 502 must count as a failure")
	}
}

func TestRecoverTurnsAPanicIntoAServerError(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("module bug") }))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/x", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("got %d", recorder.Code)
	}
}

func TestDiagnosticsListBackgroundWorkThatPanicked(t *testing.T) {
	contained("diagnostics test work", func() { panic("module bug") })
	for _, failure := range diagnostics.snapshot(0).Background {
		if failure.What == "diagnostics test work" && failure.Message == "module bug" && failure.Count >= 1 {
			return
		}
	}
	t.Fatal("a contained panic must show up in the diagnostics")
}
