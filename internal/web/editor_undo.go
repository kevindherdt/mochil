package web

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// editorUndoRemembers is how many batches stay available to take back.
const editorUndoRemembers = 20

// editorUndo keeps the last few batches so a bulk edit can be taken back. It lives in memory: after a restart the
// batch can no longer be undone, but each book can still be reset to its files.
type editorUndo struct {
	mu    sync.Mutex
	order []string
	undo  map[string]func() error
}

func (u *editorUndo) remember(undo func() error) string {
	raw := make([]byte, 8)
	_, _ = rand.Read(raw)
	token := hex.EncodeToString(raw)
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.undo == nil {
		u.undo = map[string]func() error{}
	}
	u.undo[token] = undo
	u.order = append(u.order, token)
	for len(u.order) > editorUndoRemembers {
		delete(u.undo, u.order[0])
		u.order = u.order[1:]
	}
	return token
}

// peek returns the newest batch without consuming it, so a failed undo can be retried.
func (u *editorUndo) peek(token string) (func() error, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	// An undo restores the state as it was before its batch, so only the newest batch can be taken back safely.
	if len(u.order) == 0 || u.order[len(u.order)-1] != token {
		return nil, false
	}
	return u.undo[token], true
}

// drop consumes a batch once its undo has been applied.
func (u *editorUndo) drop(token string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.order) == 0 || u.order[len(u.order)-1] != token {
		return false
	}
	delete(u.undo, token)
	u.order = u.order[:len(u.order)-1]
	return true
}
