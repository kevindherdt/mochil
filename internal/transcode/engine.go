package transcode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mochil/internal/jobs"
	"mochil/internal/safely"
)

type Arguments func(startSegment int, directory string) []string

type Request struct {
	Source    Source
	Variant   string
	Arguments Arguments
}

type Engine struct {
	root, command         string
	maxBytes              int64
	segmentSeconds        int
	pool                  *jobs.Pool
	sequence              atomic.Uint64
	mu                    sync.Mutex
	sessions              map[string]*session
	idleAfter, readyAfter time.Duration

	statsMu      sync.Mutex
	statsReady   bool
	statsTotal   int64
	statsEntries map[string]cacheStat
}

type session struct {
	key, directory  string
	restartMu       sync.Mutex
	mu              sync.Mutex
	jobKey          string
	lastUsed        time.Time
	failed          error
	retryAfter      time.Time
	failures, start int

	notifyMu sync.Mutex
	notifyCh chan struct{}
}

// wake notifies every goroutine blocked in waitChannel that something about
// this session changed (a segment landed, or the job finished/failed), so
// Open() can react immediately instead of waiting for its next poll tick.
func (s *session) wake() {
	s.notifyMu.Lock()
	old := s.notifyCh
	s.notifyCh = nil
	s.notifyMu.Unlock()
	if old != nil {
		close(old)
	}
}

// waitChannel returns a channel that is closed the next time wake() runs.
func (s *session) waitChannel() chan struct{} {
	s.notifyMu.Lock()
	defer s.notifyMu.Unlock()
	if s.notifyCh == nil {
		s.notifyCh = make(chan struct{})
	}
	return s.notifyCh
}

// watchSegments runs alongside an active ffmpeg encode. ffmpeg writes
// segment files directly to disk outside of our process, so we cannot be
// notified of a new segment the way we would for an in-process writer;
// instead this single goroutine polls tightly on behalf of every current
// and future waiter, and wakes them all the moment it observes a change.
// This replaces N concurrent Open() callers each sleeping on their own fixed
// interval with one lightweight watcher per active session.
func watchSegments(ctx context.Context, current *session) {
	// Segments appear complete (ffmpeg's temp_file + rename), so counting names is
	// enough to notice progress. Stat-ing every entry ten times a second made a
	// long recording cost a full core for the whole encode.
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	var lastCount int
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		entries, err := os.ReadDir(current.directory)
		if err != nil {
			continue
		}
		count := 0
		for _, entry := range entries {
			if SegmentName.MatchString(entry.Name()) {
				count++
			}
		}
		if count != lastCount {
			lastCount = count
			current.wake()
		}
	}
}

func NewEngine(root, command string, maxBytes int64, workers, segmentSeconds int) (*Engine, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return nil, err
	}
	if workers < 1 || workers > 8 {
		workers = 2
	}
	if maxBytes <= 0 {
		maxBytes = 4 << 30
	}
	if segmentSeconds < 1 || segmentSeconds > 30 {
		segmentSeconds = 2
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	return &Engine{root: root, command: resolved, maxBytes: maxBytes, segmentSeconds: segmentSeconds, pool: jobs.New(workers, 32), sessions: map[string]*session{}, idleAfter: 75 * time.Second, readyAfter: 20 * time.Second}, nil
}

func (e *Engine) Key(request Request) string { return Key(request.Source, request.Variant) }

func (e *Engine) Start(file *os.File, request Request) (string, error) {
	return e.start(file, request, 0, false)
}

