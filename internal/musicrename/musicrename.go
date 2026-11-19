// Package musicrename renames music files in place, safely: every rename is checked first, nothing is ever
// overwritten, a ledger of old and new names is written to disk before the first file moves, a failure part-way puts
// the already-moved files back, and a finished batch can be undone from its ledger.
package musicrename

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mochil/internal/musicnames"
)

// Move is one file: its path relative to the library folder before and after, and the track ids that go with them.
type Move struct {
	ID    string `json:"id,omitempty"` // empty for a sidecar file (lyrics) that travels with a track
	NewID string `json:"new_id,omitempty"`
	From  string `json:"from"`
	To    string `json:"to"`
	Done  bool   `json:"done"`
}

// Ledger is the record of one batch.
type Ledger struct {
	ID      string `json:"id"`
	Library string `json:"library"`
	Created string `json:"created"`
	Moves   []Move `json:"moves"`
	State   string `json:"state"` // "applying", "done", "failed" (rolled back), "undone"
}

// Engine works on one library folder and keeps its ledgers in Dir.
type Engine struct {
	Root string
	Dir  string
	mu   sync.Mutex
}

// sidecars travel with a track: a lyrics file with the same name would otherwise be left behind.
var sidecars = []string{".lrc"}

func (e *Engine) abs(relative string) (string, error) {
	full := filepath.Join(e.Root, filepath.FromSlash(relative))
	if rel, err := filepath.Rel(e.Root, full); err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", errors.New("a path leaves the library folder")
	}
	return full, nil
}

func missing(full string) bool {
	_, err := os.Lstat(full)
	return errors.Is(err, os.ErrNotExist)
}

// plan checks every move and adds the sidecar moves. Nothing is touched.
func (e *Engine) plan(moves []Move) ([]Move, error) {
	if len(moves) == 0 {
		return nil, errors.New("nothing to rename")
	}
	if len(moves) > 2000 {
		return nil, errors.New("too many files in one batch")
	}
	sources, targets := map[string]bool{}, map[string]bool{}
	for _, move := range moves {
		sources[strings.ToLower(move.From)] = true
	}
	var all []Move
	for _, move := range moves {
		if err := musicnames.ValidateRename(move.From, move.To); err != nil {
			return nil, fmt.Errorf("%s: %w", move.From, err)
		}
		from, err := e.abs(move.From)
		if err != nil {
			return nil, err
		}
		if _, err := e.abs(move.To); err != nil {
			return nil, err
		}
		info, err := os.Lstat(from)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a plain file that can be renamed", move.From)
		}
		key := strings.ToLower(move.To)
		if targets[key] || sources[key] {
			return nil, fmt.Errorf("%s: another file in this batch uses that name (swapping names is not supported)", move.To)
		}
		targets[key] = true
		if to, _ := e.abs(move.To); !missing(to) {
			return nil, fmt.Errorf("%s already exists", move.To)
		}
		all = append(all, Move{ID: move.ID, NewID: move.NewID, From: move.From, To: move.To})
		for _, extension := range sidecars {
			side, sideTo := strings.TrimSuffix(move.From, path.Ext(move.From))+extension, strings.TrimSuffix(move.To, path.Ext(move.To))+extension
			sideFrom, _ := e.abs(side)
			sideTarget, _ := e.abs(sideTo)
			if info, err := os.Lstat(sideFrom); err == nil && info.Mode().IsRegular() {
				if !missing(sideTarget) {
					return nil, fmt.Errorf("%s already exists", sideTo)
				}
				all = append(all, Move{From: side, To: sideTo})
			}
		}
	}
	return all, nil
}

