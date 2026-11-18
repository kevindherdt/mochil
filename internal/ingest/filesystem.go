package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func (e *Executor) observe(transaction Transaction) []EntryObservation {
	observed := make([]EntryObservation, len(transaction.Entries))
	for index, entry := range transaction.Entries {
		_, temporaryErr := os.Lstat(e.target(entry.TemporaryPath))
		_, targetErr := os.Lstat(e.target(entry.TargetPath))
		observed[index].TemporaryExists = temporaryErr == nil
		observed[index].TargetExists = targetErr == nil
		if observed[index].TargetExists {
			observed[index].TargetMatches, _ = fileMatches(e.target(entry.TargetPath), entry.Size, entry.SHA256)
		}
		if entry.BackupPath != "" {
			_, backupErr := os.Lstat(e.target(entry.BackupPath))
			observed[index].BackupExists = backupErr == nil
		}
	}
	return observed
}

func (e *Executor) copyEntry(ctx context.Context, entry *JournalEntry) error {
	source, info, err := e.resolveSource(entry.SourcePath)
	if err != nil {
		return err
	}
	if info.Size() != entry.Size || info.Size() > e.maxFileBytes {
		return errors.New("ingest source changed after planning")
	}
	temporary := e.target(entry.TemporaryPath)
	if err = e.ensureManagedDirectory(filepath.Dir(temporary)); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, managedFileMode)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		_ = output.Close()
		if remove {
			_ = os.Remove(temporary)
		}
	}()
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(output, hash), io.LimitReader(contextReader{ctx: ctx, reader: input}, e.maxFileBytes+1))
	if err == nil && written != entry.Size {
		err = errors.New("ingest source size changed while copying")
	}
	if err == nil {
		err = output.Sync()
	}
	if closeErr := output.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	entry.SHA256 = hex.EncodeToString(hash.Sum(nil))
	remove = false
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(buffer)
	}
}

// commitEntry reports whether it created the target even when a later fsync or
// temporary cleanup fails, allowing the caller to roll back exactly what this
// attempt owns.
func (e *Executor) commitEntry(entry JournalEntry) (bool, error) {
	target := e.target(entry.TargetPath)
	if err := e.ensureManagedDirectory(filepath.Dir(target)); err != nil {
		return false, err
	}
	temporary := e.target(entry.TemporaryPath)
	// Link is the POSIX no-clobber commit primitive here: unlike Rename it
	// atomically fails when target already exists. Both names are below the same
	// managed root, so they are guaranteed to share a filesystem.
	if err := os.Link(temporary, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, errors.New("ingest target exists; refusing to overwrite it")
		}
		return false, fmt.Errorf("create verified ingest target: %w", err)
	}
	if err := syncDirectory(filepath.Dir(target)); err != nil {
		return true, err
	}
	if err := os.Remove(temporary); err != nil {
		return true, err
	}
	return true, syncDirectory(filepath.Dir(temporary))
}

func (e *Executor) resolveSource(relative string) (string, os.FileInfo, error) {
	path := filepath.Join(e.sourceRoot, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", nil, err
	}
	if !withinRoot(e.sourceRoot, resolved) {
		return "", nil, errors.New("source resolves outside the incoming root")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() {
		return "", nil, errors.New("source is not a regular file")
	}
	return resolved, info, nil
}

func (e *Executor) ensureManagedDirectory(directory string) error {
	if !withinRoot(e.targetRoot, directory) {
		return errors.New("managed directory escapes its root")
	}
	relative, _ := filepath.Rel(e.targetRoot, directory)
	current := e.targetRoot
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err = os.Mkdir(current, managedDirectoryMode); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			info, err = os.Lstat(current)
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("managed path contains a non-directory or symbolic link")
		}
	}
	return nil
}

func (e *Executor) target(relative string) string {
	return filepath.Join(e.targetRoot, filepath.FromSlash(relative))
}

func existingDirectoryRoot(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return filepath.Clean(resolved), nil
}

func withinRoot(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func fileMatches(path string, size int64, expected string) (bool, error) {
	if expected == "" {
		return false, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return false, err
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, io.LimitReader(file, size+1)); err != nil {
		return false, err
	}
	return hex.EncodeToString(hash.Sum(nil)) == expected, nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

// Imported books are media meant to be read by the library server and by other tools (a host user, Kavita), so they are
// created readable by others, like any file in a shared library. The process umask can only narrow these.
const (
	managedFileMode      os.FileMode = 0o644
	managedDirectoryMode os.FileMode = 0o755
)
