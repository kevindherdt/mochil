package wiki

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

func (s *Store) List(account string, deleted bool) []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.data.Accounts[account]
	if a == nil {
		return []Page{}
	}
	out := make([]Page, 0, len(a.Pages))
	for _, page := range a.Pages {
		if (page.DeletedAt != nil) == deleted {
			page = clonePage(page)
			page.Content = ""
			out = append(out, page)
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	return out
}

func (s *Store) Get(account, id string) (Page, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.data.Accounts[account]
	if a == nil {
		return Page{}, false
	}
	p, ok := a.Pages[id]
	return clonePage(p), ok
}

func (s *Store) Backlinks(account, title string) []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.data.Accounts[account]
	if a == nil {
		return []Page{}
	}
	needle := "[[" + strings.ToLower(strings.TrimSpace(title)) + "]]"
	out := []Page{}
	for _, page := range a.Pages {
		if page.DeletedAt == nil && strings.Contains(strings.ToLower(page.Content), needle) {
			page = clonePage(page)
			page.Content = ""
			out = append(out, page)
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	return out
}

func clonePage(page Page) Page {
	if page.DeletedAt != nil {
		deletedAt := *page.DeletedAt
		page.DeletedAt = &deletedAt
	}
	return page
}

func (s *Store) ListFolders(account string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.data.Accounts[account]
	if a == nil {
		return []string{}
	}
	out := append([]string{}, a.Folders...)
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

func (s *Store) History(account, id string) ([]Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.data.Accounts[account]
	if a == nil {
		return nil, ErrNotFound
	}
	if _, ok := a.Pages[id]; !ok {
		return nil, ErrNotFound
	}
	return append([]Snapshot(nil), a.History[id]...), nil
}

func (s *Store) Export(account string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.data.Accounts[account]
	if a == nil {
		a = &accountData{}
		normalize(a)
	}
	return json.MarshalIndent(struct {
		Version    int                   `json:"version"`
		ExportedAt time.Time             `json:"exported_at"`
		Pages      map[string]Page       `json:"pages"`
		History    map[string][]Snapshot `json:"history"`
		Folders    []string              `json:"folders,omitempty"`
	}{1, time.Now().UTC(), a.Pages, a.History, a.Folders}, "", "  ")
}
