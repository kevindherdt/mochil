package enrichment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const maximumArtworkResponse int64 = 10 << 20

type ArtworkCache struct {
	directory string
	maximum   int64
	client    *http.Client
	mu        sync.Mutex // guards the in-flight table and cache eviction, never held during network or file transfer
	inflight  map[string]*artworkCall
	slots     chan struct{}
}

func NewArtworkCache(directory string, maximum int64, client *http.Client) *ArtworkCache {
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(request *http.Request, _ []*http.Request) error {
			if !ValidProviderArtworkURL(request.URL.String()) {
				return errors.New("artwork redirect left an approved image host")
			}
			return nil
		}}
	}
	return &ArtworkCache{directory: directory, maximum: maximum, client: client}
}

type artworkCall struct {
	done       chan struct{}
	path, mime string
	err        error
}

// maximumArtworkDownloads bounds how many remote images are fetched at once.
const maximumArtworkDownloads = 4

// Fetch returns a cached copy of an approved artwork image, downloading it once when it is missing. Cache hits never wait
// for a download, concurrent requests for the same image share one download, and waiting honours the caller's context.
func (c *ArtworkCache) Fetch(ctx context.Context, raw string) (string, string, error) {
	if !ValidProviderArtworkURL(raw) {
		return "", "", errors.New("unsupported artwork source")
	}
	if c.directory == "" || c.maximum <= 0 {
		return "", "", errors.New("artwork cache is unavailable")
	}
	digest := sha256.Sum256([]byte(raw))
	base := hex.EncodeToString(digest[:])
	if path, mime, ok := c.cached(base); ok {
		return path, mime, nil
	}
	c.mu.Lock()
	if c.inflight == nil {
		c.inflight = map[string]*artworkCall{}
		c.slots = make(chan struct{}, maximumArtworkDownloads)
	}
	if call, running := c.inflight[base]; running {
		c.mu.Unlock()
		select {
		case <-call.done:
			return call.path, call.mime, call.err
		case <-ctx.Done():
			return "", "", ctx.Err()
		}
	}
	call := &artworkCall{done: make(chan struct{})}
	c.inflight[base] = call
	slots := c.slots
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.inflight, base)
		c.mu.Unlock()
		close(call.done)
	}()
	select {
	case slots <- struct{}{}:
		defer func() { <-slots }()
	case <-ctx.Done():
		call.err = ctx.Err()
		return "", "", call.err
	}
	call.path, call.mime, call.err = c.download(ctx, raw, base)
	return call.path, call.mime, call.err
}

func (c *ArtworkCache) cached(base string) (string, string, bool) {
	for _, extension := range []string{".jpg", ".png", ".webp"} {
		path := filepath.Join(c.directory, base+extension)
		if _, err := os.Stat(path); err == nil {
			return path, mimeForExtension(extension), true
		}
	}
	return "", "", false
}

func (c *ArtworkCache) download(ctx context.Context, raw, base string) (string, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", "", err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("fetch artwork: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("artwork returned HTTP %d", response.StatusCode)
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	extension := extensionForMIME(contentType)
	if extension == "" {
		return "", "", errors.New("artwork response is not a supported image")
	}
	if response.ContentLength > maximumArtworkResponse {
		return "", "", errors.New("artwork response is too large")
	}
	if err := os.MkdirAll(c.directory, 0o750); err != nil {
		return "", "", err
	}
	temporary, err := os.CreateTemp(c.directory, ".artwork-*")
	if err != nil {
		return "", "", err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	written, copyErr := io.Copy(temporary, io.LimitReader(response.Body, maximumArtworkResponse+1))
	if copyErr != nil || written > maximumArtworkResponse {
		temporary.Close()
		return "", "", errors.New("artwork response exceeds its size limit")
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", "", err
	}
	if err := temporary.Close(); err != nil {
		return "", "", err
	}
	path := filepath.Join(c.directory, base+extension)
	if err := os.Rename(temporaryName, path); err != nil {
		return "", "", err
	}
	c.mu.Lock()
	c.evictLocked(path)
	c.mu.Unlock()
	return path, contentType, nil
}

func ValidTMDBArtworkURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Host == "image.tmdb.org" && strings.HasPrefix(parsed.Path, "/t/p/w342/") && parsed.RawQuery == "" && parsed.Fragment == ""
}

// ValidTMDBBackdropURL accepts only the wide backdrop size TMDB serves from its image host.
func ValidTMDBBackdropURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Host == "image.tmdb.org" && strings.HasPrefix(parsed.Path, "/t/p/w1280/") && parsed.RawQuery == "" && parsed.Fragment == ""
}

// ValidProviderArtworkURL confines server-side artwork fetches to image hosts
// emitted by supported metadata adapters. It deliberately rejects credentials,
// ports, query strings, and arbitrary redirects.
func ValidProviderArtworkURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" || parsed.Fragment != "" {
		return false
	}
	if ValidTMDBArtworkURL(raw) || ValidTMDBBackdropURL(raw) {
		return true
	}
	if parsed.Hostname() == "assets.hardcover.app" {
		return parsed.RawQuery == "" && strings.HasPrefix(parsed.EscapedPath(), "/")
	}
	if parsed.Hostname() != "covers.openlibrary.org" || parsed.RawQuery != "default=false" {
		return false
	}
	const prefix, suffix = "/b/id/", "-M.jpg"
	identifier := strings.TrimSuffix(strings.TrimPrefix(parsed.EscapedPath(), prefix), suffix)
	if identifier == "" || parsed.EscapedPath() != prefix+identifier+suffix {
		return false
	}
	for _, digit := range identifier {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func (c *ArtworkCache) evictLocked(protected string) {
	entries, _ := os.ReadDir(c.directory)
	type fileInfo struct {
		path     string
		size     int64
		modified time.Time
	}
	files := make([]fileInfo, 0, len(entries))
	var total int64
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		file := fileInfo{filepath.Join(c.directory, entry.Name()), info.Size(), info.ModTime()}
		files = append(files, file)
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modified.Before(files[j].modified) })
	for _, file := range files {
		if total <= c.maximum {
			break
		}
		if file.path == protected {
			continue
		}
		if os.Remove(file.path) == nil {
			total -= file.size
		}
	}
}

func extensionForMIME(value string) string {
	switch value {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	}
	return ""
}
func mimeForExtension(value string) string {
	switch value {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	}
	return "application/octet-stream"
}
