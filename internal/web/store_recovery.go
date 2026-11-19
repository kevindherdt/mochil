package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// openRecovering opens a state store. When the stored JSON is damaged (cut off by a full disk or an unclean
// shutdown), it moves the file aside as <name>.corrupt-<time> and opens the store empty once, so a damaged play
// queue or progress file does not keep the whole server from starting. Nothing is deleted: the damaged file stays
// for recovery.
//
// Only unreadable content is treated that way. A permission error, a full disk or a schema the binary rejects says
// nothing about the file's contents, and moving a good file aside would turn a fixable fault into silent data loss.
// Accounts do not use this at all: starting without them would be worse than refusing to start.
func openRecovering[T any](name, path string, open func() (T, error)) (T, error) {
	store, err := open()
	if err == nil || path == "" {
		return store, err
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		return store, err // missing, unreadable or too busy: report the real fault untouched
	}
	if json.Valid(raw) {
		return store, err // well-formed but rejected (version, values): a person has to look
	}
	aside := fmt.Sprintf("%s.corrupt-%s", path, time.Now().UTC().Format("20060102T150405"))
	if renameErr := os.Rename(path, aside); renameErr != nil {
		return store, err
	}
	slog.Error("state file is damaged; moved aside and started empty", "store", name, "file", path, "moved_to", aside, "error", err)
	return open()
}
