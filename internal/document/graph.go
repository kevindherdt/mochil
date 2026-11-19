package document

import (
	"path/filepath"
	"sort"
	"strings"
)

// GraphNode is one Markdown page in the link graph.
type GraphNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// Graph is the vault's pages and the [[links]] between them. Edges are pairs of
// indexes into Nodes, from the linking page to the page it names.
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges [][2]int    `json:"edges"`
}

// linkTargets returns each page's distinct, normalized [[link]] targets, reusing
// the per-page cache the backlink index keeps.
func (i *TextIndex) linkTargets() (map[string]cachedLinks, error) {
	if _, err := i.Backlinks(); err != nil {
		return nil, err
	}
	i.backlinkMu.Lock()
	defer i.backlinkMu.Unlock()
	return i.linkCache, nil // replaced as a whole when rebuilt, never edited in place
}

func pageKey(path string) string {
	return normalizeWikiLinkTarget(strings.TrimSuffix(filepath.ToSlash(path), filepath.Ext(path)))
}

// Graph resolves every wiki link to a page the way Obsidian does: a full path
// first, then a page name that is unique in the vault. Links that name nothing,
// or something ambiguous, are left out.
func (s *Store) Graph() (Graph, error) {
	targets, err := s.text.linkTargets()
	if err != nil {
		return Graph{}, err
	}
	s.mu.RLock()
	pages := make([]Item, 0, len(s.byID))
	for _, item := range s.byID {
		if IsMarkdown(item.Format) {
			pages = append(pages, item)
		}
	}
	s.mu.RUnlock()
	sort.Slice(pages, func(a, b int) bool { return strings.ToLower(pages[a].Path) < strings.ToLower(pages[b].Path) })

	graph := Graph{Nodes: make([]GraphNode, len(pages)), Edges: make([][2]int, 0)}
	byPath := make(map[string]int, len(pages))
	byName := make(map[string]int, len(pages))
	for index, page := range pages {
		graph.Nodes[index] = GraphNode{ID: page.ID, Title: page.Title, Path: page.Path}
		key := pageKey(page.Path)
		byPath[key] = index
		name := key[strings.LastIndex(key, "/")+1:]
		if _, taken := byName[name]; taken {
			byName[name] = -1 // ambiguous: never guess
		} else {
			byName[name] = index
		}
	}
	for index, page := range pages {
		seen := make(map[int]struct{})
		for _, target := range targets[page.ID].targets {
			to, ok := byPath[target]
			if !ok && !strings.Contains(target, "/") {
				to, ok = byName[target]
			}
			if !ok || to < 0 || to == index {
				continue
			}
			if _, dup := seen[to]; !dup {
				seen[to] = struct{}{}
				graph.Edges = append(graph.Edges, [2]int{index, to})
			}
		}
	}
	return graph, nil
}
