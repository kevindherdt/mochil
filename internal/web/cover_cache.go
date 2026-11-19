package web

import (
	"bytes"
	"context"
	"image"
	"image/color"
	stddraw "image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const maxCoverPixels = 80_000_000

type coverCache struct {
	directory     string
	maximumBytes  int64
	maximumWidth  int
	maximumHeight int
	jpegQuality   int
	logger        *slog.Logger
	mu            sync.Mutex
	size          int64
	sizeReady     bool
	// optimizeHook and evictHook, when set (tests only), let tests observe and pace expensive cache work.
	optimizeHook func()
	evictHook    func()
}

type cachedCover struct {
	path     string
	size     int64
	modified time.Time
}

func newCoverCache(directory string, maximumBytes int64, maximumWidth, maximumHeight, jpegQuality int, logger *slog.Logger) *coverCache {
	return &coverCache{directory: directory, maximumBytes: maximumBytes, maximumWidth: maximumWidth, maximumHeight: maximumHeight, jpegQuality: jpegQuality, logger: logger}
}

func (c *coverCache) path(libraryID, bookID string) string {
	return filepath.Join(c.directory, libraryID, bookID+".jpg")
}

func (c *coverCache) legacyPath(libraryID, bookID string) string {
	return filepath.Join(c.directory, libraryID, bookID)
}

func (c *coverCache) read(libraryID, bookID string) ([]byte, bool) {
	path := c.path(libraryID, bookID)
	if data, err := os.ReadFile(path); err == nil {
		// Eviction orders by modification time, and touching the file on every hit
		// meant a synchronous metadata write per cover request. An hour of
		// resolution is plenty for that ordering.
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > time.Hour {
			_ = os.Chtimes(path, time.Now(), time.Now())
		}
		return data, true
	}
	legacy := c.legacyPath(libraryID, bookID)
	legacyData, err := os.ReadFile(legacy)
	if err != nil {
		return nil, false
	}
	// The CPU-bound decode/resize/encode runs outside the lock, same as
	// write(), so concurrent cover generation is not serialized onto one
	// goroutine.
	optimized, ok := c.optimize(legacyData)
	if !ok {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// A writer may have populated the optimized path while we worked.
	if data, err := os.ReadFile(path); err == nil {
		now := time.Now()
		_ = os.Chtimes(path, now, now)
		return data, true
	}
	if c.writeLocked(path, optimized) == nil {
		_ = os.Remove(legacy)
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now)
	return optimized, true
}

func (c *coverCache) write(libraryID, bookID string, original []byte) ([]byte, bool) {
	optimized, ok := c.optimize(original)
	if !ok {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishLocked(c.path(libraryID, bookID), optimized)
	return optimized, true
}

// publishLocked stores one cover file and keeps the byte accounting honest: a
// replacement counts as a delta, not as a second copy, and both full covers and
// derived thumbnails go through here so neither can slip past the budget.
func (c *coverCache) publishLocked(path string, data []byte) {
	var previousSize int64
	if info, err := os.Stat(path); err == nil {
		previousSize = info.Size()
	}
	if err := c.writeLocked(path, data); err != nil {
		return
	}
	if !c.sizeReady {
		return
	}
	c.size += int64(len(data)) - previousSize
	if c.maximumBytes > 0 && c.size > c.maximumBytes {
		c.evictLocked()
	}
}

func (c *coverCache) invalidate(libraryID, bookID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, target := range []string{c.path(libraryID, bookID), c.legacyPath(libraryID, bookID)} {
		var size int64
		if info, err := os.Stat(target); err == nil {
			size = info.Size()
		}
		if os.Remove(target) == nil && c.sizeReady {
			c.size -= size
			if c.size < 0 {
				c.size = 0
			}
		}
	}
}

func (c *coverCache) writeLocked(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func (c *coverCache) optimize(data []byte) ([]byte, bool) {
	return c.optimizeTo(data, c.maximumWidth, c.maximumHeight)
}

// thumbnail derives (and stores) a smaller cover for grids and rails from the
// already optimized full-size cover.
func (c *coverCache) thumbnail(libraryID, id string, full []byte, width int) ([]byte, bool) {
	small, ok := c.optimizeTo(full, width, width*3/2)
	if !ok {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishLocked(c.path(libraryID, id), small)
	return small, true
}

func (c *coverCache) optimizeTo(data []byte, maximumWidth, maximumHeight int) ([]byte, bool) {
	if c.optimizeHook != nil {
		c.optimizeHook()
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxCoverPixels {
		return nil, false
	}
	width, height := config.Width, config.Height
	scale := min(float64(maximumWidth)/float64(width), float64(maximumHeight)/float64(height))
	if scale > 1 {
		scale = 1
	}
	targetWidth, targetHeight := max(1, int(float64(width)*scale)), max(1, int(float64(height)*scale))
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	target := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	stddraw.Draw(target, target.Bounds(), &image.Uniform{C: color.RGBA{245, 239, 230, 255}}, image.Point{}, stddraw.Src)
	draw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Over, nil)
	var output bytes.Buffer
	if jpeg.Encode(&output, target, &jpeg.Options{Quality: c.jpegQuality}) != nil {
		return nil, false
	}
	return output.Bytes(), true
}

func (c *coverCache) maintain(ctx context.Context) {
	var migrated, before, after int64
	_ = filepath.WalkDir(c.directory, func(path string, entry fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || entry.IsDir() || filepath.Ext(path) == ".tmp" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		before += info.Size()
		if filepath.Ext(path) == ".jpg" {
			after += info.Size()
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		optimized, ok := c.optimize(data)
		if !ok {
			return nil
		}
		newPath := path + ".jpg"

		// Decode/resize/encode above without c.mu. Recheck the source under the
		// lock before replacing it: a normal write or invalidation may have won
		// while this background maintenance work was running.
		c.mu.Lock()
		currentInfo, statErr := os.Stat(path)
		if statErr != nil || !os.SameFile(info, currentInfo) || currentInfo.Size() != info.Size() || !currentInfo.ModTime().Equal(info.ModTime()) {
			c.mu.Unlock()
			return nil
		}
		targetInfo, targetErr := os.Stat(newPath)
		if targetErr == nil {
			// A concurrent request may already have created a regular optimized file.
			if targetInfo.Mode().IsRegular() && os.Remove(path) == nil {
				migrated++
			}
			c.mu.Unlock()
			return nil
		}
		if !os.IsNotExist(targetErr) {
			// Do not replace a destination that could not be safely inspected.
			c.mu.Unlock()
			return nil
		}
		if err := c.writeLocked(newPath, optimized); err != nil {
			c.mu.Unlock()
			return nil
		}
		if os.Remove(path) == nil {
			after += int64(len(optimized))
		}
		migrated++
		c.mu.Unlock()
		return nil
	})

	c.mu.Lock()
	if !c.sizeReady || migrated > 0 || (c.maximumBytes > 0 && c.size > c.maximumBytes) {
		c.evictLocked()
	}
	c.mu.Unlock()
	if migrated > 0 && c.logger != nil {
		c.logger.Info("cover cache optimized", "covers", migrated, "before_bytes", before, "after_bytes", after, "maximum_bytes", c.maximumBytes)
	}
}

func (c *coverCache) evictLocked() {
	if c.evictHook != nil {
		c.evictHook()
	}
	var files []cachedCover
	var total int64
	_ = filepath.WalkDir(c.directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) == ".tmp" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		files = append(files, cachedCover{path: path, size: info.Size(), modified: info.ModTime()})
		total += info.Size()
		return nil
	})
	c.size, c.sizeReady = total, true
	if c.maximumBytes <= 0 {
		return
	}
	if total <= c.maximumBytes {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modified.Before(files[j].modified) })
	for _, file := range files {
		if total <= c.maximumBytes {
			break
		}
		if os.Remove(file.path) == nil {
			total -= file.size
			c.size = total
		}
	}
}
