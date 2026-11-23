package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"mochil/internal/appconfig"
	"mochil/internal/document"
	"mochil/internal/notepolish"
	"mochil/internal/vaultwriter"
)

// Polishing a note is a job on the server, not one long request: a small local model can take minutes, and a phone
// connection or proxy in between drops a request that stays open that long. POST starts the job and answers at once,
// GET reports how it is going (and carries the proposal when it is ready), DELETE stops it. Closing the panel does not
// stop it; the proposal waits for 30 minutes. Nothing is ever written to the note here: Apply is a normal page save.
//
// Administrator-only like the rest of /api/hermes, and switched off by the Hermes "polish" ability.

const (
	polishTimeout   = 15 * time.Minute
	polishKeepReady = 30 * time.Minute
)

type hermesPolishRequest struct {
	Library string `json:"library"`
	Page    string `json:"page"`
}

type polishJob struct {
	status   string // running, ready, failed
	started  time.Time
	finished time.Time
	result   map[string]any
	err      string
	cancel   context.CancelFunc
}

func polishKey(library, page string) string { return library + "\x00" + page }

// polishState is what GET and POST send back.
func polishState(job *polishJob) map[string]any {
	state := map[string]any{"status": job.status, "elapsed": int(time.Since(job.started).Seconds())}
	if job.status == "ready" {
		state["result"] = job.result
	}
	if job.status == "failed" {
		state["error"] = job.err
	}
	return state
}

func (m *multiServer) hermesPolish(w http.ResponseWriter, r *http.Request) {
	settings := m.hermesSettings()
	if !settings.AskConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Polish needs a helper model. Open Settings, Hermes, and enter the model address and name under Helper model."})
		return
	}
	if !settings.Feature("polish") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Polishing notes is switched off in Settings, Hermes."})
		return
	}
	var input hermesPolishRequest
	if r.Method == http.MethodGet {
		input = hermesPolishRequest{Library: r.URL.Query().Get("library"), Page: r.URL.Query().Get("page")}
	} else if err := decodeJSONRequest(w, r, 4096, &input, true); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	key := polishKey(input.Library, input.Page)
	m.polishMu.Lock()
	m.dropOldPolishJobs()
	job := m.polishJobs[key]
	m.polishMu.Unlock()

	switch r.Method {
	case http.MethodGet:
		if job == nil {
			writeJSON(w, http.StatusOK, map[string]any{"status": "idle"})
			return
		}
		m.polishMu.Lock()
		state := polishState(job)
		m.polishMu.Unlock()
		writeJSON(w, http.StatusOK, state)
	case http.MethodDelete:
		m.polishMu.Lock()
		if job != nil {
			if job.status == "running" {
				job.cancel()
			}
			delete(m.polishJobs, key)
		}
		m.polishMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "idle"})
	case http.MethodPost:
		m.startPolish(w, settings, input, key, job)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// dropOldPolishJobs forgets proposals nobody collected. The caller holds polishMu.
func (m *multiServer) dropOldPolishJobs() {
	for key, job := range m.polishJobs {
		if job.status != "running" && time.Since(job.finished) > polishKeepReady {
			delete(m.polishJobs, key)
		}
	}
}

func (m *multiServer) startPolish(w http.ResponseWriter, settings appconfig.Hermes, input hermesPolishRequest, key string, existing *polishJob) {
	running := false
	if existing != nil {
		m.polishMu.Lock()
		running = existing.status == "running"
		m.polishMu.Unlock()
	}
	if running {
		m.polishMu.Lock()
		state := polishState(existing)
		m.polishMu.Unlock()
		writeJSON(w, http.StatusOK, state) // the same note: pick the work back up
		return
	}
	library, ok := m.library(input.Library)
	if !ok || library.Kind != "vault" || library.Documents == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "That wiki was not found."})
		return
	}
	content, item, err := library.Documents.ReadText(input.Page, document.DefaultTextLimit)
	if err != nil || !document.IsMarkdown(item.Format) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "That note was not found."})
		return
	}
	original := string(content)
	front, body := notepolish.SplitFrontMatter(original)
	if strings.TrimSpace(body) == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "This note has no text to polish yet."})
		return
	}
	if len(body) > notepolish.MaximumBodyBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "This note is too long to polish in one go. Polish it in parts."})
		return
	}
	if !m.tryHermesTurn() {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Hermes is busy with another request. Wait for it to finish, then try again."})
		return
	}
	// The job outlives this request on purpose.
	ctx, cancel := context.WithTimeout(context.Background(), polishTimeout)
	job := &polishJob{status: "running", started: time.Now(), cancel: cancel}
	m.polishMu.Lock()
	if m.polishJobs == nil {
		m.polishJobs = map[string]*polishJob{}
	}
	m.polishJobs[key] = job
	m.polishMu.Unlock()
	go func() {
		defer m.endHermesTurn()
		defer cancel()
		reply, err := hermesAsk(ctx, settings, notepolish.Instructions, notepolish.Message(item.Title, body))
		m.polishMu.Lock()
		defer m.polishMu.Unlock()
		if m.polishJobs[key] != job { // stopped, and maybe started again, meanwhile
			return
		}
		job.finished = time.Now()
		fail := func(message string) { job.status, job.err = "failed", message }
		switch {
		case errors.Is(err, errHermesUnreachable):
			fail("Hermes did not answer in time. If the gaming PC is off, the small model on the Mac can take several minutes for a long note; try a shorter note, or try again.")
		case err != nil:
			fail("Hermes could not polish this note right now.")
		default:
			polished := notepolish.CleanReply(reply)
			if !notepolish.Plausible(body, polished) {
				fail("Hermes did not return the note, so nothing is proposed. Try again, or use a stronger model.")
				return
			}
			proposal := notepolish.Assemble(front, body, polished)
			job.status = "ready"
			job.result = map[string]any{
				"title": item.Title, "original": original, "proposal": proposal,
				"revision": vaultwriter.Revision(content), "lost": notepolish.Lost(body, polished),
				"same": proposal == original, "writable": library.VaultWriter != nil,
			}
		}
	}()
	m.polishMu.Lock()
	state := polishState(job)
	m.polishMu.Unlock()
	writeJSON(w, http.StatusAccepted, state)
}
