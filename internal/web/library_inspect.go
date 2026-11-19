package web

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// maximumInspectedEntries and maximumInspectDepth bound the walk so a huge
	// tree answers in bounded time instead of hanging the request.
	maximumInspectedEntries = 20000
	maximumInspectDepth     = 6
)

// librarySuggestion is what the folder looks like it should become. There is no
// id: the settings page derives one from the name the user settles on, so the
// suggestion cannot disagree with the id rule that actually applies.
type librarySuggestion struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Storage string `json:"storage"`
}

// libraryFileCounts tallies the files a library would pick up. Images are
// counted so the card can mention them, but they never decide the kind.
type libraryFileCounts struct {
	Books     int `json:"books"`
	Comics    int `json:"comics"`
	Audio     int `json:"audio"`
	Video     int `json:"video"`
	Documents int `json:"documents"`
	Images    int `json:"images"`
}

type libraryInspection struct {
	Path      string            `json:"path"`
	Suggested librarySuggestion `json:"suggested"`
	Evidence  []string          `json:"evidence"`
}

// libraryFileCategory names the counting bucket for a file, or "" when the
// library would not serve it.
func libraryFileCategory(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".cbz", ".cbr", ".cb7":
		return "comics"
	case ".mp4", ".mkv", ".avi", ".m4v", ".webm":
		return "video"
	case ".mp3", ".m4a", ".m4b", ".flac", ".ogg":
		return "audio"
	case ".md", ".markdown":
		return "documents"
	case ".epub", ".pdf", ".mobi", ".azw3", ".azw":
		return "books"
	case ".jpg", ".jpeg", ".png", ".webp":
		return "images"
	}
	return ""
}

func (counts *libraryFileCounts) add(name string) {
	switch libraryFileCategory(name) {
	case "books":
		counts.Books++
	case "comics":
		counts.Comics++
	case "audio":
		counts.Audio++
	case "video":
		counts.Video++
	case "documents":
		counts.Documents++
	case "images":
		counts.Images++
	}
}

func (counts libraryFileCounts) total() int {
	return counts.Books + counts.Comics + counts.Audio + counts.Video + counts.Documents + counts.Images
}

// kind picks the library kind the files suggest. Largest group wins; on a tie
// the earlier group wins, and a folder with nothing recognisable suggests
// books. Images are counted but never chosen — a cover dump is not a library.
func (counts libraryFileCounts) kind() string {
	best, bestCount := "books", counts.Books
	for _, candidate := range []struct {
		kind  string
		count int
	}{
		{"comics", counts.Comics},
		{"movies", counts.Video},
		{"audiobooks", counts.Audio},
		{"vault", counts.Documents},
	} {
		if candidate.count > bestCount {
			best, bestCount = candidate.kind, candidate.count
		}
	}
	return best
}

func (counts libraryFileCounts) evidence(calibre bool, visited int) []string {
	evidence := make([]string, 0, 8)
	if calibre {
		evidence = append(evidence, "Calibre database found")
	} else if visited > 0 && counts.Books+counts.Comics > 0 {
		// Only a folder that holds publications could have been a Calibre
		// library, so a video or notes folder is not told about one.
		evidence = append(evidence, "No Calibre database")
	}
	for _, group := range []struct {
		count            int
		singular, plural string
	}{
		{counts.Books, "book file", "book files"},
		{counts.Comics, "comic file", "comic files"},
		{counts.Video, "video file", "video files"},
		{counts.Audio, "audio file", "audio files"},
		{counts.Documents, "markdown note", "markdown notes"},
		{counts.Images, "image", "images"},
	} {
		if group.count == 1 {
			evidence = append(evidence, "1 "+group.singular)
		} else if group.count > 1 {
			evidence = append(evidence, fmt.Sprintf("%d %s", group.count, group.plural))
		}
	}
	if visited == 0 {
		evidence = append(evidence, "Empty folder")
	} else if counts.total() == 0 {
		evidence = append(evidence, "No library files in this folder")
	}
	return evidence
}

// inspectLibraryFolder walks one folder and never leaves it: hidden entries are
// skipped, symbolic links are not followed, and both the number of entries and
// the depth are capped.
func inspectLibraryFolder(root string) (counts libraryFileCounts, calibre bool, visited int, err error) {
	separator := string(filepath.Separator)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root {
				return walkErr
			}
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		visited++
		if visited > maximumInspectedEntries {
			return fs.SkipAll
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		depth := strings.Count(relative, separator)
		if entry.IsDir() {
			if depth >= maximumInspectDepth {
				return fs.SkipDir
			}
			// WalkDir uses Lstat and never descends into a symbolic link, so a
			// link cannot carry the walk out of the folder.
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		if depth == 0 && entry.Name() == "metadata.db" {
			calibre = true
			return nil
		}
		counts.add(entry.Name())
		return nil
	})
	return counts, calibre, visited, err
}

// suggestedLibraryName is the folder's own name; a filesystem root has none, so
// it becomes "Library".
func suggestedLibraryName(path string) string {
	name := strings.TrimSpace(filepath.Base(filepath.Clean(path)))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "Library"
	}
	return name
}

// apiLibraryInspect reads one folder and suggests a library for it. It is
// read-only, administrative, and confined to the configured browse roots
// exactly like /api/folders.
func (m *multiServer) apiLibraryInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	if requested == "" {
		http.Error(w, "Library path is required", http.StatusBadRequest)
		return
	}
	if !filepath.IsAbs(requested) {
		http.Error(w, "Library path must be absolute", http.StatusBadRequest)
		return
	}
	requested = filepath.Clean(requested)
	resolved, err := filepath.EvalSymlinks(requested)
	if err != nil {
		http.Error(w, "Library folder was not found", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		http.Error(w, "Library path must be a folder", http.StatusBadRequest)
		return
	}
	var rootResolved string
	for _, real := range resolvedBrowseRoots(m.cfg.BrowseRoots) {
		if withinRoot(resolved, real) && len(real) >= len(rootResolved) {
			rootResolved = real
		}
	}
	if rootResolved == "" {
		http.Error(w, "Library folder is outside the browsable locations", http.StatusBadRequest)
		return
	}
	counts, calibre, visited, err := inspectLibraryFolder(resolved)
	if err != nil {
		http.Error(w, "Library folder cannot be read", http.StatusForbidden)
		return
	}
	name := suggestedLibraryName(resolved)
	kind := counts.kind()
	storage := "filesystem"
	if calibre {
		storage, kind = "calibre", "books"
	}
	writeJSON(w, http.StatusOK, libraryInspection{
		Path: requested,
		Suggested: librarySuggestion{
			Name:    name,
			Kind:    kind,
			Storage: storage,
		},
		Evidence: counts.evidence(calibre, visited),
	})
}
