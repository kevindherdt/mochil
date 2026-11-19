package document

import (
	"regexp"
	"sort"
	"strings"
)

// inlineTag matches #tag the way Obsidian does: it starts a word, may contain
// letters, digits, "_", "-" and "/", and cannot be digits only.
var inlineTag = regexp.MustCompile(`(?:^|[\s(])#([\p{L}\p{N}_/-]+)`)

// cachedTags remembers the tags of one page for the revision they were read from,
// so a save only costs the page that changed, not the whole vault.
type cachedTags struct {
	revision string
	tags     []string
}

// TagGroup lists the pages carrying one tag.
type TagGroup struct {
	Tag string   `json:"tag"`
	IDs []string `json:"ids"`
}

func cleanTag(value string) string {
	value = strings.ToLower(strings.Trim(strings.TrimSpace(value), `#"',`))
	value = strings.Trim(value, "/")
	if value == "" || strings.IndexFunc(value, func(r rune) bool { return r < '0' || r > '9' }) < 0 {
		return ""
	}
	return value
}

// splitFrontmatter separates a leading "---" block from the page body.
func splitFrontmatter(content string) (front, body string) {
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return "", content
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimRight(lines[index], "\r\n") == "---" {
			return strings.Join(lines[1:index], ""), strings.Join(lines[index+1:], "")
		}
	}
	return "", content
}

func frontmatterTags(front string) []string {
	var tags []string
	inList := false
	for _, raw := range strings.Split(front, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if inList {
			if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
				tags = append(tags, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
				continue
			}
			if trimmed == "" {
				continue
			}
			inList = false
		}
		key, value, found := strings.Cut(line, ":")
		key = strings.ToLower(strings.TrimSpace(key))
		if !found || (key != "tags" && key != "tag") || line != strings.TrimLeft(line, " \t") {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			inList = true
			continue
		}
		value = strings.Trim(value, "[]")
		tags = append(tags, strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })...)
	}
	return tags
}

// fenceStart reports the fence marker a line opens with, when it opens one.
func fenceStart(line string) (byte, int) {
	if line == "" || (line[0] != '`' && line[0] != '~') {
		return 0, 0
	}
	run, _ := fenceRun(line, line[0])
	if run < 3 {
		return 0, 0
	}
	return line[0], run
}

// fenceRun counts leading marker characters of one kind and returns the rest.
// A backtick fence's info string may not itself contain a backtick.
func fenceRun(line string, char byte) (int, string) {
	run := 0
	for run < len(line) && line[run] == char {
		run++
	}
	return run, line[run:]
}

// pageTags returns the distinct, lowercased tags of one Markdown page.
func pageTags(content string) []string {
	front, body := splitFrontmatter(content)
	seen := make(map[string]struct{})
	var tags []string
	add := func(value string) {
		if tag := cleanTag(value); tag != "" {
			if _, exists := seen[tag]; !exists {
				seen[tag] = struct{}{}
				tags = append(tags, tag)
			}
		}
	}
	for _, tag := range frontmatterTags(front) {
		add(tag)
	}
	fenceChar, fenceLen := byte(0), 0
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if fenceChar != 0 {
			// Only the fence's own marker closes it, and only when nothing else follows.
			if run, rest := fenceRun(trimmed, fenceChar); run >= fenceLen && strings.TrimSpace(rest) == "" {
				fenceChar, fenceLen = 0, 0
			}
			continue
		}
		if char, run := fenceStart(trimmed); run >= 3 {
			fenceChar, fenceLen = char, run
			continue
		}
		if strings.IndexByte(line, '#') < 0 {
			continue
		}
		// Inline code never carries tags.
		parts := strings.Split(line, "`")
		for part := 0; part < len(parts); part += 2 {
			for _, match := range inlineTag.FindAllStringSubmatch(parts[part], -1) {
				add(match[1])
			}
		}
	}
	return tags
}

// Tags groups indexed Markdown pages by tag. The result is rebuilt once per
// published index version, like Backlinks.
func (i *TextIndex) Tags() ([]TagGroup, error) {
	if err := i.ensure(); err != nil {
		return nil, err
	}
	i.tagMu.Lock()
	defer i.tagMu.Unlock()
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.tags != nil && i.tagVersion == i.version {
		return i.tags, nil
	}
	grouped := make(map[string][]string)
	cache := make(map[string]cachedTags, len(i.entries))
	for id, entry := range i.entries {
		known, ok := i.tagCache[id]
		if !ok || known.revision != entry.Revision {
			known = cachedTags{revision: entry.Revision, tags: pageTags(entry.Content)}
		}
		cache[id] = known
		for _, tag := range known.tags {
			grouped[tag] = append(grouped[tag], id)
		}
	}
	i.tagCache = cache
	groups := make([]TagGroup, 0, len(grouped))
	for tag, ids := range grouped {
		sort.Strings(ids)
		groups = append(groups, TagGroup{Tag: tag, IDs: ids})
	}
	sort.Slice(groups, func(a, b int) bool { return groups[a].Tag < groups[b].Tag })
	i.tags, i.tagVersion = groups, i.version
	return groups, nil
}
