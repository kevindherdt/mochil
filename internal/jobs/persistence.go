package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const persistentStateVersion = 1

type persistentState struct {
	Version int     `json:"version"`
	States  []State `json:"states"`
}

func (p *Pool) loadPersistent() error {
	if p.persistPath == "" {
		return nil
	}
	file, err := os.Open(p.persistPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read job state: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect job state: %w", err)
	}
	if info.Size() > 4<<20 {
		return errors.New("job state exceeds the 4 MiB limit")
	}
	data := make([]byte, info.Size())
	if _, err = io.ReadFull(file, data); err != nil {
		return fmt.Errorf("read job state: %w", err)
	}
	var stored persistentState
	if json.Unmarshal(data, &stored) != nil || stored.Version != persistentStateVersion {
		return errors.New("job state is malformed or uses an unsupported version")
	}
	now := time.Now().UTC()
	start := max(0, len(stored.States)-p.history)
	seen := make(map[string]bool, len(stored.States)-start)
	for _, state := range stored.States[start:] {
		if state.Key == "" || len(state.Key) > 512 || state.QueuedAt.IsZero() || state.ProgressCurrent < 0 || state.ProgressTotal < 0 || state.ProgressTotal > 0 && state.ProgressCurrent > state.ProgressTotal || len(state.ProgressMessage) > maximumProgressMessageBytes || len(state.Error) > maximumStateErrorBytes || seen[state.Key] {
			return errors.New("job state contains an invalid entry")
		}
		seen[state.Key] = true
		if state.Status == "queued" || state.Status == "running" {
			state.Status = "failed"
			state.FinishedAt = &now
			state.Error = "job interrupted by process restart"
		}
		if state.Status != "ready" && state.Status != "failed" && state.Status != "canceled" {
			return errors.New("job state contains an invalid status")
		}
		completion := &completion{done: make(chan struct{}), state: state}
		close(completion.done)
		p.states[state.Key] = state
		p.done[state.Key] = completion
		p.order = append(p.order, state.Key)
	}
	return p.persistLocked()
}

func (p *Pool) persistLocked() error {
	if p.persistPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p.persistPath), 0o750); err != nil {
		return fmt.Errorf("create job state directory: %w", err)
	}
	states := make([]State, 0, len(p.order))
	for _, key := range p.order {
		if state, ok := p.states[key]; ok {
			states = append(states, state)
		}
	}
	temporary, err := os.CreateTemp(filepath.Dir(p.persistPath), ".jobs-*")
	if err != nil {
		return fmt.Errorf("create job state: %w", err)
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = temporary.Chmod(0o600); err == nil {
		err = json.NewEncoder(temporary).Encode(persistentState{Version: persistentStateVersion, States: states})
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, p.persistPath)
	}
	if err != nil {
		return fmt.Errorf("save job state: %w", err)
	}
	return nil
}
