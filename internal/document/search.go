package document

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"mochil/internal/statefile"
)

const (
	textIndexVersion   = 1
	maximumIndexedText = int64(64 << 20)
	textIndexWorkers   = 4
)

type TextCoverage struct {
	Indexed  int  `json:"indexed"`
	Eligible int  `json:"eligible"`
	Complete bool `json:"complete"`
}

type textEntry struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
	Content  string `json:"content"`
	// Folded is Content lower-cased once, when the index is built. Searching used to lower-case every document on every
	// query, inside the read lock: at 20,000 notes that was 82 MB and 111 ms per keystroke, and it blocked other readers
	// for the whole scan. It is not persisted -- the cache file keeps its size and the fold is recomputed when it loads,
	// which is one pass at startup instead of one pass per keystroke.
	Folded string `json:"-"`
}

// folded returns the searchable form of an entry, computing it for an entry that was loaded from an older cache.
func (e textEntry) folded() string {
	if e.Folded != "" {
		return e.Folded
	}
	return strings.ToLower(e.Content)
}

type textIndexFile struct {
	Version int         `json:"version"`
	Entries []textEntry `json:"entries"`
	Built   time.Time   `json:"built"`
}

type TextIndex struct {
	store           *Store
	cache           string
	buildMu         sync.Mutex
	backlinkMu      sync.Mutex
	tagMu           sync.Mutex
	tags            []TagGroup
	tagCache        map[string]cachedTags
	tagVersion      uint64
	mu              sync.RWMutex
	version         uint64
	entries         map[string]textEntry
	cover           TextCoverage
	backlinks       map[string][]string
	linkCache       map[string]cachedLinks
	backlinkVersion uint64
}

func NewTextIndex(store *Store, cache string) *TextIndex {
	index := &TextIndex{store: store, cache: cache, entries: make(map[string]textEntry)}
	for _, entry := range index.load() {
		entry.Folded = strings.ToLower(entry.Content)
		index.entries[entry.ID] = entry
	}
	return index
}

// cachedLinks is the distinct wiki-link targets of one page at one revision.
type cachedLinks struct {
	revision string
	targets  []string
}

// backlinkBuildHook runs between the snapshot and the publish. It is nil in
// production; a test sets it to publish a new index mid-build and prove the retry.
var backlinkBuildHook func()

// Backlinks returns indexed Markdown documents linking to any normalized
// Obsidian target. The reverse index is built once per published store version
// and remains bounded by the existing 64 MiB text-index budget.
func (i *TextIndex) Backlinks(targets ...string) ([]string, error) {
	if err := i.ensure(); err != nil {
		return nil, err
	}
	i.backlinkMu.Lock()
	defer i.backlinkMu.Unlock()
	// A build that finishes after the index moved on describes a version nobody
	// reads any more: publishing it is refused, and answering from the old map
	// would return a backlink list that is missing whatever was just added. So
	// rebuild against the newer snapshot instead of serving the stale one.
	for attempt := 0; attempt < 4; attempt++ {
		if reverse, ok := i.publishedBacklinks(); ok {
			return collectBacklinks(reverse, targets), nil
		}
		reverse, links, version := i.buildBacklinks()
		i.mu.Lock()
		if i.version == version {
			i.linkCache, i.backlinks, i.backlinkVersion = links, reverse, version
			i.mu.Unlock()
			return collectBacklinks(reverse, targets), nil
		}
		i.mu.Unlock()
	}
	// The index kept moving under us; answer from whatever is published rather
	// than failing the request.
	i.mu.Lock()
	defer i.mu.Unlock()
	return collectBacklinks(i.backlinks, targets), nil
}

// publishedBacklinks returns the reverse index when it matches the current
// version, which is the common case once one build has happened.
func (i *TextIndex) publishedBacklinks() (map[string][]string, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.backlinks == nil || i.backlinkVersion != i.version {
		return nil, false
	}
	return i.backlinks, true
}

// buildBacklinks snapshots the entries and builds the reverse index without
// holding the index lock, so a slow build never blocks a reader or a writer.
func (i *TextIndex) buildBacklinks() (map[string][]string, map[string]cachedLinks, uint64) {
	i.mu.RLock()
	entries := make(map[string]textEntry, len(i.entries))
	for id, entry := range i.entries {
		entries[id] = entry
	}
	cached := i.linkCache
	version := i.version
	i.mu.RUnlock()
	if backlinkBuildHook != nil {
		backlinkBuildHook()
	}
	reverse := make(map[string][]string)
	links := make(map[string]cachedLinks, len(entries))
	for id, entry := range entries {
		known, ok := cached[id]
		if !ok || known.revision != entry.Revision {
			known = cachedLinks{revision: entry.Revision}
			seen := make(map[string]struct{})
			for _, target := range wikiLinkTargets(entry.Content) {
				if _, exists := seen[target]; !exists {
					seen[target] = struct{}{}
					known.targets = append(known.targets, target)
				}
			}
		}
		links[id] = known
		for _, target := range known.targets {
			reverse[target] = append(reverse[target], id)
		}
	}
	return reverse, links, version
}