func (e *Engine) Ensure(file *os.File, request Request, segment int) error {
	if segment < 0 || request.Source.Duration > 0 && float64(segment*e.segmentSeconds) >= request.Source.Duration+float64(e.segmentSeconds) {
		closeFile(file)
		return errors.New("invalid HLS segment") // past the end: never start an encoder there
	}
	key := e.Key(request)
	if key == "" {
		closeFile(file)
		return errors.New("invalid HLS source")
	}
	directory := filepath.Join(e.root, key)
	if readyFile(filepath.Join(directory, fmt.Sprintf("seg%05d.m4s", segment))) {
		closeFile(file)
		return nil
	}
	e.mu.Lock()
	active := e.sessions[key]
	e.mu.Unlock()
	if active != nil {
		active.mu.Lock()
		running, start := active.jobKey != "", active.start
		active.mu.Unlock()
		if running && segment >= start && segment <= highestSegment(directory, start)+10 {
			closeFile(file)
			return nil
		}
	}
	_, err := e.start(file, request, segment, true)
	return err
}

func (e *Engine) start(file *os.File, request Request, start int, force bool) (string, error) {
	key := e.Key(request)
	if file == nil || key == "" || request.Arguments == nil {
		closeFile(file)
		return "", errors.New("invalid HLS source")
	}
	directory := filepath.Join(e.root, key)
	e.mu.Lock()
	current := e.sessions[key]
	if current == nil {
		current = &session{key: key, directory: directory, lastUsed: time.Now()}
		e.sessions[key] = current
	}
	e.mu.Unlock()
	current.restartMu.Lock()
	defer current.restartMu.Unlock()
	current.mu.Lock()
	current.lastUsed = time.Now()
	if !force && completeManifest(filepath.Join(directory, "_ffmpeg.m3u8")) {
		current.failed, current.failures, current.retryAfter = nil, 0, time.Time{}
		current.mu.Unlock()
		closeFile(file)
		return key, nil
	}
	if current.failed != nil && time.Now().Before(current.retryAfter) {
		retry := current.retryAfter
		current.mu.Unlock()
		closeFile(file)
		return "", fmt.Errorf("HLS retry unavailable until %s", retry.UTC().Format(time.RFC3339))
	}
	if current.jobKey != "" && !force {
		current.mu.Unlock()
		closeFile(file)
		return key, nil
	}
	if force && current.jobKey != "" {
		previous := current.jobKey
		current.mu.Unlock()
		e.pool.Cancel(previous)
		ctx, cancel := context.WithTimeout(context.Background(), e.readyAfter)
		err := e.pool.Wait(ctx, previous)
		cancel()
		if err != nil && !errors.Is(err, context.Canceled) {
			closeFile(file)
			return "", fmt.Errorf("stop previous HLS conversion: %w", err)
		}
		current.mu.Lock()
		if current.jobKey != "" {
			current.mu.Unlock()
			closeFile(file)
			return "", errors.New("previous HLS conversion is still stopping")
		}
		if errors.Is(current.failed, context.Canceled) {
			current.failed = nil
			current.retryAfter = time.Time{}
		}
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		current.mu.Unlock()
		closeFile(file)
		return "", err
	}
	jobKey := "hls:" + key + ":" + strconv.Itoa(start) + ":" + strconv.FormatUint(e.sequence.Add(1), 10)
	current.jobKey, current.failed, current.start = jobKey, nil, start
	current.mu.Unlock()
	args := request.Arguments(start, directory)
	if err := e.pool.SubmitWithCleanup(jobKey, func(ctx context.Context) error { return e.run(ctx, current, jobKey, file, args) }, func() {
		// Cancelled while still queued, so run() never starts: release the source
		// file and clear the session marker exactly as run() would have.
		_ = file.Close()
		e.finish(current, jobKey, context.Canceled)
	}); err != nil {
		closeFile(file)
		current.mu.Lock()
		if current.jobKey == jobKey {
			current.jobKey = ""
			current.failed = err
			current.retryAfter = time.Now().Add(time.Second)
		}
		current.mu.Unlock()
		return "", err
	}
	go safely.Run("transcode idle stop", func() { e.stopWhenIdle(current, jobKey) })
	return key, nil
}

