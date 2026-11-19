package document

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const cacheVersion = 1

type Stats struct{ Items, Reused, Errors int }

type cacheFile struct {
	Version int       `json:"version"`
	Items   []Item    `json:"items"`
	Scanned time.Time `json:"scanned"`
}

type Store struct {
	libraryID, root, cache string
	mu                     sync.RWMutex
	scanMu                 sync.Mutex
	items                  []Item
	directories            []string
	recentItems            []Item
	byID                   map[string]Item
	titlePositions         map[string]int
	recentPositions        map[string]int
	markdownTitleCounts    map[string]int
	version                uint64
	text                   *TextIndex
	allowVaultAssets       bool
}

func NewStore(libraryID, root, cache string) *Store {
	return newStore(libraryID, root, cache, false)
}

func NewVaultStore(libraryID, root, cache string) *Store {
	return newStore(libraryID, root, cache, true)
}

func newStore(libraryID, root, cache string, allowVaultAssets bool) *Store {
	s := &Store{libraryID: libraryID, root: root, cache: cache, allowVaultAssets: allowVaultAssets}
	s.text = NewTextIndex(s, textCachePath(cache))
	if cached := s.load(); len(cached) > 0 {
		s.publish(cached)
	}
	return s
}

func textCachePath(cache string) string {
	if cache == "" {
		return ""
	}
	extension := filepath.Ext(cache)
	return strings.TrimSuffix(cache, extension) + "-text" + extension
}

func (s *Store) SearchText(query string) (map[string]string, TextCoverage, error) {
	return s.text.Search(query)
}

func (s *Store) Backlinks(item Item) ([]Item, error) {
	pathTarget := strings.TrimSuffix(filepath.ToSlash(item.Path), filepath.Ext(item.Path))
	targets := []string{pathTarget}
	if !strings.Contains(pathTarget, "/") || s.uniqueMarkdownTitle(item) {
		targets = append(targets, item.Title)
	}
	ids, err := s.text.Backlinks(targets...)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Item, 0, len(ids))
	for _, id := range ids {
		if id == item.ID {
			continue
		}
		if linked, ok := s.byID[id]; ok && IsMarkdown(linked.Format) {
			items = append(items, linked)
		}
	}
	sort.Slice(items, func(left, right int) bool {
		return strings.ToLower(items[left].Path) < strings.ToLower(items[right].Path)
	})
	if len(items) > 100 {
		items = items[:100]
	}
	return items, nil
}

// Tags lists every tag used by a Markdown page, with the pages that carry it.
func (s *Store) Tags() ([]TagGroup, error) {
	groups, err := s.text.Tags()
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TagGroup, 0, len(groups))
	for _, group := range groups {
		ids := make([]string, 0, len(group.IDs))
		for _, id := range group.IDs {
			if item, ok := s.byID[id]; ok && IsMarkdown(item.Format) {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			out = append(out, TagGroup{Tag: group.Tag, IDs: ids})
		}
	}
	return out, nil
}

func (s *Store) uniqueMarkdownTitle(item Item) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.markdownTitleCounts[strings.ToLower(item.Title)] <= 1
}

func (s *Store) Items() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Item(nil), s.items...)
}

// Directories returns local non-hidden folders discovered during the latest
// scan. Remote document sources do not currently expose empty directories.
func (s *Store) Directories() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.directories...)
}

// Page returns one stable slice from the store-owned title or recent index.
// afterID is a keyset cursor anchor; version lets callers reject cursors from
// an older published snapshot instead of silently skipping changed content.
func (s *Store) Page(sortName, afterID string, offset, limit int) (items []Item, total int, version uint64, anchorFound, hasMore bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ordered, positions := s.items, s.titlePositions
	if sortName == "recent" {
		ordered, positions = s.recentItems, s.recentPositions
	}
	start := max(offset, 0)
	anchorFound = afterID == ""
	if afterID != "" {
		position, ok := positions[afterID]
		if !ok {
			return nil, len(ordered), s.version, false, false
		}
		start, anchorFound = position+1, true
	}
	start = min(start, len(ordered))
	end := min(start+max(limit, 0), len(ordered))
	return append([]Item(nil), ordered[start:end]...), len(ordered), s.version, anchorFound, end < len(ordered)
}

// Count is the cheap summary path used by shelf discovery.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

func (s *Store) Item(id string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.byID[id]
	return item, ok
}

func (s *Store) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

