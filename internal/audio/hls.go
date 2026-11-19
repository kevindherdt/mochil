package audio

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"mochil/internal/jobs"
	"mochil/internal/transcode"
)

const audioHLSSegmentSeconds = 2

type HLSStore struct{ engine *transcode.Engine }

type HLSQuality string

const (
	HLSQualityHigh   HLSQuality = "high"
	HLSQualityMedium HLSQuality = "medium"
	HLSQualityLow    HLSQuality = "low"
)

func ParseHLSQuality(value string) (HLSQuality, bool) {
	switch HLSQuality(value) {
	case HLSQualityHigh, HLSQualityMedium, HLSQualityLow:
		return HLSQuality(value), true
	default:
		return HLSQualityHigh, value == ""
	}
}

func (q HLSQuality) bitrate() string {
	switch q {
	case HLSQualityLow:
		return "80k"
	case HLSQualityMedium:
		return "128k"
	default:
		return "192k"
	}
}

func NewHLSStore(root, command string, maxBytes int64, workers int) (*HLSStore, error) {
	engine, err := transcode.NewEngine(root, command, maxBytes, workers, audioHLSSegmentSeconds)
	if err != nil {
		return nil, err
	}
	return &HLSStore{engine: engine}, nil
}

func (s *HLSStore) request(item Item, quality HLSQuality) transcode.Request {
	if parsed, ok := ParseHLSQuality(string(quality)); ok {
		quality = parsed
	} else {
		quality = HLSQualityHigh
	}
	return transcode.Request{
		Source:  transcode.Source{LibraryID: item.LibraryID, ItemID: item.ID, Revision: item.Revision, Duration: item.Duration, Kind: "audio"},
		Variant: "aac-stereo-" + quality.bitrate() + "-v1",
		Arguments: func(start int, directory string) []string {
			args := []string{"-v", "error", "-nostdin", "-y"}
			if start > 0 {
				args = append(args, "-ss", strconv.Itoa(start*audioHLSSegmentSeconds))
			}
			return append(args, "-i", "/proc/self/fd/3", "-map", "0:a:0", "-vn", "-sn", "-c:a", "aac", "-ac", "2", "-b:a", quality.bitrate(), "-f", "hls", "-hls_time", strconv.Itoa(audioHLSSegmentSeconds), "-hls_list_size", "0", "-hls_segment_type", "fmp4", "-hls_fmp4_init_filename", "init.mp4", "-hls_segment_filename", filepath.Join(directory, "seg%05d.m4s"), "-start_number", strconv.Itoa(start), "-hls_flags", "independent_segments+temp_file", filepath.Join(directory, "_ffmpeg.m3u8"))
		},
	}
}

func (s *HLSStore) Key(item Item, quality HLSQuality) string {
	if s == nil || s.engine == nil {
		return ""
	}
	return s.engine.Key(s.request(item, quality))
}
func (s *HLSStore) Start(file *os.File, item Item, quality HLSQuality) (string, error) {
	if s == nil || s.engine == nil {
		closeAudioFile(file)
		return "", errors.New("audio transcoding is unavailable")
	}
	return s.engine.Start(file, s.request(item, quality))
}
func (s *HLSStore) EnsureSegment(file *os.File, item Item, quality HLSQuality, segment int) error {
	if s == nil || s.engine == nil {
		closeAudioFile(file)
		return errors.New("audio transcoding is unavailable")
	}
	return s.engine.Ensure(file, s.request(item, quality), segment)
}
func (s *HLSStore) Playlist(item Item) []byte {
	return transcode.Manifest(item.Duration, audioHLSSegmentSeconds)
}
func (s *HLSStore) Open(key, name string) (*os.File, string, error) {
	if s == nil || s.engine == nil {
		return nil, "", os.ErrNotExist
	}
	return s.engine.Open(key, name)
}
func (s *HLSStore) Jobs() []jobs.State {
	if s == nil || s.engine == nil {
		return nil
	}
	return s.engine.Jobs()
}
func (s *HLSStore) Close() {
	if s != nil && s.engine != nil {
		s.engine.Close()
	}
}

// RewriteHLSPlaylist points the manifest at the versioned base. The segments are
// served with a private browser cache, so a file replaced in place (same item ID,
// new content) has to get new URLs or the player mixes old segments with new
// ones; the video path already did this.
func RewriteHLSPlaylist(data []byte, base string, version ...string) []byte {
	value := ""
	if len(version) > 0 {
		value = version[0]
	}
	return transcode.RewriteManifestVersioned(data, base, value)
}
func closeAudioFile(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}
