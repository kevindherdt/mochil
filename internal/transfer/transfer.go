// Package transfer moves or copies publications from one library folder to another. A move on one disk is a link
// plus an unlink, so no data is copied; across disks the file is copied, checked and only then removed from the
// source. Nothing is ever overwritten, and every step can be undone.
package transfer

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Entry is one book: its path below the source folder and the path it should have below the destination folder.
type Entry struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Problem says why one entry cannot be transferred. An empty Reason means it can.
type Problem struct {
	Entry
	Reason string `json:"reason,omitempty"`
}

// Item is one file that was transferred (a book or a companion file beside it), kept so the work can be undone.
type Item struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Size   int64  `json:"size"`
}

// companions are files that describe the book beside them and travel with it, but only when the book is alone in
// its folder (otherwise they could belong to a neighbour).
var companions = []string{"metadata.opf", "cover.jpg", "cover.jpeg", "cover.png"}

var publicationExtensions = map[string]bool{".epub": true, ".pdf": true, ".cbz": true, ".cbr": true, ".mobi": true, ".azw3": true, ".azw": true, ".fb2": true}

type Mover struct{ from, to string }

// New checks that both folders exist and do not contain one another.
func New(from, to string) (*Mover, error) {
	source, err := resolve(from)
	if err != nil {
		return nil, fmt.Errorf("source folder: %w", err)
	}
	target, err := resolve(to)
	if err != nil {
		return nil, fmt.Errorf("destination folder: %w", err)
	}
	if source == target || within(source, target) || within(target, source) {
		return nil, errors.New("the two libraries must be separate folders, neither inside the other")
	}
	return &Mover{from: source, to: target}, nil
}

func resolve(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", errors.New("the folder does not exist")
	}
	// The error used to be dropped, so a folder whose symlink could not be resolved (a FUSE mount answering ENOTSUP, a
	// dangling link swapped in between the stat and this call) became an empty root — and every path joined against ""
	// is inside it, which put the copies in the process's working directory.
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved == "" {
		return "", errors.New("the folder could not be resolved to a real path")
	}
	return resolved, nil
}

func within(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
}

func clean(root, relative string) (string, bool) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", false
	}
	full := filepath.Join(root, filepath.FromSlash(relative))
	return full, within(root, full)
}

// Check reports, for every entry, whether it could be transferred right now. It changes nothing.
func (m *Mover) Check(entries []Entry) []Problem {
	targets := map[string]bool{}
	out := make([]Problem, 0, len(entries))
	for _, entry := range entries {
		problem := Problem{Entry: entry}
		source, okSource := clean(m.from, entry.Source)
		target, okTarget := clean(m.to, entry.Target)
		key := strings.ToLower(filepath.ToSlash(entry.Target))
		switch info, err := os.Lstat(source); {
		case !okSource || !okTarget:
			problem.Reason = "The path is not inside the library."
		case err != nil || !info.Mode().IsRegular():
			problem.Reason = "The book is no longer there."
		case targets[key]:
			problem.Reason = "Two books would end up with the same name."
		default:
			if _, err := os.Lstat(target); err == nil {
				problem.Reason = "The other library already has a file with this name."
			}
		}
		targets[key] = true
		out = append(out, problem)
	}
	return out
}

// Run transfers every entry, stopping at the first failure; what was already done is returned so it can be undone.
func (m *Mover) Run(entries []Entry, move bool) ([]Item, error) {
	var done []Item
	for _, problem := range m.Check(entries) {
		if problem.Reason != "" {
			return done, fmt.Errorf("%s: %s", problem.Source, problem.Reason)
		}
	}
	for _, entry := range entries {
		items, err := m.one(entry, move)
		done = append(done, items...)
		if err != nil {
			return done, fmt.Errorf("%s: %w", entry.Source, err)
		}
	}
	return done, nil
}

func (m *Mover) one(entry Entry, move bool) ([]Item, error) {
	source, _ := clean(m.from, entry.Source)
	files := []Item{{Source: entry.Source, Target: entry.Target}}
	if alone(source) {
		for _, name := range companions {
			if _, err := os.Lstat(filepath.Join(filepath.Dir(source), name)); err != nil {
				continue
			}
			sibling := Item{Source: filepath.ToSlash(filepath.Join(filepath.Dir(entry.Source), name)), Target: filepath.ToSlash(filepath.Join(filepath.Dir(entry.Target), name))}
			if _, err := os.Lstat(filepath.Join(m.to, filepath.FromSlash(sibling.Target))); err == nil {
				continue // never overwrite a neighbour's description
			}
			files = append(files, sibling)
		}
	}
	var done []Item
	for _, file := range files {
		from, _ := clean(m.from, file.Source)
		to, _ := clean(m.to, file.Target)
		size, err := put(from, to, move)
		if err != nil {
			return done, err
		}
		file.Size = size
		done = append(done, file)
	}
	if move {
		m.prune(filepath.Dir(source), m.from)
	}
	return done, nil
}

