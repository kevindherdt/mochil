package document

import (
	"sort"
	"strings"
)

// PropertyGroup lists the pages whose front matter gives one property a value, like "status: draft".
type PropertyGroup struct {
	Key   string   `json:"key"`
	Value string   `json:"value"`
	IDs   []string `json:"ids"`
}

// pageProperties reads the simple "key: value", "key: [a, b]" and "- item" lines of a page's front matter.
// Tags have their own index and are left out. Anything fancier is skipped, never guessed at.
func pageProperties(content string) [][2]string {
	front, _ := splitFrontmatter(content)
	var out [][2]string
	key := ""
	for _, raw := range strings.Split(front, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			if key != "" {
				if value := cleanPropertyValue(strings.TrimPrefix(trimmed, "-")); value != "" {
					out = append(out, [2]string{key, value})
				}
			}
			continue
		}
		name, value, found := strings.Cut(line, ":")
		key = ""
		if !found || line != strings.TrimLeft(line, " \t") {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || name == "tags" || name == "tag" {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			key = name
			continue
		}
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			for _, part := range strings.Split(strings.Trim(value, "[]"), ",") {
				if item := cleanPropertyValue(part); item != "" {
					out = append(out, [2]string{name, item})
				}
			}
			continue
		}
		if strings.ContainsAny(value[:1], "|>{&*!") {
			continue
		}
		if item := cleanPropertyValue(value); item != "" {
			out = append(out, [2]string{name, item})
		}
	}
	return out
}

func cleanPropertyValue(value string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(value), `"'`))
}

// Properties groups indexed Markdown pages by property and value.
func (i *TextIndex) Properties() ([]PropertyGroup, error) {
	if err := i.ensure(); err != nil {
		return nil, err
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	grouped := make(map[[2]string][]string)
	for id, entry := range i.entries {
		seen := make(map[[2]string]bool)
		for _, pair := range pageProperties(entry.Content) {
			pair[1] = strings.ToLower(pair[1])
			if !seen[pair] {
				seen[pair] = true
				grouped[pair] = append(grouped[pair], id)
			}
		}
	}
	groups := make([]PropertyGroup, 0, len(grouped))
	for pair, ids := range grouped {
		sort.Strings(ids)
		groups = append(groups, PropertyGroup{Key: pair[0], Value: pair[1], IDs: ids})
	}
	sort.Slice(groups, func(a, b int) bool {
		if groups[a].Key != groups[b].Key {
			return groups[a].Key < groups[b].Key
		}
		return groups[a].Value < groups[b].Value
	})
	return groups, nil
}