func (s *Store) Scan() (Stats, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	root, err := filepath.Abs(s.root)
	if err != nil {
		return Stats{}, err
	}
	info, statErr := os.Stat(root)
	if statErr != nil || !info.IsDir() {
		items := s.Items()
		if len(items) == 0 {
			items = s.load()
			if len(items) > 0 {
				s.publish(items)
			}
		}
		if len(items) > 0 {
			return Stats{Items: len(items), Reused: len(items), Errors: 1}, nil
		}
		if statErr != nil {
			return Stats{}, fmt.Errorf("document library: %w", statErr)
		}
		return Stats{}, fmt.Errorf("document library is not a directory: %s", root)
	}
	previous := make(map[string]Item)
	for _, item := range s.Items() {
		previous[item.Path] = item
	}
	items := make([]Item, 0)
	directories := make([]string, 0)
	stats := Stats{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// One unreadable directory says nothing about the rest of the vault.
			stats.Errors++
			return nil
		}
		if path != root && entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path != root && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if path != root {
				relative, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				directories = append(directories, filepath.ToSlash(relative))
			}
			return nil
		}
		fileInfo, infoErr := entry.Info()
		if infoErr == nil && !fileInfo.Mode().IsRegular() {
			// A FIFO or device node in a synced vault blocks on open forever, and
			// one of them stalled text indexing and every later scan behind it.
			return nil
		}
		if infoErr != nil {
			// Deleted or renamed between ReadDir and this lstat, which a synced
			// vault does constantly. Dropping the whole walk for that would leave
			// the library empty and fail the save that triggered the refresh.
			stats.Errors++
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		item, ok := identify(s.libraryID, relative, fileInfo.Size(), fileInfo.ModTime(), s.allowVaultAssets)
		if !ok {
			return nil
		}
		if old, exists := previous[item.Path]; exists && old.Revision == item.Revision {
			item = old
			stats.Reused++
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return Stats{}, err
	}
	sort.Slice(items, func(i, j int) bool {
		return titleLess(items[i], items[j])
	})
	if err = s.save(items); err != nil {
		return Stats{}, err
	}
	s.publish(items)
	s.mu.Lock()
	s.directories = directories
	s.mu.Unlock()
	stats.Items = len(items)
	return stats, nil
}

func (s *Store) Open(id string) (*os.File, Item, error) {
	item, ok := s.Item(id)
	if !ok {
		return nil, Item{}, os.ErrNotExist
	}
	root, err := filepath.Abs(s.root)
	if err != nil {
		return nil, Item{}, err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, Item{}, err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(item.Path)))
	if err != nil {
		return nil, Item{}, err
	}
	relative, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, Item{}, os.ErrPermission
	}
	file, err := os.Open(resolved)
	return file, item, err
}

func (s *Store) publish(items []Item) {
	byID := make(map[string]Item, len(items))
	titlePositions := make(map[string]int, len(items))
	markdownTitleCounts := make(map[string]int)
	for index, item := range items {
		byID[item.ID] = item
		titlePositions[item.ID] = index
		if IsMarkdown(item.Format) {
			markdownTitleCounts[strings.ToLower(item.Title)]++
		}
	}
	recent := append([]Item(nil), items...)
	sort.SliceStable(recent, func(i, j int) bool {
		if !recent[i].Modified.Equal(recent[j].Modified) {
			return recent[i].Modified.After(recent[j].Modified)
		}
		return titleLess(recent[i], recent[j])
	})
	recentPositions := make(map[string]int, len(recent))
	for index, item := range recent {
		recentPositions[item.ID] = index
	}
	s.mu.Lock()
	s.items, s.recentItems, s.byID = append([]Item(nil), items...), recent, byID
	s.titlePositions, s.recentPositions, s.markdownTitleCounts, s.version = titlePositions, recentPositions, markdownTitleCounts, s.version+1
	s.mu.Unlock()
}

func titleLess(left, right Item) bool {
	leftTitle, rightTitle := strings.ToLower(left.Title), strings.ToLower(right.Title)
	if leftTitle != rightTitle {
		return leftTitle < rightTitle
	}
	return left.Path < right.Path
}

func (s *Store) load() []Item {
	var cached cacheFile
	if statefile.DecodeJSON(s.cache, statefile.MaximumCatalogueBytes, &cached) != nil || cached.Version != cacheVersion || len(cached.Items) > 1_000_000 {
		return nil
	}
	return cached.Items
}

func (s *Store) save(items []Item) error {
	if s.cache == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.cache), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.cache), ".document-cache-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = statefile.EncodeJSON(temporary, statefile.MaximumCatalogueBytes, cacheFile{Version: cacheVersion, Items: items, Scanned: time.Now().UTC()}, false); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, s.cache)
}

// Properties lists the front-matter values of Markdown pages, with the pages that carry each.
func (s *Store) Properties() ([]PropertyGroup, error) {
	groups, err := s.text.Properties()
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PropertyGroup, 0, len(groups))
	for _, group := range groups {
		ids := make([]string, 0, len(group.IDs))
		for _, id := range group.IDs {
			if item, ok := s.byID[id]; ok && IsMarkdown(item.Format) {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			out = append(out, PropertyGroup{Key: group.Key, Value: group.Value, IDs: ids})
		}
	}
	return out, nil
}
