// Package changelog holds the release history shown in the apps and on the web. The data lives in changelog.json,
// newest first, and is edited by hand with each release.
package changelog

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed changelog.json
var data []byte

// Entry is one release of one part of the system ("android", "server" or "web").
type Entry struct {
	Component   string   `json:"component"`
	Version     string   `json:"version"`
	Date        string   `json:"date"`
	New         []string `json:"new,omitempty"`
	Fixed       []string `json:"fixed,omitempty"`
	Performance []string `json:"performance,omitempty"`
	// Changes are notes that are not sorted into new, fixed or performance.
	Changes []string `json:"changes,omitempty"`
	TV      bool     `json:"tv,omitempty"`
}

var (
	once    sync.Once
	entries []Entry
)

func load() {
	var file struct {
		Entries []Entry `json:"entries"`
	}
	if json.Unmarshal(data, &file) == nil {
		entries = file.Entries
	}
}

// List returns the entries newest first, optionally only one component's, and at most limit when limit > 0.
func List(component string, limit int) []Entry {
	once.Do(load)
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if component != "" && e.Component != component {
			continue
		}
		out = append(out, e)
		if limit > 0 && len(out) == limit {
			break
		}
	}
	return out
}

// For returns the entry of one release.
func For(component, version string) (Entry, bool) {
	once.Do(load)
	for _, e := range entries {
		if e.Component == component && e.Version == version {
			return e, true
		}
	}
	return Entry{}, false
}