func (e *Engine) save(l Ledger) error {
	if err := os.MkdirAll(e.Dir, 0o750); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(l, "", " ")
	temporary, err := os.CreateTemp(e.Dir, ".ledger-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err = temporary.Write(raw); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporary.Name(), filepath.Join(e.Dir, l.ID+".json"))
}

// Apply renames the files. On any failure the files already moved are put back and the error says why.
func (e *Engine) Apply(library string, moves []Move) (Ledger, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	all, err := e.plan(moves)
	if err != nil {
		return Ledger{}, err
	}
	token := make([]byte, 4)
	_, _ = rand.Read(token)
	ledger := Ledger{ID: time.Now().UTC().Format("20060102-150405") + "-" + hex.EncodeToString(token), Library: library, Created: time.Now().UTC().Format(time.RFC3339), Moves: all, State: "applying"}
	if err := e.save(ledger); err != nil { // before the first file moves, so a crash leaves a record to undo from
		return Ledger{}, errors.New("could not write the rename record, so nothing was renamed")
	}
	for i := range ledger.Moves {
		from, _ := e.abs(ledger.Moves[i].From)
		to, _ := e.abs(ledger.Moves[i].To)
		err := errors.New("the new name appeared meanwhile")
		if missing(to) {
			err = os.Rename(from, to)
		}
		if err != nil {
			if e.revert(&ledger) > 0 {
				// Some files could not be put back: keep the record open so Undo can try again.
				_ = e.save(ledger)
				return ledger, fmt.Errorf("could not rename %s (%v); some files already renamed could not be put back, use Undo to retry", ledger.Moves[i].From, err)
			}
			ledger.State = "failed"
			_ = e.save(ledger)
			return ledger, fmt.Errorf("could not rename %s (%v); the files already renamed were put back", ledger.Moves[i].From, err)
		}
		ledger.Moves[i].Done = true
		_ = e.save(ledger)
	}
	ledger.State = "done"
	if err := e.save(ledger); err != nil {
		return ledger, errors.New("the files were renamed but the record could not be finished")
	}
	return ledger, nil
}

// revert moves every done move back, newest first, and reports how many could not be put back.
func (e *Engine) revert(l *Ledger) (stuck int) {
	for i := len(l.Moves) - 1; i >= 0; i-- {
		move := &l.Moves[i]
		if !move.Done {
			continue
		}
		from, _ := e.abs(move.From)
		to, _ := e.abs(move.To)
		if missing(from) && !missing(to) && os.Rename(to, from) == nil {
			move.Done = false
		} else {
			stuck++
		}
	}
	return stuck
}

// Load reads one ledger.
func (e *Engine) Load(id string) (Ledger, error) {
	if id == "" || strings.ContainsAny(id, `/\.`) {
		return Ledger{}, errors.New("unknown rename record")
	}
	raw, err := os.ReadFile(filepath.Join(e.Dir, id+".json"))
	if err != nil {
		return Ledger{}, errors.New("unknown rename record")
	}
	var l Ledger
	if json.Unmarshal(raw, &l) != nil || l.ID != id {
		return Ledger{}, errors.New("unknown rename record")
	}
	return l, nil
}

// Latest is the newest batch of a library that can still be undone (finished, or interrupted part-way).
func (e *Engine) Latest(library string) (Ledger, bool) {
	names, _ := filepath.Glob(filepath.Join(e.Dir, "*.json"))
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		l, err := e.Load(strings.TrimSuffix(filepath.Base(name), ".json"))
		if err == nil && l.Library == library && (l.State == "done" || l.State == "applying") {
			return l, true
		}
	}
	return Ledger{}, false
}

// Undo puts every file of a batch back under its old name. Files that cannot be put back (the old name is taken, or the
// renamed file is gone) are left and counted; the ledger then stays so it can be tried again.
func (e *Engine) Undo(id string) (Ledger, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	l, err := e.Load(id)
	if err != nil {
		return Ledger{}, err
	}
	if l.State == "undone" || l.State == "failed" {
		return l, errors.New("this rename was already undone")
	}
	stuck := e.revert(&l)
	if stuck > 0 {
		_ = e.save(l)
		return l, fmt.Errorf("%d files could not be put back (their old names are taken or the files moved); the rest were restored", stuck)
	}
	l.State = "undone"
	if err := e.save(l); err != nil {
		return l, errors.New("the files were restored but the record could not be finished")
	}
	return l, nil
}