// collectBacklinks resolves the requested targets against a reverse index.
func collectBacklinks(reverse map[string][]string, targets []string) []string {
	ids := make(map[string]struct{})
	for _, target := range targets {
		for _, id := range reverse[normalizeWikiLinkTarget(target)] {
			ids[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out
}

func wikiLinkTargets(content string) []string {
	result := make([]string, 0)
	text := markdownProse(content)
	start := -1
	alias := false
	for i := 0; i < len(text); {
		if text[i] == 0 || text[i] == '\n' || text[i] == '\r' {
			start, alias = -1, false
			i++
			continue
		}
		if i+1 < len(text) && text[i:i+2] == "[[" && !alias {
			start = i + 2
			alias = false
			i += 2
			continue
		}
		if start >= 0 && text[i] == '|' {
			alias = true
		}
		if start >= 0 && i+1 < len(text) && text[i:i+2] == "]]" {
			target := text[start:i]
			if separator := strings.IndexAny(target, "|#"); separator >= 0 {
				target = target[:separator]
			}
			if !strings.ContainsAny(target, "[]") {
				if normalized := normalizeWikiLinkTarget(target); normalized != "" {
					result = append(result, normalized)
				}
			}
			start, alias = -1, false
			i += 2
			continue
		}
		i++
	}
	return result
}

// markdownProse masks code instead of deleting it, so text on either side can
// never be joined into a link. Keep fence handling independent of tag parsing.
func markdownProse(content string) string {
	text := []byte(content)
	fenceChar, fenceLen := byte(0), 0
	for pos := 0; pos < len(text); {
		end := pos
		for end < len(text) && text[end] != '\n' {
			end++
		}
		line := strings.TrimSpace(string(text[pos:end]))
		masked := fenceChar != 0
		if masked {
			if run, rest := fenceRun(line, fenceChar); run >= fenceLen && strings.TrimSpace(rest) == "" {
				fenceChar, fenceLen = 0, 0
			}
		} else if char, run := fenceStart(line); run >= 3 && (char != '`' || !strings.ContainsRune(line[run:], '`')) {
			fenceChar, fenceLen, masked = char, run, true
		}
		if !masked && line == "" {
			// Blank lines separate Markdown paragraphs; code spans cannot cross them.
			masked = true
		}
		if masked {
			for i := pos; i < end; i++ {
				text[i] = 0
			}
			if end < len(text) {
				text[end] = 0
			}
		}
		pos = end + 1
	}
	// Record the next full run by length in a reverse scan. A backslash
	// escapes only the first backtick when a run is an opener; inside code,
	// however, the entire run remains available as a closer.
	if !strings.ContainsRune(string(text), '`') {
		return string(text)
	}
	links := make([]int, len(text)) // closer byte offset plus one; zero means none
	next := make(map[int]int)
	for i := len(text) - 1; i >= 0; {
		if text[i] == 0 {
			clear(next)
			i--
			continue
		}
		if text[i] != '`' {
			i--
			continue
		}
		end := i + 1
		for i >= 0 && text[i] == '`' {
			i--
		}
		start := i + 1
		length := end - start
		links[start] = next[length]
		if length > 1 {
			links[start+1] = next[length-1]
		}
		next[length] = start + 1
	}
	for i := 0; i < len(text); {
		if text[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(text) && text[i] == '`' {
			i++
		}
		slashes := 0
		for j := start - 1; j >= 0 && text[j] == '\\'; j-- {
			slashes++
		}
		if slashes%2 != 0 {
			start++
		}
		if start == i || links[start] == 0 {
			continue
		}
		end := links[start] - 1
		for end < len(text) && text[end] == '`' {
			end++
		}
		for j := start; j < end; j++ {
			text[j] = 0
		}
		i = end
	}
	return string(text)
}

func normalizeWikiLinkTarget(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.TrimSuffix(strings.ToLower(value), ".md")
	return strings.Trim(value, "/")
}

func (i *TextIndex) Search(query string) (map[string]string, TextCoverage, error) {
	if err := i.ensure(); err != nil {
		return nil, TextCoverage{}, err
	}
	needle := strings.ToLower(strings.TrimSpace(query))
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.searchLocked(needle), i.cover, nil
}

// searchLocked scans the entries the index already holds. Split out from Search so the scan itself can be tested and
// benchmarked without the cache and the store behind it.
func (i *TextIndex) searchLocked(needle string) map[string]string {
	matches := make(map[string]string)
	if needle != "" {
		for id, entry := range i.entries {
			folded := entry.folded()
			position := strings.Index(folded, needle)
			if position < 0 {
				continue
			}
			// Case folding can change the byte length ("\u1E9E" is three bytes,
			// its lower-case form two), so the folded offset is not an offset
			// into the original text.
			if len(folded) != len(entry.Content) {
				position = originalOffset(entry.Content, position)
			}
			matches[id] = textSnippet(entry.Content, position, len(needle))
		}
	}
	return matches
}

func (i *TextIndex) ensure() error {
	version := i.store.Version()
	i.mu.RLock()
	ready := i.version == version
	i.mu.RUnlock()
	if ready {
		return nil
	}
	i.buildMu.Lock()
	defer i.buildMu.Unlock()
	i.store.scanMu.Lock()
	defer i.store.scanMu.Unlock()
	version = i.store.Version()
	i.mu.RLock()
	ready = i.version == version
	i.mu.RUnlock()
	if ready {
		return nil
	}
	items := i.store.Items()
	i.mu.RLock()
	previous := make(map[string]textEntry, len(i.entries))
	for id, entry := range i.entries {
		previous[id] = entry
	}
	i.mu.RUnlock()
	entries := make(map[string]textEntry)
	jobs := make([]Item, 0)
	var reserved int64
	coverage := TextCoverage{}
	for _, item := range items {
		if !TextReadable(item.Format) {
			continue
		}
		coverage.Eligible++
		if old, ok := previous[item.ID]; ok && old.Revision == item.Revision && reserved+int64(len(old.Content)) <= maximumIndexedText {
			entries[item.ID] = old
			reserved += int64(len(old.Content))
			continue
		}
		reservation := min(item.Size, DefaultTextLimit)
		if officeTextReadable(item.Format) && item.Size > maximumOfficeContainer {
			continue
		}
		if officeTextReadable(item.Format) {
			reservation = DefaultTextLimit
		}
		// RTF carries uncompressed bytes, so its encoded size says nothing about its
		// text length; the decoder budget bounds what is indexed.
		if manageableTextSize(item) {
			if reserved+reservation > maximumIndexedText {
				continue
			}
			jobs = append(jobs, item)
			reserved += reservation
		}
	}
	type result struct {
		entry textEntry
		err   error
	}
	queue := make(chan Item)
	results := make(chan result)
	workers := min(textIndexWorkers, len(jobs))
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for item := range queue {
				content, err := readTextContained(i.store, item.ID)
				text := string(content)
				results <- result{entry: textEntry{ID: item.ID, Revision: item.Revision, Content: text, Folded: strings.ToLower(text)}, err: err}
			}
		}()
	}
	go func() {
		for _, item := range jobs {
			queue <- item
		}
		close(queue)
		wait.Wait()
		close(results)
	}()
	for result := range results {
		if result.err == nil {
			entries[result.entry.ID] = result.entry
		}
	}
	coverage.Indexed = len(entries)
	coverage.Complete = coverage.Indexed == coverage.Eligible
	list := make([]textEntry, 0, len(entries))
	for _, item := range items {
		if entry, ok := entries[item.ID]; ok {
			list = append(list, entry)
		}
	}
	if err := i.save(list); err != nil {
		return err
	}
	i.mu.Lock()
	i.entries, i.cover, i.version = entries, coverage, version
	i.mu.Unlock()
	return nil
}

// originalOffset maps a byte offset in strings.ToLower(content) back to content.
func originalOffset(content string, foldedPosition int) int {
	folded := 0
	for index, value := range content {
		if folded >= foldedPosition {
			return index
		}
		folded += len(strings.ToLower(string(value)))
	}
	return len(content)
}

func textSnippet(content string, bytePosition, needleBytes int) string {
	if !utf8.ValidString(content) {
		return ""
	}
	// Slice a bounded byte window first and count runes inside it, instead of converting the whole document to runes three
	// times per match: a 3.2 MB note cost 13 MB of allocation per hit.
	windowStart := max(0, bytePosition-(48*4))
	for windowStart > 0 && !utf8.RuneStart(content[windowStart]) {
		windowStart--
	}
	windowEnd := min(len(content), bytePosition+needleBytes+(72*4))
	for windowEnd < len(content) && !utf8.RuneStart(content[windowEnd]) {
		windowEnd++
	}
	window := content[windowStart:windowEnd]
	before := content[windowStart:min(bytePosition, len(content))]
	startRune := len([]rune(before))
	needleRunes := len([]rune(content[bytePosition:min(bytePosition+needleBytes, len(content))]))
	runes := []rune(window)
	start := max(0, startRune-48)
	end := min(len(runes), startRune+needleRunes+72)
	snippet := strings.Join(strings.Fields(string(runes[start:end])), " ")
	if windowStart > 0 {
		snippet = "…" + snippet
	}
	if windowEnd < len(content) {
		snippet += "…"
	}
	return snippet
}

func (i *TextIndex) load() []textEntry {
	var cached textIndexFile
	if statefile.DecodeJSON(i.cache, statefile.MaximumCatalogueBytes, &cached) != nil || cached.Version != textIndexVersion || len(cached.Entries) > 1_000_000 {
		return nil
	}
	return cached.Entries
}

func (i *TextIndex) save(entries []textEntry) error {
	if i.cache == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(i.cache), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(i.cache), ".document-text-index-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = statefile.EncodeJSON(temporary, statefile.MaximumCatalogueBytes, textIndexFile{Version: textIndexVersion, Entries: entries, Built: time.Now().UTC()}, false); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, i.cache)
}

// readTextContained keeps a parser panic on one malformed document from ending the index.
func readTextContained(store *Store, id string) (content []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("reading %s panicked: %v", id, recovered)
		}
	}()
	content, _, err = store.ReadText(id, DefaultTextLimit)
	return content, err
}
