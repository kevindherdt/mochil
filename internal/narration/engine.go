// Package narration speaks text with a local Pocket TTS model. The model lives
// in a separate Python process that starts on first use and is unloaded when
// idle, so a book server that never narrates pays nothing and a crash in the
// model cannot take the server down with it.
package narration

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"mochil/internal/safely"
)

//go:embed worker.py
var workerScript string

const (
	// MaxTextBytes bounds one request: a sentence or two, never a chapter.
	MaxTextBytes = 1500
	maxReplyLine = 24 << 20
)

var (
	ErrUnavailable = errors.New("narration is not set up")
	ErrTooLong     = errors.New("text is too long to narrate in one request")
	ErrFailed      = errors.New("narration failed")

	// ErrCloningUnavailable means the model weights that can clone a voice are not installed.
	ErrCloningUnavailable = errors.New("voice cloning needs the Pocket TTS weights with voice cloning")
)

// Config says where the runtime is and how hard it may work.
type Config struct {
	Python   string `json:"python,omitempty"`       // interpreter of an environment with pocket-tts installed
	Cache    string `json:"cache,omitempty"`        // Hugging Face cache holding the model weights
	Threads  int    `json:"threads,omitempty"`      // CPU threads for synthesis; default 4
	Token    string `json:"token,omitempty"`        // optional Hugging Face token for gated model weights
	Offline  bool   `json:"offline,omitempty"`      // never contact Hugging Face
	IdleSecs int    `json:"idle_seconds,omitempty"` // unload the model after this long unused; default 600

	// command replaces the Python launch, and ffmpeg the decoder, in tests.
	command []string
	ffmpeg  string
	// transcribeCommand replaces the speech recognition worker in tests.
	transcribeCommand []string
}

// Audio is one spoken piece as 16-bit little-endian mono PCM.
type Audio struct {
	SampleRate int
	PCM        []byte
}

// Seconds is how long the audio plays.
func (a Audio) Seconds() float64 {
	if a.SampleRate == 0 {
		return 0
	}
	return float64(len(a.PCM)/2) / float64(a.SampleRate)
}

type reply struct {
	ID         int64  `json:"id"`
	Ready      bool   `json:"ready"`
	OK         bool   `json:"ok"`
	Code       string `json:"code"`
	More       bool   `json:"more"` // a piece of a streamed answer; the final reply has ok set
	SampleRate int    `json:"sample_rate"`
	PCM        string `json:"pcm"`
}

type process struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	replies chan reply
	done    chan struct{}
	quit    chan struct{} // closed when the process is dropped, so the reader never waits for replies nobody wants
	killer  sync.Once
	stderr  *tailBuffer
}

// Engine owns at most one worker process and serves one request at a time.
type Engine struct {
	cfg Config

	mu     sync.Mutex // serializes requests and guards proc
	proc   *process
	nextID int64
	idle   *time.Timer
	closed bool
}

func New(cfg Config) *Engine {
	if cfg.Threads <= 0 {
		cfg.Threads = 4
	}
	if cfg.IdleSecs <= 0 {
		cfg.IdleSecs = 600
	}
	if cfg.IdleSecs > 86400 {
		// A whole-day-or-longer idle value would overflow the Duration conversion and
		// fire immediately (killing the worker after every request).
		cfg.IdleSecs = 86400
	}
	return &Engine{cfg: cfg}
}

// Available reports whether a runtime is configured and present.
func (e *Engine) Available() bool {
	if len(e.cfg.command) > 0 {
		return true
	}
	if e.cfg.Python == "" {
		return false
	}
	info, err := os.Stat(e.cfg.Python)
	return err == nil && !info.IsDir()
}

// Loaded reports whether the model is in memory right now.
func (e *Engine) Loaded() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.proc != nil
}

