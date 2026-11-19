package audio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"mochil/internal/artwork"
)

type ArtworkStore struct{ generator *artwork.Generator }

type ArtworkVariant string

const (
	ArtworkCard   ArtworkVariant = "card"
	ArtworkPlayer ArtworkVariant = "player"
)

func NewArtworkStore(root, command string, maxBytes int64, workers int) (*ArtworkStore, error) {
	generator, err := artwork.NewGenerator(root, command, maxBytes, workers)
	if err != nil {
		return nil, err
	}
	return &ArtworkStore{generator}, nil
}
func (s *ArtworkStore) Cover(file *os.File, item Item) (string, error) {
	return s.CoverVariant(file, item, ArtworkCard, item.Revision)
}

// CoverVariant extracts or resizes artwork into one of the bounded display
// profiles. sourceRevision must change when a sidecar changes.
func (s *ArtworkStore) CoverVariant(file *os.File, item Item, variant ArtworkVariant, sourceRevision string) (string, error) {
	width, height, quality := 320, 480, "5"
	if variant == ArtworkPlayer {
		width, height, quality = 720, 1080, "3"
	} else {
		variant = ArtworkCard
	}
	keyMaterial := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00cover-v2", item.LibraryID, item.ID, sourceRevision, variant)
	sum := sha256.Sum256([]byte(keyMaterial))
	filter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", width, height)
	return s.generator.Image(file, hex.EncodeToString(sum[:16]), "-i", "/proc/self/fd/3", "-map", "0:v:0", "-frames:v", "1", "-vf", filter, "-q:v", quality)
}
func (s *ArtworkStore) Close() { s.generator.Close() }
func (s *Store) OpenSidecar(id string) (*os.File, string, bool) {
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
	directory := filepath.Dir(filepath.Join(root, filepath.FromSlash(item.Path)))
	for pathWithin(root, directory) {
		for _, stem := range []string{"cover", "folder", "poster"} {
			for _, extension := range []string{".jpg", ".jpeg", ".png", ".webp"} {
				candidate := filepath.Join(directory, stem+extension)
				resolved, e := filepath.EvalSymlinks(candidate)
				if e != nil || !pathWithin(resolvedRoot, resolved) {
					continue
				}
				file, e := os.Open(resolved)
				if e != nil {
					continue
				}
				info, e := file.Stat()
				if e == nil && info.Mode().IsRegular() && info.Size() <= 32<<20 {
					return file, extension, true
				}
				file.Close()
			}
		}
		if directory == root {
			break
		}
		directory = filepath.Dir(directory)
	}
	return nil, "", false
}

// ArtworkSiblings returns a bounded set of tracks from the same album folder.
// Music collections commonly embed a cover in only some tracks.
func (s *Store) ArtworkSiblings(id string, limit int) []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.byID[id]
	if !ok || limit <= 0 {
		return nil
	}
	if limit > len(s.items) {
		limit = len(s.items)
	}
	directory := filepath.Dir(filepath.FromSlash(item.Path))
	withArtwork := make([]Item, 0, limit)
	unknown := make([]Item, 0, limit)
	for _, candidate := range s.items {
		if candidate.ID == id || filepath.Dir(filepath.FromSlash(candidate.Path)) != directory {
			continue
		}
		if candidate.HasArtwork {
			withArtwork = append(withArtwork, candidate)
			if len(withArtwork) == limit {
				break
			}
		} else if len(unknown) < limit {
			unknown = append(unknown, candidate)
		}
	}
	if len(withArtwork) >= limit {
		return cloneItems(withArtwork[:limit])
	}
	remaining := limit - len(withArtwork)
	if len(unknown) > remaining {
		unknown = unknown[:remaining]
	}
	return cloneItems(append(withArtwork, unknown...))
}
func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !(len(relative) > 3 && relative[:3] == ".."+string(filepath.Separator))
}
