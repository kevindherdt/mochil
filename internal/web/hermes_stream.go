package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// A reply can take a minute or more (a long prompt, a slower backup model). Browsers and the proxy in front of the
// app drop a connection that stays silent for about 100 seconds, which the page then reports as a broken stream. So
// once Hermes has not answered quickly, the relay opens the stream and says "still waiting" every few seconds.
var (
	hermesQuickAnswer    = 3 * time.Second
	hermesKeepAliveEvery = 10 * time.Second
)

// hermesStreamWriter writes server-sent events to the browser and commits the response only when it must, so a fast
// failure can still be reported as a normal error status.
type hermesStreamWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	started time.Time
	open    bool
	gone    bool
}

func newHermesStreamWriter(w http.ResponseWriter) *hermesStreamWriter {
	flusher, _ := w.(http.Flusher)
	return &hermesStreamWriter{w: w, flusher: flusher, started: time.Now()}
}

func (s *hermesStreamWriter) begin() {
	if s.open {
		return
	}
	s.w.Header().Set("Content-Type", "text/event-stream")
	s.w.Header().Set("Cache-Control", "no-cache")
	s.w.Header().Set("X-Accel-Buffering", "no")
	s.w.WriteHeader(http.StatusOK)
	s.open = true
}

// write passes bytes on. After the browser goes away it does nothing, so the caller keeps reading and Hermes still
// finishes and saves the whole reply.
func (s *hermesStreamWriter) write(data []byte) {
	if s.gone {
		return
	}
	s.begin()
	if _, err := s.w.Write(data); err != nil {
		s.gone = true
		return
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

func (s *hermesStreamWriter) waiting() {
	seconds := int(time.Since(s.started).Seconds())
	s.write([]byte(fmt.Sprintf("event: hermes.waiting\ndata: {\"seconds\":%d}\n\n", seconds)))
}

func (s *hermesStreamWriter) fail(message string) {
	body, _ := json.Marshal(map[string]string{"error": message})
	s.write([]byte("event: hermes.error\ndata: " + string(body) + "\n\n"))
}

type hermesUpstream struct {
	response *http.Response
	err      error
}

// waitForHermes returns the upstream result. If it takes longer than hermesQuickAnswer the stream is opened and a
// waiting event goes out every hermesKeepAliveEvery until Hermes answers.
func waitForHermes(stream *hermesStreamWriter, done <-chan hermesUpstream) hermesUpstream {
	quick := time.NewTimer(hermesQuickAnswer)
	defer quick.Stop()
	tick := time.NewTicker(hermesKeepAliveEvery)
	defer tick.Stop()
	for {
		select {
		case result := <-done:
			return result
		case <-quick.C:
			stream.waiting()
		case <-tick.C:
			if stream.open {
				stream.waiting()
			}
		}
	}
}

// pumpHermes copies the upstream body to the browser, adding a waiting event whenever Hermes is silent for a while.
func pumpHermes(stream *hermesStreamWriter, body io.Reader) {
	chunks := make(chan []byte)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		buffer := make([]byte, 4<<10)
		for {
			n, err := body.Read(buffer)
			if n > 0 {
				chunks <- append([]byte(nil), buffer[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()
	tick := time.NewTicker(hermesKeepAliveEvery)
	defer tick.Stop()
	last := time.Now()
	for {
		select {
		case chunk := <-chunks:
			last = time.Now()
			stream.write(chunk)
		case <-finished:
			return
		case <-tick.C:
			if time.Since(last) >= hermesKeepAliveEvery {
				stream.waiting()
			}
		}
	}
}
