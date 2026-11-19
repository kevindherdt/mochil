package web

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// androidDiagnosticStore keeps a small, bounded journal across server restarts.
// Reports have already passed the strict validation in androidDiagnostics.
type androidDiagnosticStore struct {
	path     string
	maxBytes int64
	mu       sync.Mutex
}

func newAndroidDiagnosticStore(path string, maxBytes int64) *androidDiagnosticStore {
	if path == "" || maxBytes < 1 {
		return nil
	}
	return &androidDiagnosticStore{path: path, maxBytes: maxBytes}
}

func (s *androidDiagnosticStore) Append(report androidDiagnosticReport) error {
	if s == nil {
		return nil
	}
	line, err := json.Marshal(report)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	if info, statErr := os.Stat(s.path); statErr == nil && info.Size()+int64(len(line)) > s.maxBytes {
		if err := os.Rename(s.path, s.path+".previous"); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	if _, err = file.Write(line); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
