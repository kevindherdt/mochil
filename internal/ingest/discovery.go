package ingest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	MaximumDiscoveryItems  = 100
	maximumDiscoveryVisits = 25000
	discoveryCacheTTL      = 5 * time.Second
)

// discoveredFile is one regular, non-symlink file found below the source
// root, cached between pagination calls so that consecutive DiscoverNamed
// calls don't each re-walk the whole tree just to skip past what a cursor has
// already returned.
type discoveredFile struct {
	relative string
	size     int64
}

// discoveryCache is a short-lived snapshot of one full walk of the source
// tree. It is rebuilt from scratch whenever it is missing or older than
// discoveryCacheTTL, so pagination that completes within the TTL (the normal
// case: a client paging through results) does one walk total instead of one
// walk per page, while a cache that outlives the TTL never serves a listing
// that is more than discoveryCacheTTL stale.
type discoveryCache struct {
	builtAt   time.Time
	files     []discoveredFile // sorted by relative path: DiscoverNamed's cursor is a lexicographic comparison, which WalkDir's own visit order does not always match
	truncated bool
}

// discoveryListing returns the cached walk of the source tree, rebuilding it
// if the cache is missing, stale, or in flight for the first time.
func (e *Executor) discoveryListing(ctx context.Context) (*discoveryCache, error) {
	e.discoveryCacheMu.Lock()
	defer e.discoveryCacheMu.Unlock()
	if e.discoveryCache != nil && e.now().Sub(e.discoveryCache.builtAt) < discoveryCacheTTL {
		return e.discoveryCache, nil
	}
	cache := &discoveryCache{builtAt: e.now()}
	visits := 0
	err := filepath.WalkDir(e.sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		visits++
		if e.discoveryVisitHook != nil {
			e.discoveryVisitHook()
		}
		if visits > maximumDiscoveryVisits {
			cache.truncated = true
			return fs.SkipAll
		}
		if path == e.sourceRoot || entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		relative, err := filepath.Rel(e.sourceRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		cache.files = append(cache.files, discoveredFile{relative: relative, size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	// WalkDir visits a directory's contents before a sibling file such as "A!.mp3"
	// (a byte below '/' after the directory name), so its order is not the string
	// order the page cursor compares against; without this a later page skips it.
	sort.Slice(cache.files, func(i, j int) bool { return cache.files[i].relative < cache.files[j].relative })
	e.discoveryCache = cache
	return cache, nil
}

type DiscoveryItem struct {
	SourcePath      string   `json:"source_path"`
	Size            int64    `json:"size"`
	Kind            Kind     `json:"kind"`
	Proposal        Proposal `json:"proposal"`
	SuggestedTarget string   `json:"suggested_target"`
}

type DiscoveryPage struct {
	Items      []DiscoveryItem `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
	Truncated  bool            `json:"truncated,omitempty"`
}

// Discover returns a bounded, stable, read-only inventory of recognized media
// below the incoming root. Directory symlinks are never followed. Suggestions
// are previews only and still pass through Plan's confinement checks if used.
func (e *Executor) Discover(ctx context.Context, requested Kind, after string, limit int) (DiscoveryPage, error) {
	return e.DiscoverNamed(ctx, requested, after, limit, NamingDefault)
}

// DiscoverNamed is Discover with the managed-library layout the suggested targets follow.
func (e *Executor) DiscoverNamed(ctx context.Context, requested Kind, after string, limit int, naming string) (DiscoveryPage, error) {
	if !ValidNamingPreset(naming) {
		return DiscoveryPage{}, errors.New("unsupported naming preset")
	}
	if requested != "" && requested != Publication && requested != Music && requested != Video {
		return DiscoveryPage{}, errors.New("unsupported discovery kind")
	}
	if after != "" && !safeJournalPath(after) {
		return DiscoveryPage{}, errors.New("invalid discovery cursor")
	}
	if limit < 1 || limit > MaximumDiscoveryItems {
		limit = MaximumDiscoveryItems
	}
	if err := ctx.Err(); err != nil {
		return DiscoveryPage{}, err
	}
	listing, err := e.discoveryListing(ctx)
	if err != nil {
		return DiscoveryPage{}, err
	}
	containers := map[string]bool{}
	items := make([]DiscoveryItem, 0, limit+1)
	for _, file := range listing.files {
		if file.relative <= after {
			continue
		}
		kind, recognized := kindForExtension(filepath.Ext(file.relative))
		if !recognized || (requested != "" && requested != kind) {
			continue
		}
		if file.size > e.maxFileBytes {
			continue
		}
		proposal := InferPath(kind, file.relative)
		if kind == Publication {
			proposal = dropContainerAuthor(proposal, file.relative, e.sourceRoot, containers)
		}
		if kind == Publication && strings.EqualFold(filepath.Ext(file.relative), ".epub") {
			// the book's own metadata beats a guess from its name; an unreadable book keeps the guess
			path := filepath.Join(e.sourceRoot, filepath.FromSlash(file.relative))
			if embedded, err := ReadEmbedded(path); err == nil {
				proposal = Combine(proposal, embedded)
			}
		}
		target, _ := TargetFor(naming, file.relative, proposal) // a book that cannot be named is listed without a target, for review
		items = append(items, DiscoveryItem{SourcePath: file.relative, Size: file.size, Kind: kind, Proposal: proposal, SuggestedTarget: target})
		if len(items) > limit {
			break
		}
	}
	page := DiscoveryPage{Items: items}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.NextCursor = page.Items[len(page.Items)-1].SourcePath
	}
	if listing.truncated {
		page.Truncated = true
	}
	return page, nil
}

func kindForExtension(extension string) (Kind, bool) {
	switch strings.ToLower(extension) {
	case ".epub", ".pdf", ".cbz", ".cbr", ".mobi", ".azw3":
		return Publication, true
	case ".mp3", ".m4a", ".m4b", ".flac", ".ogg", ".opus", ".wav", ".aac":
		return Music, true
	case ".mp4", ".mkv", ".webm", ".avi", ".mov", ".m4v":
		return Video, true
	default:
		return "", false
	}
}

func suggestedTarget(source string, proposal Proposal) string {
	extension := strings.ToLower(filepath.Ext(source))
	component := func(value string) string {
		value = strings.TrimSpace(value)
		value = strings.Map(func(character rune) rune {
			if character == '/' || character == '\\' || character < 32 {
				return '-'
			}
			return character
		}, value)
		return strings.Trim(value, " .")
	}
	var parts []string
	switch proposal.Kind {
	case Publication:
		// A book with no title cannot be named. Falling through would build a target out of the author alone (the file part
		// is empty, and the clean loop drops it), which plan and execute would happily write as a regular file named after
		// the author: copied, marked imported, and never indexed because the catalogue only reads known extensions. An
		// empty target is the honest answer — it puts the row back in front of a person.
		if component(proposal.Title) == "" {
			return ""
		}
		if len(proposal.Authors) > 0 {
			parts = append(parts, component(proposal.Authors[0]))
		}
		if proposal.Series != "" {
			parts = append(parts, component(proposal.Series))
		}
		name := component(proposal.Title)
		if proposal.SeriesIndex != nil {
			volume := "Vol " + strconv.FormatFloat(*proposal.SeriesIndex, 'f', -1, 64)
			if proposal.Series != "" {
				volume = component(proposal.Series) + " - " + volume
			}
			// a title that only repeats the series and volume ("Series - Volume 01") adds nothing to the name
			if lean := leanTitle(proposal.Series, proposal.Title); lean != "" {
				name = volume + " - " + component(lean)
			} else {
				name = volume
			}
		}
		parts = append(parts, name+extension)
	case Music:
		parts = append(parts, component(proposal.AlbumArtist), component(proposal.Album))
		prefix := ""
		if proposal.Track > 0 {
			prefix = fmt.Sprintf("%02d - ", proposal.Track)
		}
		parts = append(parts, prefix+component(proposal.Title)+extension)
	case Video:
		if proposal.Series != "" && proposal.Episode > 0 {
			parts = append(parts, component(proposal.Series), fmt.Sprintf("Season %02d", proposal.Season), fmt.Sprintf("%s - S%02dE%02d - %s%s", component(proposal.Series), proposal.Season, proposal.Episode, component(proposal.Title), extension))
		} else {
			name := component(proposal.Title)
			if proposal.Year > 0 {
				name += fmt.Sprintf(" (%d)", proposal.Year)
			}
			parts = append(parts, name+extension)
		}
	}
	clean := parts[:0]
	for _, part := range parts {
		if part != "" && part != extension {
			clean = append(clean, part)
		}
	}
	if len(clean) == 0 {
		return filepath.ToSlash(filepath.Clean(source))
	}
	return strings.Join(clean, "/")
}

var leadingVolume = regexp.MustCompile(`(?i)^(?:v(?:ol(?:ume)?)?\.?|book|#)\s*\d+(?:\.\d+)?\b`)

// leanTitle removes a leading series name and volume marker from a title, leaving only what is particular to the book.
func leanTitle(series, title string) string {
	title = strings.TrimSpace(title)
	if series = strings.TrimSpace(series); series != "" && len(title) >= len(series) && strings.EqualFold(title[:len(series)], series) {
		title = strings.TrimLeft(title[len(series):], " ,.:;-–—_")
	}
	title = strings.TrimLeft(leadingVolume.ReplaceAllString(title, ""), " ,.:;-–—_")
	return strings.TrimSpace(title)
}

// maximumAuthorFolders is how many folders an author's own folder may hold. A folder with more is a batch or category
// folder (a whole drop of series), and its name is not an author.
const maximumAuthorFolders = 12

// dropContainerAuthor removes an author that was only guessed from a top-level folder holding many series.
func dropContainerAuthor(proposal Proposal, relative, root string, cache map[string]bool) Proposal {
	parts := strings.Split(relative, "/")
	if len(parts) != 3 || len(proposal.Authors) != 1 || proposal.Authors[0] != display(parts[0]) {
		return proposal
	}
	container, known := cache[parts[0]]
	if !known {
		entries, _ := os.ReadDir(filepath.Join(root, parts[0]))
		folders := 0
		for _, entry := range entries {
			if entry.IsDir() {
				folders++
			}
		}
		container = folders > maximumAuthorFolders
		cache[parts[0]] = container
	}
	if !container {
		return proposal
	}
	proposal.Authors = nil
	kept := proposal.Evidence[:0:0]
	for _, evidence := range proposal.Evidence {
		if evidence.Field != "authors" {
			kept = append(kept, evidence)
		}
	}
	proposal.Evidence = kept
	return proposal
}
