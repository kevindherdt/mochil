package acquisition

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"mochil/internal/catalog"
)

type PublicationCache struct {
	directory    string
	maximumBytes int64
	expireAfter  time.Duration
	logger       *slog.Logger
	mu           sync.Mutex
}

func NewPublicationCache(directory string, maximumBytes int64, expireAfter time.Duration, logger *slog.Logger) *PublicationCache {
	cache := &PublicationCache{directory: directory, maximumBytes: maximumBytes, expireAfter: expireAfter, logger: logger}
	_ = os.MkdirAll(directory, 0o750)
	return cache
}

func (c *PublicationCache) Path(libraryID string, book catalog.Book) string {
	sum := sha256.Sum256([]byte(libraryID + "\x00" + book.ID + "\x00" + book.Path + "\x00" + book.Modified.UTC().Format(time.RFC3339Nano)))
	return filepath.Join(c.directory, hex.EncodeToString(sum[:16])+".book")
}

func (c *PublicationCache) Open(libraryID string, book catalog.Book) (*os.File, os.FileInfo, bool) {
	file, err := os.Open(c.Path(libraryID, book))
	if err != nil {
		return nil, nil, false
	}
	info, err := file.Stat()
	if err != nil || (c.expireAfter > 0 && time.Since(info.ModTime()) > c.expireAfter) {
		_ = file.Close()
		_ = os.Remove(c.Path(libraryID, book))
		return nil, nil, false
	}
	now := time.Now()
	_ = os.Chtimes(c.Path(libraryID, book), now, now)
	return file, info, true
}

func (c *PublicationCache) Begin(libraryID string, book catalog.Book) (*os.File, string, bool) {
	if c.maximumBytes <= 0 || (book.Size > 0 && book.Size > c.maximumBytes) {
		return nil, "", false
	}
	if err := os.MkdirAll(c.directory, 0o750); err != nil {
		return nil, "", false
	}
	file, err := os.CreateTemp(c.directory, ".publication-*")
	if err != nil {
		return nil, "", false
	}
	return file, c.Path(libraryID, book), true
}

func (c *PublicationCache) Commit(file *os.File, target string) bool {
	name := file.Name()
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(name)
		return false
	}
	if err := file.Close(); err != nil || os.Rename(name, target) != nil {
		_ = os.Remove(name)
		return false
	}
	return true
}

func (c *PublicationCache) Discard(file *os.File) {
	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
}

func (c *PublicationCache) MaintainContext(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries, err := os.ReadDir(c.directory)
	if err != nil {
		return
	}
	type cachedFile struct {
		path string
		size int64
		when time.Time
	}
	files := make([]cachedFile, 0, len(entries))
	var total int64
	now := time.Now()
	for _, entry := range entries {
		if ctx.Err() != nil {
			return
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".book" {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		path := filepath.Join(c.directory, entry.Name())
		if c.expireAfter > 0 && now.Sub(info.ModTime()) > c.expireAfter {
			_ = os.Remove(path)
			continue
		}
		files = append(files, cachedFile{path: path, size: info.Size(), when: info.ModTime()})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].when.Before(files[j].when) })
	removed := 0
	for _, file := range files {
		if ctx.Err() != nil {
			return
		}
		if total <= c.maximumBytes {
			break
		}
		if os.Remove(file.path) == nil {
			total -= file.size
			removed++
		}
	}
	if removed > 0 && c.logger != nil {
		c.logger.Info("publication cache pruned", "removed", removed, "remaining_bytes", total, "maximum_bytes", c.maximumBytes)
	}
}
