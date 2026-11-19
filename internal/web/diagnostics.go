package web

import (
	"log/slog"
	"net/http"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"mochil/internal/safely"
)

const (
	slowRequestThreshold = 700 * time.Millisecond
	slowRequestsKept     = 25
	recentFailuresKept   = 25
)

// diagnosticsRecord is one remembered request. Query strings are dropped so
// tokens and search terms never reach the page.
type diagnosticsRecord struct {
	At         int64  `json:"at"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	DurationMS int64  `json:"duration_ms"`
}

type requestDiagnostics struct {
	mu       sync.Mutex
	started  time.Time
	requests int64
	failures int64
	slow     []diagnosticsRecord
	errors   []diagnosticsRecord
}

var diagnostics = &requestDiagnostics{started: time.Now()}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Observe records request counts, the slowest requests and recent server
// failures for the diagnostics panel. Streaming media is excluded from the
// slow list, since a long transfer is not a slow response.
func Observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		diagnostics.record(r, writer.status, time.Since(started))
	})
}

func streamingPath(path string) bool {
	for _, prefix := range []string{"/videos/", "/audio/", "/podcast-media/", "/api/video-hls/", "/api/events"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (d *requestDiagnostics) record(r *http.Request, status int, took time.Duration) {
	if status == 0 {
		status = http.StatusOK
	}
	entry := diagnosticsRecord{At: time.Now().UnixMilli(), Method: r.Method, Path: r.URL.Path, Status: status, DurationMS: took.Milliseconds()}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.requests++
	if status >= 500 && status != http.StatusNotImplemented { // 501 means "feature not configured", not a fault
		d.failures++
		d.errors = append([]diagnosticsRecord{entry}, d.errors...)
		if len(d.errors) > recentFailuresKept {
			d.errors = d.errors[:recentFailuresKept]
		}
	}
	if took >= slowRequestThreshold && !streamingPath(r.URL.Path) {
		d.slow = append(d.slow, entry)
		sort.Slice(d.slow, func(i, j int) bool { return d.slow[i].DurationMS > d.slow[j].DurationMS })
		if len(d.slow) > slowRequestsKept {
			d.slow = d.slow[:slowRequestsKept]
		}
	}
}

type diagnosticsView struct {
	UptimeSeconds int64               `json:"uptime_seconds"`
	Requests      int64               `json:"requests"`
	Failures      int64               `json:"failures"`
	HeapMB        float64             `json:"heap_mb"`
	SystemMB      float64             `json:"system_mb"`
	Goroutines    int                 `json:"goroutines"`
	GC            uint32              `json:"gc_cycles"`
	Libraries     int                 `json:"libraries"`
	Slow          []diagnosticsRecord `json:"slow"`
	Errors        []diagnosticsRecord `json:"errors"`
	Background    []safely.Failure    `json:"background"`
}

func (d *requestDiagnostics) snapshot(libraries int) diagnosticsView {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	d.mu.Lock()
	defer d.mu.Unlock()
	return diagnosticsView{
		UptimeSeconds: int64(time.Since(d.started).Seconds()), Requests: d.requests, Failures: d.failures,
		HeapMB: float64(memory.HeapAlloc) / (1 << 20), SystemMB: float64(memory.Sys) / (1 << 20),
		Goroutines: runtime.NumGoroutine(), GC: memory.NumGC, Libraries: libraries,
		Slow: append([]diagnosticsRecord{}, d.slow...), Errors: append([]diagnosticsRecord{}, d.errors...),
		Background: safely.Failures(),
	}
}

func (m *multiServer) apiDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, diagnostics.snapshot(len(m.libraries)))
}

// Recover keeps a panic in one handler from ending the connection without a
// word. The request fails with a 500, the stack goes to the log, and the
// diagnostics panel counts it like any other server error.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if recovered == http.ErrAbortHandler {
				panic(recovered)
			}
			slog.Error("request panicked", "path", r.URL.Path, "panic", recovered, "stack", string(debug.Stack()))
			// Headers may already be out; this is then a harmless no-op.
			http.Error(w, "Something went wrong on the server", http.StatusInternalServerError)
		}()
		next.ServeHTTP(w, r)
	})
}