// Speak turns text into audio with the named voice: a preset name, or the path
// of a saved voice's state file, in the language of the model that made it.
// It waits its turn behind other requests.
func (e *Engine) Speak(ctx context.Context, text, voice, language string) (Audio, error) {
	if len(text) > MaxTextBytes {
		return Audio{}, ErrTooLong
	}
	r, err := e.call(ctx, map[string]any{"text": text, "voice": voice, "language": language})
	if err != nil {
		return Audio{}, err
	}
	pcm, err := base64.StdEncoding.DecodeString(r.PCM)
	if err != nil || len(pcm)%2 != 0 {
		return Audio{}, fmt.Errorf("%w: unreadable audio", ErrFailed)
	}
	return Audio{SampleRate: r.SampleRate, PCM: pcm}, nil
}

// SpeakStream is Speak, but each piece of audio is handed to emit as soon as the model has made it, so the caller can
// start playing before the sentence is finished. It holds the worker for the whole sentence, like Speak.
func (e *Engine) SpeakStream(ctx context.Context, text, voice, language string, emit func(Audio) error) error {
	if len(text) > MaxTextBytes {
		return ErrTooLong
	}
	_, err := e.callStream(ctx, map[string]any{"text": text, "voice": voice, "language": language, "stream": true}, func(r reply) error {
		pcm, err := base64.StdEncoding.DecodeString(r.PCM)
		if err != nil || len(pcm)%2 != 0 {
			return fmt.Errorf("%w: unreadable audio", ErrFailed)
		}
		return emit(Audio{SampleRate: r.SampleRate, PCM: pcm})
	})
	return err
}

// Warm starts the worker and loads the model and voice, so the first sentence
// is not spoken from a cold start. It says nothing.
func (e *Engine) Warm(ctx context.Context, voice, language string) error {
	_, err := e.call(ctx, map[string]any{"op": "warm", "voice": voice, "language": language})
	return err
}

// Encode turns a prepared recording into a saved voice state file.
func (e *Engine) Encode(ctx context.Context, reference, state, language string) error {
	_, err := e.call(ctx, map[string]any{"op": "encode", "reference": reference, "state": state, "language": language})
	return err
}

// call sends one request to the worker and waits for its answer.
func (e *Engine) call(ctx context.Context, request map[string]any) (reply, error) {
	return e.callStream(ctx, request, nil)
}

// callStream is call for a worker that may answer in pieces: each piece (more set) goes to piece, and the final reply is
// returned.
func (e *Engine) callStream(ctx context.Context, request map[string]any, piece func(reply) error) (reply, error) {
	if !e.Available() {
		return reply{}, ErrUnavailable
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return reply{}, ErrUnavailable
	}
	if err := ctx.Err(); err != nil {
		return reply{}, err
	}
	if e.idle != nil {
		e.idle.Stop()
	}
	defer e.armIdleLocked()

	proc, err := e.startLocked(ctx)
	if err != nil {
		return reply{}, err
	}
	e.nextID++
	id := e.nextID
	request["id"] = id
	line, _ := json.Marshal(request)
	if _, err := proc.stdin.Write(append(line, '\n')); err != nil {
		e.stopLocked()
		return reply{}, fmt.Errorf("%w: %v", ErrFailed, err)
	}
	// A model that is not loaded yet may also have to be downloaded first.
	timeout := time.NewTimer(300 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case r, ok := <-proc.replies:
			if !ok {
				e.stopLocked()
				return reply{}, fmt.Errorf("%w: the worker stopped", ErrFailed)
			}
			if r.ID != id {
				continue // a late answer to a request that gave up
			}
			if r.More {
				if piece != nil {
					if err := piece(r); err != nil {
						e.stopLocked() // the caller left, or the audio was unreadable: drop the sentence
						return reply{}, err
					}
				}
				continue
			}
			if !r.OK {
				if r.Code == "cloning_unavailable" {
					return reply{}, ErrCloningUnavailable
				}
				return reply{}, fmt.Errorf("%w: %s", ErrFailed, r.Code)
			}
			return r, nil
		case <-ctx.Done():
			// The model cannot be interrupted mid-sentence; drop it so the next
			// request starts clean rather than queueing behind abandoned work.
			e.stopLocked()
			return reply{}, ctx.Err()
		case <-timeout.C:
			e.stopLocked()
			return reply{}, fmt.Errorf("%w: timed out", ErrFailed)
		}
	}
}

