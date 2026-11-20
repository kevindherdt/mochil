package vaultwriter

import (
	"fmt"
	"os"
	"path/filepath"
)

// The destination hard link must be durable before unlinking the original.
// On a sync failure retain both names, rather than risk losing the only durable
// directory entry. Callers must report the failure, not automatically retry.
func completeLinkedMove(source, target string, syncDir func(string) error) error {
	if err := syncDir(filepath.Dir(target)); err != nil {
		return fmt.Errorf("persist destination vault entry (original retained): %w", err)
	}
	if err := os.Remove(source); err != nil {
		return fmt.Errorf("remove original vault entry (destination retained): %w", err)
	}
	return syncDir(filepath.Dir(source))
}
