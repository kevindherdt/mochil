package video

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strconv"

	"mochil/internal/artwork"
)

const PreviewIntervalSeconds = 10

// PreviewStore owns disposable trick-play frames. It deliberately has a
// separate queue and cache from posters so scrubbing cannot delay catalogue
// artwork.
type PreviewStore struct {
	generator *artwork.Generator
}

func NewPreviewStore(root, command string, maxBytes int64, workers int) (*PreviewStore, error) {
	generator, err := artwork.NewGenerator(root, command, maxBytes, workers)
	if err != nil {
		return nil, err
	}
	return &PreviewStore{generator: generator}, nil
}

func previewPosition(item Item, requested int) (int, error) {
	if requested < 0 || item.Duration <= 0 {
		return 0, errors.New("preview position is unavailable")
	}
	maximum := int(item.Duration)
	if maximum < 1 {
		return 0, errors.New("preview position is unavailable")
	}
	if requested >= maximum {
		requested = maximum - 1
	}
	return requested / PreviewIntervalSeconds * PreviewIntervalSeconds, nil
}

func previewKey(item Item, position int) string {
	sum := sha256.Sum256([]byte(item.LibraryID + "\x00" + item.ID + "\x00" + item.Revision + "\x00preview-v1\x00" + strconv.Itoa(position)))
	return hex.EncodeToString(sum[:16])
}

func (s *PreviewStore) Frame(file *os.File, item Item, requested int) (string, int, error) {
	if file == nil || s == nil || s.generator == nil {
		if file != nil {
			_ = file.Close()
		}
		return "", 0, ErrUnavailable
	}
	position, err := previewPosition(item, requested)
	if err != nil {
		_ = file.Close()
		return "", 0, err
	}
	// Only the keyframe at the seek point is decoded: "thumbnail" buffered up to a
	// hundred frames to pick a representative one, which cost ~2.4s per frame here
	// against ~0.8s for a single keyframe, and these frames are fetched in bursts
	// while a timeline is scrubbed. Audio and subtitle streams are skipped too.
	// Near the end of a file no keyframe may be left after the seek, so the last
	// buckets keep the old full-decode path rather than returning nothing.
	arguments := make([]string, 0, 14)
	if int(item.Duration)-position > 2*PreviewIntervalSeconds {
		arguments = append(arguments, "-skip_frame", "nokey")
	}
	arguments = append(arguments,
		"-ss", strconv.Itoa(position), "-i", "/proc/self/fd/3",
		"-frames:v", "1", "-vf", "scale=320:-2", "-an", "-sn", "-q:v", "5")
	path, err := s.generator.Image(file, previewKey(item, position), arguments...)
	return path, position, err
}

func (s *PreviewStore) Close() {
	if s != nil && s.generator != nil {
		s.generator.Close()
	}
}