func (e *Engine) startLocked(ctx context.Context) (*process, error) {
	if e.proc != nil {
		select {
		case <-e.proc.done:
			e.proc = nil
		default:
			return e.proc, nil
		}
	}
	args := e.cfg.command
	if len(args) == 0 {
		config, _ := json.Marshal(map[string]any{
			"cache": e.cfg.Cache, "threads": e.cfg.Threads,
			"offline": e.cfg.Offline, "token": e.cfg.Token, "quantize": true, "temperature": 0.3,
		})
		args = []string{e.cfg.Python, "-u", "-c", workerScript, string(config)}
	}
	cmd := exec.Command(args[0], args[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailed, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailed, err)
	}
	stderr := &tailBuffer{limit: 2048}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailed, err)
	}
	p := &process{cmd: cmd, stdin: stdin, replies: make(chan reply, 4), done: make(chan struct{}), quit: make(chan struct{}), stderr: stderr}
	go func() {
		defer safely.Recover("narration reader")
		defer close(p.done)
		defer close(p.replies)
		reader := bufio.NewReaderSize(stdout, 1<<20)
		for {
			line, err := readLine(reader)
			if err != nil {
				_ = cmd.Wait()
				return
			}
			var r reply
			if json.Unmarshal(line, &r) != nil {
				continue
			}
			select {
			case p.replies <- r:
			case <-p.quit:
				_ = cmd.Wait() // dropped mid-answer: reap the process instead of leaving it a zombie
				return
			}
		}
	}()
	// The first line says the model has loaded. Loading takes a few seconds.
	loading := time.NewTimer(180 * time.Second)
	defer loading.Stop()
	for {
		select {
		case r, ok := <-p.replies:
			if !ok {
				return nil, fmt.Errorf("%w: the worker could not start (%s)", ErrFailed, stderr.String())
			}
			if r.Ready {
				e.proc = p
				return p, nil
			}
		case <-ctx.Done():
			kill(p)
			return nil, ctx.Err()
		case <-loading.C:
			kill(p)
			return nil, fmt.Errorf("%w: the model took too long to load", ErrFailed)
		}
	}
}

func (e *Engine) armIdleLocked() {
	if e.proc == nil || e.closed {
		return
	}
	var timer *time.Timer
	timer = time.AfterFunc(time.Duration(e.cfg.IdleSecs)*time.Second, func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		// A timer that already fired can be blocked on e.mu while a request warms a
		// new worker and re-arms the idle timer; only the current timer may stop it.
		if e.idle != timer {
			return
		}
		e.stopLocked()
	})
	e.idle = timer
}

func (e *Engine) stopLocked() {
	if e.proc != nil {
		kill(e.proc)
		e.proc = nil
	}
}

// Close stops the worker. The engine cannot be used afterwards.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	if e.idle != nil {
		e.idle.Stop()
	}
	e.stopLocked()
}

func kill(p *process) {
	p.killer.Do(func() { close(p.quit) })
	_ = p.stdin.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

func readLine(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		chunk, isPrefix, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		line = append(line, chunk...)
		if len(line) > maxReplyLine {
			return nil, errors.New("reply too large")
		}
		if !isPrefix {
			return line, nil
		}
	}
}

// tailBuffer keeps the last few bytes a worker wrote to stderr, for error messages.
type tailBuffer struct {
	mu    sync.Mutex
	limit int
	data  []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data = append(t.data, p...)
	if len(t.data) > t.limit {
		t.data = t.data[len(t.data)-t.limit:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(bytes.TrimSpace(t.data))
}
