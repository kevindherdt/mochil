package video

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ArtworkScope uint8

const (
	ArtworkItem ArtworkScope = iota
	ArtworkShow
	ArtworkSeason
)

var artworkExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}

// OpenSidecar preserves the nearest-parent lookup used for individual items.
func (s *Store) OpenSidecar(id string) (*os.File, string, bool) {
	return s.OpenSidecarFor(id, ArtworkItem)
}

// OpenSidecarFor resolves artwork within an explicit catalogue scope. Show and
// season lookups never drift into another season while searching for a match.
func (s *Store) OpenSidecarFor(id string, scope ArtworkScope) (*os.File, string, bool) {
	item, ok := s.Item(id)
	if !ok {
		return nil, "", false
	}
	root, err := filepath.Abs(s.root)
	if err != nil {
		return nil, "", false
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, "", false
	}

	for _, candidate := range sidecarCandidates(root, item, scope) {
		for _, extension := range artworkExtensions {
			resolved, resolveErr := filepath.EvalSymlinks(candidate.directory + string(filepath.Separator) + candidate.stem + extension)
			if resolveErr != nil || !pathWithin(resolvedRoot, resolved) {
				continue
			}
			file, openErr := os.Open(resolved)
			if openErr != nil {
				continue
			}
			info, statErr := file.Stat()
			if statErr == nil && info.Mode().IsRegular() && info.Size() <= 32<<20 {
				return file, extension, true
			}
			file.Close()
		}
	}
	return nil, "", false
}

type sidecarCandidate struct {
	directory string
	stem      string
}

func sidecarCandidates(root string, item Item, scope ArtworkScope) []sidecarCandidate {
	parts := strings.Split(filepath.ToSlash(item.Path), "/")
	itemDirectory := filepath.Dir(filepath.Join(root, filepath.FromSlash(item.Path)))
	standard := []string{"poster", "cover", "folder"}
	var candidates []sidecarCandidate
	add := func(directory string, stems ...string) {
		if !pathWithin(root, directory) {
			return
		}
		for _, stem := range stems {
			candidates = append(candidates, sidecarCandidate{directory: directory, stem: stem})
		}
	}

	switch scope {
	case ArtworkShow:
		if item.Show != "" && len(parts) >= 2 {
			add(filepath.Join(root, filepath.FromSlash(parts[0])), "show", "poster", "cover", "folder")
		}
	case ArtworkSeason:
		if item.Show == "" || len(parts) < 2 {
			return nil
		}
		showDirectory := filepath.Join(root, filepath.FromSlash(parts[0]))
		if folder := seasonFolder.FindStringSubmatch(parts[len(parts)-2]); len(folder) == 2 {
			add(itemDirectory, "season", "poster", "cover", "folder")
		}
		season := strconv.Itoa(item.Season)
		padded := season
		if item.Season >= 0 && item.Season < 10 {
			padded = "0" + season
		}
		add(showDirectory, "season"+padded+"-poster", "season"+season+"-poster", "season"+padded, "season"+season)
	case ArtworkItem:
		base := strings.TrimSuffix(filepath.Base(item.Path), filepath.Ext(item.Path))
		add(itemDirectory, base, base+"-poster", base+"-thumb")
		// A season poster describes the group, not an individual episode.
		// Episodic items without a file-specific image fall through to the
		// generated frame poster instead of repeating one image for every row.
		if item.Show != "" {
			break
		}
		for directory := itemDirectory; pathWithin(root, directory); directory = filepath.Dir(directory) {
			add(directory, standard...)
			if directory == root {
				break
			}
		}
	}
	return candidates
}
