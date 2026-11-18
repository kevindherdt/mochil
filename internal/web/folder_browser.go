package web

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maximumBrowsedFolders = 500

type browsedFolder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type folderListing struct {
	Path        string          `json:"path"`
	Parent      string          `json:"parent,omitempty"`
	Directories []browsedFolder `json:"directories"`
	Truncated   bool            `json:"truncated,omitempty"`
}

// resolvedBrowseRoots returns the configured roots that exist, with symlinks
// resolved so containment checks compare real locations.
func resolvedBrowseRoots(configured []string) map[string]string {
	roots := make(map[string]string, len(configured))
	for _, root := range configured {
		root = filepath.Clean(strings.TrimSpace(root))
		if !filepath.IsAbs(root) {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			if info, err := os.Stat(resolved); err == nil && info.IsDir() {
				roots[root] = resolved
			}
		}
	}
	return roots
}

func withinRoot(resolved, root string) bool {
	relative, err := filepath.Rel(root, resolved)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// apiFolders lists sub-folders so administrators can pick a library location
// instead of typing it. It lists directories only, never files, and never
// leaves the configured browse roots.
func (m *multiServer) apiFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	roots := resolvedBrowseRoots(m.cfg.BrowseRoots)
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	if requested == "" {
		listing := folderListing{Directories: []browsedFolder{}}
		for root := range roots {
			listing.Directories = append(listing.Directories, browsedFolder{Name: root, Path: root})
		}
		sort.Slice(listing.Directories, func(i, j int) bool { return listing.Directories[i].Path < listing.Directories[j].Path })
		writeJSON(w, http.StatusOK, listing)
		return
	}
	requested = filepath.Clean(requested)
	if !filepath.IsAbs(requested) {
		http.Error(w, "Folder path must be absolute", http.StatusBadRequest)
		return
	}
	resolved, err := filepath.EvalSymlinks(requested)
	if err != nil {
		http.Error(w, "Folder was not found", http.StatusNotFound)
		return
	}
	var rootPath, rootResolved string
	for configured, real := range roots {
		if withinRoot(resolved, real) && len(real) >= len(rootResolved) {
			rootPath, rootResolved = configured, real
		}
	}
	if rootPath == "" {
		http.Error(w, "Folder is outside the browsable locations", http.StatusForbidden)
		return
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		http.Error(w, "Folder cannot be read", http.StatusForbidden)
		return
	}
	listing := folderListing{Path: requested, Directories: []browsedFolder{}}
	if requested != rootPath {
		listing.Parent = filepath.Dir(requested)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		child := filepath.Join(resolved, entry.Name())
		if info, err := os.Stat(child); err != nil || !info.IsDir() {
			continue
		}
		if real, err := filepath.EvalSymlinks(child); err != nil || !withinRoot(real, rootResolved) {
			continue
		}
		if len(listing.Directories) == maximumBrowsedFolders {
			listing.Truncated = true
			break
		}
		listing.Directories = append(listing.Directories, browsedFolder{Name: entry.Name(), Path: filepath.Join(requested, entry.Name())})
	}
	sort.Slice(listing.Directories, func(i, j int) bool {
		return strings.ToLower(listing.Directories[i].Name) < strings.ToLower(listing.Directories[j].Name)
	})
	writeJSON(w, http.StatusOK, listing)
}