// alone reports whether a book is the only publication in its folder.
func alone(path string) bool {
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return false
	}
	count := 0
	for _, entry := range entries {
		if publicationExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			count++
		}
	}
	return count == 1
}

// put places one file at its new path without overwriting anything.
func put(from, to string, move bool) (int64, error) {
	info, err := os.Lstat(from)
	if err != nil || !info.Mode().IsRegular() {
		return 0, errors.New("the file is not there")
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return 0, err
	}
	if move {
		// A hard link fails if the target exists, so nothing can be overwritten, and on one disk it copies no data.
		err := os.Link(from, to)
		if err == nil {
			// Flush the new directory entry before dropping the source. The link and the
			// unlink live in different directories, so a crash between them can
			// otherwise lose the book from both libraries.
			if err := syncDirectory(filepath.Dir(to)); err != nil {
				_ = os.Remove(to)
				return 0, err
			}
			if err := os.Remove(from); err != nil {
				_ = os.Remove(to)
				return 0, err
			}
			return info.Size(), nil
		}
		if os.IsExist(err) {
			return 0, errors.New("a file with that name appeared in the other library")
		}
		var link *os.LinkError
		if !errors.As(err, &link) || !errors.Is(link.Err, syscall.EXDEV) {
			// Some filesystems cannot link at all; copying works everywhere.
			if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.ENOTSUP) && !errors.Is(err, syscall.EMLINK) {
				return 0, err
			}
		}
	}
	if err := copyChecked(from, to, info); err != nil {
		return 0, err
	}
	if move {
		if err := syncDirectory(filepath.Dir(to)); err != nil {
			_ = os.Remove(to)
			return 0, err
		}
		if err := os.Remove(from); err != nil {
			_ = os.Remove(to)
			return 0, err
		}
	}
	return info.Size(), nil
}

// Same reports whether two files hold identical bytes. A move across filesystems
// copies first and unlinks the source afterwards, so a restart can find both; only
// identical content makes that state safe to finish rather than refuse.
func Same(left, right string) bool {
	leftInfo, err := os.Stat(left)
	if err != nil || !leftInfo.Mode().IsRegular() {
		return false
	}
	rightInfo, err := os.Stat(right)
	if err != nil || !rightInfo.Mode().IsRegular() || leftInfo.Size() != rightInfo.Size() {
		return false
	}
	leftSum, err := fileHash(left)
	if err != nil {
		return false
	}
	rightSum, err := fileHash(right)
	if err != nil {
		return false
	}
	return bytes.Equal(leftSum, rightSum)
}

func fileHash(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

// syncDirectory flushes a directory entry, so a rename or link survives a crash.
func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

// copyChecked copies to a temporary file beside the target, verifies it byte for byte, and links it into place.
func copyChecked(from, to string, info fs.FileInfo) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	temporary, err := os.CreateTemp(filepath.Dir(to), ".transfer-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(temporary, hash), in)
	if err == nil && written != info.Size() {
		err = errors.New("the copy is not the same size")
	}
	if err == nil {
		err = temporary.Chmod(0o644)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	check, err := os.Open(name)
	if err != nil {
		return err
	}
	defer check.Close()
	again := sha256.New()
	if _, err := io.Copy(again, check); err != nil {
		return err
	}
	if fmt.Sprintf("%x", again.Sum(nil)) != fmt.Sprintf("%x", hash.Sum(nil)) {
		return errors.New("the copy does not match the original")
	}
	if err := os.Link(name, to); err != nil {
		if os.IsExist(err) {
			return errors.New("a file with that name appeared in the other library")
		}
		return err
	}
	return nil
}

// prune removes folders left empty by a move, never the library folder itself.
func (m *Mover) prune(dir, root string) {
	for dir != root && within(root, dir) {
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

// Undo puts back what Run did: a move goes back to where it came from, a copy is deleted (only if it is still the
// file that was made). It continues past a file it cannot restore and reports them together.
func (m *Mover) Undo(done []Item, move bool) error {
	var failed []string
	for i := len(done) - 1; i >= 0; i-- {
		item := done[i]
		from, _ := clean(m.to, item.Target)
		back, _ := clean(m.from, item.Source)
		info, err := os.Lstat(from)
		if err != nil || !info.Mode().IsRegular() || info.Size() != item.Size {
			failed = append(failed, item.Target)
			continue
		}
		if move {
			if _, err := os.Lstat(back); err == nil {
				failed = append(failed, item.Target)
				continue
			}
			if _, err := put(from, back, true); err != nil {
				failed = append(failed, item.Target)
				continue
			}
		} else if err := os.Remove(from); err != nil {
			failed = append(failed, item.Target)
			continue
		}
		m.prune(filepath.Dir(from), m.to)
	}
	if len(failed) > 0 {
		return fmt.Errorf("could not restore: %s", strings.Join(failed, ", "))
	}
	return nil
}
