package artwork

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mochil/internal/jobs"
)

// Generator owns bounded FFmpeg image jobs and a disposable LRU cache. Media
// modules supply only extraction arguments and stable content keys.
type Generator struct {
	root, command string
	maxBytes      int64
	pool          *jobs.Pool
}

func NewGenerator(root, command string, maxBytes int64, workers int) (*Generator, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return nil, err
	}
	if workers < 1 || workers > 4 {
		workers = 2
	}
	if maxBytes <= 0 {
		maxBytes = 1 << 30
	}
	if err = os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	return &Generator{root: root, command: resolved, maxBytes: maxBytes, pool: jobs.New(workers, 64)}, nil
}
func (g *Generator) Image(file *os.File, key string, args ...string) (string, error) {
	if file == nil || key == "" {
		return "", errors.New("artwork input is unavailable")
	}
	defer file.Close()
	target := filepath.Join(g.root, key+".jpg")
	if info, err := os.Stat(target); err == nil && info.Size() > 0 {
		_ = os.Chtimes(target, time.Now(), time.Now())
		return target, nil
	}
	err := g.pool.Do(context.Background(), key, func(parent context.Context) error {
		if info, statErr := os.Stat(target); statErr == nil && info.Size() > 0 {
			_ = os.Chtimes(target, time.Now(), time.Now())
			return nil
		}
		temporary := target + ".tmp"
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		arguments := append([]string{"-v", "error", "-nostdin", "-y"}, args...)
		arguments = append(arguments, "-f", "image2", temporary)
		command := exec.CommandContext(ctx, g.command, arguments...)
		command.ExtraFiles = []*os.File{file}
		if output, commandErr := command.CombinedOutput(); commandErr != nil {
			_ = os.Remove(temporary)
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = commandErr.Error()
			}
			return errors.New(message)
		}
		if renameErr := os.Rename(temporary, target); renameErr != nil {
			_ = os.Remove(temporary)
			return renameErr
		}
		g.evict(target)
		return nil
	})
	if err != nil {
		return "", err
	}
	return target, nil
}

func (g *Generator) Close() { g.pool.Close() }
func (g *Generator) evict(keep string) {
	entries, err := os.ReadDir(g.root)
	if err != nil {
		return
	}
	type cached struct {
		path     string
		size     int64
		modified time.Time
	}
	files := []cached{}
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jpg" {
			continue
		}
		info, e := entry.Info()
		if e != nil {
			continue
		}
		files = append(files, cached{filepath.Join(g.root, entry.Name()), info.Size(), info.ModTime()})
		total += info.Size()
	}
	if total <= g.maxBytes {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modified.Before(files[j].modified) })
	for _, file := range files {
		if total <= g.maxBytes {
			break
		}
		if file.path == keep {
			continue
		}
		if os.Remove(file.path) == nil {
			total -= file.size
		}
	}
}
