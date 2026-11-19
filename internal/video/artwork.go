package video

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"

	"mochil/internal/artwork"
)

// ArtworkStore generates bounded, disposable video posters. Provider metadata
// and catalogue grouping deliberately remain outside this cache.
type ArtworkStore struct {
	generator *artwork.Generator
}

func NewArtworkStore(root, command string, maxBytes int64, workers int) (*ArtworkStore, error) {
	generator, err := artwork.NewGenerator(root, command, maxBytes, workers)
	if err != nil {
		return nil, err
	}
	return &ArtworkStore{generator: generator}, nil
}

func (s *ArtworkStore) key(item Item) string {
	sum := sha256.Sum256([]byte(item.LibraryID + "\x00" + item.ID + "\x00" + item.Revision + "\x00poster-v2"))
	return hex.EncodeToString(sum[:16])
}

func (s *ArtworkStore) Poster(file *os.File, item Item) (string, error) {
	if file == nil {
		return "", ErrUnavailable
	}
	key := s.key(item)
	seek := posterSeekSeconds(item.Duration)
	return s.generator.Image(file, key, "-ss", strconv.FormatFloat(seek, 'f', 3, 64), "-i", "/proc/self/fd/3", "-frames:v", "1", "-vf", "thumbnail,scale=480:-2", "-q:v", "4")
}

func posterSeekSeconds(duration float64) float64 {
	seek := duration * .1
	if seek < 5 {
		return 5
	}
	if seek > 600 {
		return 600
	}
	return seek
}

func (s *ArtworkStore) Close() { s.generator.Close() }
