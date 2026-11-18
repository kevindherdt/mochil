package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"
	"sync"
	"time"
)

var (
	ErrClosed    = errors.New("job pool is closed")
	ErrQueueFull = errors.New("job queue is full")
	ErrDuplicate = errors.New("job is already queued or running")
)

type State struct {
	Key             string     `json:"key"`
	Status          string     `json:"status"`
	QueuedAt        time.Time  `json:"queued_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	Error           string     `json:"error,omitempty"`
	ProgressCurrent int64      `json:"progress_current,omitempty"`
	ProgressTotal   int64      `json:"progress_total,omitempty"`
	ProgressMessage string     `json:"progress_message,omitempty"`
}
type task struct {
	key        string
	run        func(context.Context) error
	completion *completion
	onSkip     func()
}
type completion struct {
	done  chan struct{}
	state State
}
type Pool struct {
	ctx         context.Context
	cancel      context.CancelFunc
	queue       chan task
	mu          sync.RWMutex
	states      map[string]State
	cancels     map[string]context.CancelFunc
	done        map[string]*completion
	pending     map[string]bool
	order       []string
	history     int
	persistPath string
	lastPersist map[string]time.Time
	closed      bool
	wait        sync.WaitGroup
}

func New(workers, queue int) *Pool {
	p, _ := newPool(workers, queue, "")
	return p
}

// NewPersistent creates a pool whose bounded state history is atomically
// stored at path. Running or queued work restored after a process restart is
// marked failed, retaining its last reported progress so callers can retry it.
func NewPersistent(workers, queue int, path string) (*Pool, error) {
	if path == "" {
		return nil, errors.New("job persistence path is required")
	}
	return newPool(workers, queue, path)
}

func newPool(workers, queue int, persistPath string) (*Pool, error) {
	if workers < 1 {
		workers = 1
	}
	if queue < workers {
		queue = workers
	}
	ctx, cancel := context.WithCancel(context.Background())
	history := queue * 4
	if history < 64 {
		history = 64
	}
	p := &Pool{ctx: ctx, cancel: cancel, queue: make(chan task, queue), states: map[string]State{}, cancels: map[string]context.CancelFunc{}, done: map[string]*completion{}, pending: map[string]bool{}, history: history, persistPath: persistPath, lastPersist: map[string]time.Time{}}
	if err := p.loadPersistent(); err != nil {
		cancel()
		return nil, err
	}
	for range workers {
		p.wait.Add(1)
		go p.worker()
	}
	return p, nil
}

// SubmitWithCleanup is Submit with a callback that runs when the task is
// cancelled while still queued and therefore never starts. A caller that hands
// resources to the closure (an open file, a session marker) needs it: without
// it those resources outlive a job that will never run its body.
func (p *Pool) SubmitWithCleanup(key string, run func(context.Context) error, onSkip func()) error {
	return p.submit(key, run, onSkip)
}

func (p *Pool) Submit(key string, run func(context.Context) error) error {
	return p.submit(key, run, nil)
}

func (p *Pool) submit(key string, run func(context.Context) error, onSkip func()) error {
	if key == "" || run == nil {
		return errors.New("invalid job")
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	if p.pending[key] {
		p.mu.Unlock()
		return ErrDuplicate
	}
	state := State{Key: key, Status: "queued", QueuedAt: time.Now().UTC()}
	completion := &completion{done: make(chan struct{}), state: state}
	p.states[key] = state
	p.done[key] = completion
	p.pending[key] = true
	delete(p.lastPersist, key)
	p.removeOrderLocked(key)
	p.order = append(p.order, key)
	p.pruneLocked()
	if err := p.persistLocked(); err != nil {
		delete(p.states, key)
		delete(p.done, key)
		delete(p.pending, key)
		p.removeOrderLocked(key)
		p.persistOrLog("submit rollback")
		p.mu.Unlock()
		return err
	}
	select {
	case p.queue <- task{key: key, run: run, completion: completion, onSkip: onSkip}:
		p.mu.Unlock()
		return nil
	default:
		delete(p.states, key)
		delete(p.done, key)
		delete(p.pending, key)
		p.removeOrderLocked(key)
		p.persistOrLog("queue-full rollback") // the queued record written above must not outlive the rejection
		p.mu.Unlock()
		return ErrQueueFull
	}
}

// Do submits keyed work and waits for its terminal state. Concurrent callers
// with the same key coalesce behind the first submission.
func (p *Pool) Do(ctx context.Context, key string, run func(context.Context) error) error {
	err := p.Submit(key, run)
	if err != nil && !errors.Is(err, ErrDuplicate) {
		return err
	}
	return p.Wait(ctx, key)
}

// Wait waits for an existing job without taking ownership of or canceling it.
func (p *Pool) Wait(ctx context.Context, key string) error {
	p.mu.RLock()
	completion, ok := p.done[key]
	p.mu.RUnlock()
	if !ok {
		return errors.New("job does not exist")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-completion.done:
	}
	state := completion.state
	if state.Status == "canceled" {
		return context.Canceled
	}
	if state.Status == "failed" {
		return errors.New(state.Error)
	}
	return nil
}

func (p *Pool) Cancel(key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	state, ok := p.states[key]
	if !ok || state.Status == "ready" || state.Status == "failed" || state.Status == "canceled" {
		return false
	}
	if cancel := p.cancels[key]; cancel != nil {
		cancel()
	}
	now := time.Now().UTC()
	state.Status = "canceled"
	state.FinishedAt = &now
	p.states[key] = state
	if completion := p.done[key]; completion != nil {
		completion.state = state
	}
	if state.StartedAt == nil {
		p.closeDoneLocked(key)
	}
	p.pruneLocked()
	p.persistOrLog("before-run")
	return true
}
func (p *Pool) State(key string) (State, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	state, ok := p.states[key]
	return state, ok
}
func (p *Pool) Snapshot() []State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]State, 0, len(p.states))
	for _, state := range p.states {
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.cancel()
	now := time.Now().UTC()
	for key, state := range p.states {
		if state.Status != "queued" {
			continue
		}
		state.Status = "canceled"
		state.FinishedAt = &now
		p.states[key] = state
		p.closeDoneLocked(key)
	}
	p.pruneLocked()
	p.persistOrLog("cancel")
	p.mu.Unlock()
	p.wait.Wait()
}
func (p *Pool) worker() {
	defer p.wait.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case job := <-p.queue:
			if p.ctx.Err() != nil {
				return
			}
			p.execute(job)
		}
	}
}
func (p *Pool) execute(job task) {
	p.mu.Lock()
	state, ok := p.states[job.key]
	if !ok || p.done[job.key] != job.completion {
		p.mu.Unlock()
		return
	}
	if state.Status != "queued" {
		delete(p.pending, job.key)
		p.pruneLocked()
		p.persistOrLog("skipped canceled job")
		p.mu.Unlock()
		// The body never runs, so anything it was given has to be released here.
		if job.onSkip != nil {
			job.onSkip()
		}
		return
	}
	ctx, cancel := context.WithCancel(p.ctx)
	ctx = context.WithValue(ctx, progressContextKey{}, progressReporter(func(current, total int64, message string) {
		p.report(job.key, job.completion, current, total, message)
	}))
	p.cancels[job.key] = cancel
	now := time.Now().UTC()
	state.Status = "running"
	state.StartedAt = &now
	p.states[job.key] = state
	p.persistOrLog("running")
	p.mu.Unlock()
	err := runContained(job.run, ctx)
	cancel()
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.cancels, job.key)
	state = p.states[job.key]
	if state.Status == "canceled" {
		delete(p.pending, job.key)
		p.closeDoneLocked(job.key)
		p.pruneLocked()
		p.persistOrLog("canceled job finished")
		return
	}
	done := time.Now().UTC()
	state.FinishedAt = &done
	if err != nil {
		state.Status = "failed"
		state.Error = truncateUTF8(err.Error(), maximumStateErrorBytes)
	} else {
		state.Status = "ready"
	}
	p.states[job.key] = state
	delete(p.pending, job.key)
	p.closeDoneLocked(job.key)
	p.pruneLocked()
	p.persistOrLog("finished")
}

func (p *Pool) report(key string, owner *completion, current, total int64, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	state, ok := p.states[key]
	if !ok || p.done[key] != owner || state.Status != "running" {
		return
	}
	state.ProgressCurrent = current
	state.ProgressTotal = total
	state.ProgressMessage = message
	p.states[key] = state
	now := time.Now().UTC()
	if now.Sub(p.lastPersist[key]) >= 2*time.Second || (total > 0 && current == total) {
		p.persistOrLog("progress")
		p.lastPersist[key] = now
	}
}

func (p *Pool) closeDoneLocked(key string) {
	completion := p.done[key]
	if completion == nil {
		return
	}
	completion.state = p.states[key]
	select {
	case <-completion.done:
	default:
		close(completion.done)
	}
}

func (p *Pool) removeOrderLocked(key string) {
	for index, candidate := range p.order {
		if candidate == key {
			p.order = append(p.order[:index], p.order[index+1:]...)
			return
		}
	}
}

func (p *Pool) pruneLocked() {
	for len(p.states) > p.history {
		removed := false
		for index, key := range p.order {
			state, ok := p.states[key]
			// A canceled job still owns its key until its worker exits (or a
			// canceled queued task is consumed). Status alone is not ownership.
			if !p.pending[key] && (!ok || (state.Status != "queued" && state.Status != "running")) {
				delete(p.states, key)
				delete(p.done, key)
				delete(p.cancels, key)
				delete(p.pending, key)
				delete(p.lastPersist, key)
				p.order = append(p.order[:index], p.order[index+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			return
		}
	}
}

// runContained turns a panic inside a job into that job's failure. One bad file
// or one buggy module must not take the server and every other library with it.
func runContained(run func(context.Context) error, ctx context.Context) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("job panicked", "panic", recovered, "stack", string(debug.Stack()))
			err = fmt.Errorf("job panicked: %v", recovered)
		}
	}()
	return run(ctx)
}

// persistOrLog saves job state where the caller cannot act on a failure; the failure is reported to operators instead of dropped.
func (p *Pool) persistOrLog(transition string) {
	if err := p.persistLocked(); err != nil {
		slog.Error("job state could not be saved", "transition", transition, "error", err)
	}
}