func (e *Engine) run(ctx context.Context, current *session, jobKey string, file *os.File, args []string) error {
	defer file.Close()
	watchCtx, stopWatch := context.WithCancel(ctx)
	go safely.Run("transcode segment watch", func() { watchSegments(watchCtx, current) })
	defer stopWatch()
	command := exec.CommandContext(ctx, e.command, args...)
	command.ExtraFiles = []*os.File{file}
	// See the matching comment in internal/video/hls.go's run(): without
	// WaitDelay, a grandchild process that outlives a killed direct child
	// would make Run() hang until it exits on its own.
	command.WaitDelay = 5 * time.Second
	stderr := &boundedBuffer{remaining: 128 << 10}
	command.Stdout, command.Stderr = io.Discard, stderr
	err := command.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	} else if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		err = errors.New(message)
	}
	e.finish(current, jobKey, err)
	current.wake()
	e.evict(current.key)
	return err
}

func (e *Engine) finish(current *session, jobKey string, err error) {
	current.mu.Lock()
	defer current.mu.Unlock()
	if current.jobKey != jobKey {
		return
	}
	current.jobKey = ""
	current.failed = err
	if err == nil {
		current.failures = 0
		current.retryAfter = time.Time{}
		return
	}
	current.failures++
	current.retryAfter = time.Now().Add(time.Second * time.Duration(1<<min(current.failures, 5)))
}

func (e *Engine) stopWhenIdle(current *session, jobKey string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		current.mu.Lock()
		active := current.jobKey == jobKey
		idle := time.Since(current.lastUsed)
		current.mu.Unlock()
		if !active {
			return
		}
		if idle > e.idleAfter {
			e.pool.Cancel(jobKey)
			return
		}
	}
}

func (e *Engine) Open(key, name string) (*os.File, string, error) {
	if len(key) != 32 || (name != "init.mp4" && !SegmentName.MatchString(name)) {
		return nil, "", os.ErrNotExist
	}
	e.mu.Lock()
	current := e.sessions[key]
	e.mu.Unlock()
	if current == nil {
		return nil, "", os.ErrNotExist
	}
	path := filepath.Join(current.directory, name)
	deadline := time.Now().Add(e.readyAfter)
	for {
		current.mu.Lock()
		current.lastUsed = time.Now()
		failed := current.failed
		current.mu.Unlock()
		file, err := os.Open(path)
		if err == nil {
			return file, mapResourceType(name), nil
		}
		if failed != nil {
			return nil, "", fmt.Errorf("HLS conversion failed: %w", failed)
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, "", errors.New("HLS segment is not ready")
		}
		// Wait to be signaled by the watcher started alongside the active
		// encode (or by finish() on completion/failure) instead of busy
		// polling on a fixed interval; the timer only guards the overall
		// deadline.
		wait := current.waitChannel()
		timer := time.NewTimer(remaining)
		select {
		case <-wait:
		case <-timer.C:
		}
		timer.Stop()
	}
}

func (e *Engine) Jobs() []jobs.State { return e.pool.Snapshot() }
func (e *Engine) Close()             { e.pool.Close() }

func mapResourceType(name string) string {
	if name == "init.mp4" {
		return "audio/mp4"
	}
	return "audio/iso.segment"
}
func closeFile(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}
func readyFile(name string) bool { info, err := os.Stat(name); return err == nil && info.Size() > 0 }
func completeManifest(name string) bool {
	data, err := os.ReadFile(name)
	return err == nil && strings.Contains(string(data), "#EXT-X-ENDLIST")
}
func highestSegment(directory string, start int) int {
	for segment := start; ; segment++ {
		if !readyFile(filepath.Join(directory, fmt.Sprintf("seg%05d.m4s", segment))) {
			return segment - 1
		}
	}
}

type boundedBuffer struct {
	data      []byte
	remaining int
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	written := len(data)
	if b.remaining > 0 {
		keep := min(len(data), b.remaining)
		b.data = append(b.data, data[:keep]...)
		b.remaining -= keep
	}
	return written, nil
}
func (b *boundedBuffer) String() string { return string(b.data) }
