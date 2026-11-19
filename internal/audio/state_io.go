package audio

import (
	"os"
	"path/filepath"

	"mochil/internal/statefile"
)

const maximumAudioStateBytes = 4 << 20

func decodeAudioState(path string, target any) error {
	return statefile.DecodeJSON(path, maximumAudioStateBytes, target)
}

// saveAudioState shares the decoder's byte bound and replaces only a complete,
// synced temporary file. Stores retain ownership of locking and rollback.
func saveAudioState(path, temporaryPattern string, maximumBytes int64, value any) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), temporaryPattern)
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = statefile.EncodeJSON(temporary, maximumBytes, value, false); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}
