package web

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

var folderNamePattern = regexp.MustCompile(`^[^/\\]+$`)

// apiFoldersCreate makes one new, empty sub-folder inside an already-browsable
// folder. It never recurses (os.Mkdir, not MkdirAll) and never leaves the
// configured browse roots, exactly like the listing above.
func (m *multiServer) apiFoldersCreate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if decodeJSONRequest(w, r, 4096, &input, true) != nil {
		http.Error(w, "The request could not be read.", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 255 || name == "." || name == ".." || strings.HasPrefix(name, ".") || !folderNamePattern.MatchString(name) {
		http.Error(w, "Give the folder a plain name, without slashes or a leading dot.", http.StatusBadRequest)
		return
	}
	requested := filepath.Clean(strings.TrimSpace(input.Path))
	if !filepath.IsAbs(requested) {
		http.Error(w, "Folder path must be absolute", http.StatusBadRequest)
		return
	}
	resolved, err := filepath.EvalSymlinks(requested)
	if err != nil {
		http.Error(w, "Folder was not found", http.StatusNotFound)
		return
	}
	roots := resolvedBrowseRoots(m.cfg.BrowseRoots)
	within := false
	for _, real := range roots {
		if withinRoot(resolved, real) {
			within = true
			break
		}
	}
	if !within {
		http.Error(w, "Folder is outside the browsable locations", http.StatusForbidden)
		return
	}
	created := filepath.Join(resolved, name)
	if err := os.Mkdir(created, 0o755); err != nil {
		if os.IsExist(err) {
			http.Error(w, "A folder with that name already exists.", http.StatusConflict)
			return
		}
		http.Error(w, "The folder could not be created.", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, browsedFolder{Name: name, Path: filepath.Join(requested, name)})
}

// apiFolders lists sub-folders so administrators can pick a library location
// instead of typing it. It lists directories only, never files, and never
// leaves the configured browse roots. POST creates a new, empty sub-folder
// inside an already-browsable folder, for picking a destination that does
// not exist yet without leaving the browser for a shell.
func (m *multiServer) apiFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		m.apiFoldersCreate(w, r)
		return
	}
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
