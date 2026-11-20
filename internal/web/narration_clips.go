package web

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// narrationClips keeps the audio of sentences that have been spoken, and shares the
// work between two requests for the same one. The voice speaks only a little faster
// than it is heard, so the app cannot wait to ask for each sentence as the last one
// ends: the server generates the next few while the current one plays, and a request
// for a sentence already being made waits for that instead of starting another.
// errNarrationBusy is returned when too many sentences are already being made.
var errNarrationBusy = errors.New("narration is busy")

const maxClipsInProgress = 16

type narrationClips struct {
	pending int
	mutex   sync.Mutex
	maximum int
	bytes   int
	order   *list.List // finished clips, newest first
	clips   map[string]*clip
}

type clip struct {
	key     string
	done    chan struct{}
	wav     []byte
	err     error
	element *list.Element
}

func newNarrationClips(maximumBytes int) *narrationClips {
	return &narrationClips{maximum: maximumBytes, order: list.New(), clips: map[string]*clip{}}
}

// has reports whether a sentence's audio is ready or on its way.
func (c *narrationClips) has(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, ok := c.clips[key]
	return ok
}

// get returns the audio for key, making it with speak when nobody has yet. The work
// carries on if the caller gives up, so what was started is ready for the next request.
func (c *narrationClips) get(ctx context.Context, key string, speak func(context.Context) ([]byte, error)) ([]byte, error) {
	c.mutex.Lock()
	entry, ok := c.clips[key]
	if ok {
		if entry.element != nil {
			c.order.MoveToFront(entry.element)
		}
	} else {
		if c.pending >= maxClipsInProgress {
			c.mutex.Unlock()
			return nil, errNarrationBusy
		}
		c.pending++
		entry = &clip{key: key, done: make(chan struct{})}
		c.clips[key] = entry
		go func() {
			work, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			wav, err := speakContained(work, speak)
			c.mutex.Lock()
			defer c.mutex.Unlock()
			c.pending--
			entry.wav, entry.err = wav, err
			close(entry.done)
			if err != nil {
				delete(c.clips, key) // a failure is not remembered; the next request tries again
				return
			}
			entry.element = c.order.PushFront(entry)
			c.bytes += len(wav)
			for c.bytes > c.maximum && c.order.Len() > 1 {
				oldest := c.order.Back()
				old := c.order.Remove(oldest).(*clip)
				delete(c.clips, old.key)
				c.bytes -= len(old.wav)
			}
		}()
	}
	c.mutex.Unlock()
	select {
	case <-entry.done:
		return entry.wav, entry.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// speakContained turns a panic in the voice engine into that clip's failure: left alone it would end the server, or
// leave the clip "in progress" for ever with everyone who asked for it waiting.
func speakContained(ctx context.Context, speak func(context.Context) ([]byte, error)) (wav []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("narration clip panicked", "panic", recovered, "stack", string(debug.Stack()))
			wav, err = nil, fmt.Errorf("narration failed: %v", recovered)
		}
	}()
	return speak(ctx)
}
