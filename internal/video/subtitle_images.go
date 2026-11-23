package video

// Image-based subtitle tracks (PGS/VobSub) are the normal case in a 1080p remux, and they cannot become WebVTT: the
// cues are bitmaps, not text. This extracts a PGS stream once into timestamped PNG cues and serves them, so the web
// player can draw a caption the same way a Blu-ray player does — an image over the picture — while still being able
// to toggle it, restyle nothing, and never re-encode the video.
//
// One ffmpeg pass per track. The trick is that ffmpeg decodes PGS to a frame per *display set*, including the empty
// set that hides a caption, so splitting the stream lets one branch measure each frame while the other writes it:
//
//	[0:s]split[a][b]
//	[a]alphaextract,signalstats,metadata=mode=print:file=…   → pts_time + YMAX per frame (YMAX==0 means "no caption")
//	[b]null                                                   → the actual RGBA frame, written as PNG
//
// Pairing those two lists gives each caption its lifetime: a frame with ink starts a cue, the next frame (which is
// either the hide marker or the next line) ends it. Nothing is cropped: a PGS frame is already positioned inside the
// video's own rectangle, so drawing it full-size over the letterboxed video keeps the studio's placement, and a
// mostly-transparent 1920x1080 PNG compresses to ~18 kB.
//
// Extraction of a two-hour film takes about a minute, which is longer than an HTTP request should wait, so the first
// request answers 202 while a single background pass runs per (item, track). Subsequent requests read the manifest.

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// maximumSubtitleCues stops a malformed or pathological track from filling the cache: a two-hour film has roughly
// 900–2000 display sets, so this is far above anything real.
const maximumSubtitleCues = 20000

// subtitleImageTimeout bounds one extraction pass. A 4K two-hour film on a slow disk is the worst case.
const subtitleImageTimeout = 20 * time.Minute

// SubtitleImageCue is one caption: when it appears, when it goes away, and the file that draws it.
type SubtitleImageCue struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	File  string  `json:"file"`
	// Where the caption sits inside the frame. Client-side sizing scales the bitmap about this box's bottom centre, so
	// the caption grows upward from where the studio put it instead of drifting. Pixel space, matching Width/Height.
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// SubtitleImages is the manifest the player fetches: the video's pixel size (so the overlay can scale the caption to
// the letterboxed picture) and every cue in order.
type SubtitleImages struct {
	Width  int                `json:"width"`
	Height int                `json:"height"`
	Cues   []SubtitleImageCue `json:"cues"`
}

// frameStat is one line of the metadata print: its time and whether it carried any ink.
type frameStat struct {
	Time float64
	Ink  bool
	// The caption's bounding box, from cropdetect on the alpha mask. Zero on a frame that draws nothing.
	X int
	Y int
	W int
	H int
}

// cueSpan is a half-open interval over the frame list: frame Start draws until frame End begins.
type cueSpan struct {
	Start int
	End   int
}

// pairCues turns per-frame ink flags into cue spans. Frames alternate between "caption shown" and "caption hidden"
// (the hidden one is what makes a cue end), but a caption that is replaced without a blank frame in between is two
// consecutive inked frames, so each inked frame closes the previous one.
func pairCues(frames []frameStat) []cueSpan {
	spans := make([]cueSpan, 0, len(frames)/2+1)
	open := -1
	for index, frame := range frames {
		if frame.Ink {
			if open >= 0 {
				spans = append(spans, cueSpan{Start: open, End: index})
			}
			open = index
			continue
		}
		if open >= 0 {
			spans = append(spans, cueSpan{Start: open, End: index})
			open = -1
		}
	}
	if open >= 0 {
		spans = append(spans, cueSpan{Start: open, End: -1})
	}
	return spans
}

// SubtitleImageStore extracts and serves image-based subtitle tracks, one directory per (item revision, track).
type SubtitleImageStore struct {
	root    string
	command string
	probe   string

	mu       sync.Mutex
	inflight map[string]bool
}

// NewSubtitleImageStore prepares the cache directory. `command` is the ffmpeg binary; `probe` the ffprobe next to it
// (empty means "ffprobe on PATH"), used only to read a stream's pixel size before the first PNG exists.
func NewSubtitleImageStore(root, command, probe string) (*SubtitleImageStore, error) {
	if root == "" {
		return nil, errors.New("subtitle image cache directory is required")
	}
	resolved, err := exec.LookPath(command)
	if err != nil {
		return nil, err
	}
	if probe == "" {
		probe = "ffprobe"
	}
	if resolvedProbe, probeErr := exec.LookPath(probe); probeErr == nil {
		probe = resolvedProbe
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &SubtitleImageStore{root: root, command: resolved, probe: probe, inflight: map[string]bool{}}, nil
}

func (s *SubtitleImageStore) key(item Item, streamIndex int) string {
	sum := sha1.Sum([]byte(item.LibraryID + "\x00" + item.ID + "\x00" + item.Revision + "\x00" + strconv.Itoa(streamIndex)))
	return hex.EncodeToString(sum[:])
}

func (s *SubtitleImageStore) directory(item Item, streamIndex int) string {
	return filepath.Join(s.root, s.key(item, streamIndex))
}

// Manifest returns the extracted cues for a track. When the track has not been extracted yet it starts a single
// background pass and reports ready=false, so the caller can answer 202 and let the player retry.
func (s *SubtitleImageStore) Manifest(ctx context.Context, item Item, reuse *os.File, streamIndex int,
	open func() (*os.File, error)) (*SubtitleImages, bool, error) {
	if streamIndex < 0 {
		return nil, false, errors.New("invalid subtitle track")
	}
	directory := s.directory(item, streamIndex)
	if images, err := readSubtitleImages(directory); err == nil {
		if manifestHasCrop(images) {
			return images, true, nil
		}
		// A manifest extracted before cues carried their box would place captions at native size forever.
		_ = os.RemoveAll(directory)
	} else if !os.IsNotExist(err) {
		// A half-written directory (an extraction killed mid-flight) is a cache miss, not a failure: start over.
		_ = os.RemoveAll(directory)
	}

	s.mu.Lock()
	if s.inflight[directory] {
		s.mu.Unlock()
		return nil, false, nil
	}
	s.inflight[directory] = true
	s.mu.Unlock()

	track := streamIndex
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.inflight, directory)
			s.mu.Unlock()
		}()
		// The request's own handle cannot outlive the request, so the background pass reopens the file itself.
		background, cancel := context.WithTimeout(context.Background(), subtitleImageTimeout)
		defer cancel()
		_ = s.extract(background, item, open, track)
	}()
	return nil, false, nil
}

// Open serves one cue image. Only names this store writes are accepted, so the handler cannot be talked into reading
// outside the cache directory.
func (s *SubtitleImageStore) Open(item Item, streamIndex int, name string) (*os.File, error) {
	if !strings.HasPrefix(name, "c") || !strings.HasSuffix(name, ".png") {
		return nil, os.ErrNotExist
	}
	if strings.ContainsAny(name, "/\\") {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(s.directory(item, streamIndex), name))
}

// boxAfter finds the bounding box belonging to the caption shown at `index`. cropdetect emits it on a later frame, so
// this looks forward a couple of frames; the `hide` frame that ends a cue carries the previous caption's box, which is
// exactly what is wanted here.
func boxAfter(frames []frameStat, index int) frameStat {
	for offset := 1; offset <= 3; offset++ {
		next := index + offset
		if next >= len(frames) {
			break
		}
		if frames[next].W > 0 && frames[next].H > 0 {
			return frames[next]
		}
	}
	return frames[index]
}

// manifestHasCrop reports whether every cue knows where it sits.
func manifestHasCrop(images *SubtitleImages) bool {
	for _, cue := range images.Cues {
		if cue.W == 0 || cue.H == 0 {
			return false
		}
	}
	return len(images.Cues) > 0
}

func readSubtitleImages(directory string) (*SubtitleImages, error) {
	data, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var images SubtitleImages
	if err := json.Unmarshal(data, &images); err != nil {
		return nil, err
	}
	return &images, nil
}

func (s *SubtitleImageStore) extract(ctx context.Context, item Item, open func() (*os.File, error), streamIndex int) error {
	input, err := open()
	if err != nil {
		return err
	}
	defer input.Close()

	final := s.directory(item, streamIndex)
	working := final + ".tmp"
	_ = os.RemoveAll(working)
	if err := os.MkdirAll(working, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(working)

	metadataPath := filepath.Join(working, "frames.txt")
	frames, err := s.decode(ctx, input, streamIndex, working, metadataPath)
	if err != nil {
		return err
	}
	spans := pairCues(frames)
	if len(spans) == 0 {
		return errors.New("subtitle track has no cues")
	}
	if len(spans) > maximumSubtitleCues {
		spans = spans[:maximumSubtitleCues]
	}

	images := SubtitleImages{Cues: make([]SubtitleImageCue, 0, len(spans))}
	for index, span := range spans {
		source := filepath.Join(working, fmt.Sprintf("%05d.png", span.Start+1))
		name := fmt.Sprintf("c%05d.png", index+1)
		if err := os.Rename(source, filepath.Join(working, name)); err != nil {
			return err
		}
		end := frames[span.Start].Time + 8
		if span.End >= 0 {
			end = frames[span.End].Time
		}
		// The frame that draws the caption is not always the frame that measures it: cropdetect reports a box one
		// frame later than the image it describes (it needs the whole mask before it can bound it), so a cue's box is
		// taken from the first frame that carries one rather than from the ink frame itself. Reading only its own
		// frame left every box at zero and the caption locked to native size.
		box := boxAfter(frames, span.Start)
		images.Cues = append(images.Cues, SubtitleImageCue{
			Start: frames[span.Start].Time, End: end, File: name,
			X: box.X, Y: box.Y, W: box.W, H: box.H,
		})
	}

	// Only the cue images survive; the hide markers and the frame list are working state.
	entries, err := os.ReadDir(working)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "c") || !strings.HasSuffix(entry.Name(), ".png") {
			_ = os.Remove(filepath.Join(working, entry.Name()))
		}
	}

	if images.Width == 0 || images.Height == 0 {
		images.Width, images.Height = s.probeSize(ctx, input, streamIndex)
	}
	if (images.Width == 0 || images.Height == 0) && len(images.Cues) > 0 {
		// ffprobe absent or unhelpful: the cue PNG carries the track's own size.
		images.Width, images.Height = pngSize(filepath.Join(working, images.Cues[0].File))
	}
	encoded, err := json.Marshal(images)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(working, "manifest.json"), encoded, 0o644); err != nil {
		return err
	}

	// Swap the finished directory into place. If another pass won the race, its directory is just as good.
	if err := os.Rename(working, final); err != nil {
		if _, readErr := readSubtitleImages(final); readErr == nil {
			return nil
		}
		return err
	}
	return nil
}

// decode runs the single ffmpeg pass: PNG per display set in `directory`, and one metadata block per frame in
// `metadataPath`. Frame N in one is frame N in the other — same graph, same frames — which is what makes pairing legal.
func (s *SubtitleImageStore) decode(ctx context.Context, input *os.File, streamIndex int, directory, metadataPath string) ([]frameStat, error) {
	// cropdetect measures the caption's own rectangle on the alpha mask, which is what lets the client scale a caption
	// without moving it; limit=1 keeps faint anti-aliased edge pixels from being trimmed.
	graph := fmt.Sprintf(
		"[0:%d]split[a][b];[a]alphaextract,cropdetect=limit=1:round=2,signalstats,metadata=mode=print:file=%s[m];[b]null[v]",
		streamIndex, metadataPath,
	)
	command := exec.CommandContext(ctx, s.command,
		"-v", "error", "-nostdin", "-copyts",
		"-i", "/proc/self/fd/3",
		"-filter_complex", graph,
		"-map", "[v]", "-c:v", "png", "-vsync", "0", filepath.Join(directory, "%05d.png"),
		"-map", "[m]", "-f", "null", "-",
	)
	command.ExtraFiles = []*os.File{input}
	stderr := &limitedBuffer{remaining: 64 << 10}
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, errors.New(message)
	}
	return readFrameStats(metadataPath)
}

// The cropdetect keys. `x1/x2/y1/y2` are the raw edges and `x/y/w/h` the rounded box; only the latter are used.
const (
	cropKeyX = "lavfi.cropdetect.x="
	cropKeyY = "lavfi.cropdetect.y="
	cropKeyW = "lavfi.cropdetect.w="
	cropKeyH = "lavfi.cropdetect.h="
)

func parseIntOr(line, prefix string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimPrefix(line, prefix))
	if err != nil {
		return fallback
	}
	return value
}

// readFrameStats parses the metadata print. Each block is "frame:N pts:… pts_time:T" followed by signalstats keys; only
// the time and YMAX matter — YMAX is the brightest pixel of the alpha mask, so zero means the frame draws nothing.
func readFrameStats(path string) ([]frameStat, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	frames := make([]frameStat, 0, 1024)
	var current frameStat
	pending := false
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "frame:"):
			if pending {
				frames = append(frames, current)
			}
			current = frameStat{}
			pending = true
			if _, value, found := strings.Cut(line, "pts_time:"); found {
				if seconds, parseErr := strconv.ParseFloat(strings.TrimSpace(value), 64); parseErr == nil {
					current.Time = seconds
				}
			}
		case strings.HasPrefix(line, "lavfi.signalstats.YMAX="):
			if value, parseErr := strconv.Atoi(strings.TrimPrefix(line, "lavfi.signalstats.YMAX=")); parseErr == nil && value > 0 {
				current.Ink = true
			}
		case strings.HasPrefix(line, cropKeyX):
			current.X = parseIntOr(line, cropKeyX, current.X)
		case strings.HasPrefix(line, cropKeyY):
			current.Y = parseIntOr(line, cropKeyY, current.Y)
		case strings.HasPrefix(line, cropKeyW):
			current.W = parseIntOr(line, cropKeyW, current.W)
		case strings.HasPrefix(line, cropKeyH):
			current.H = parseIntOr(line, cropKeyH, current.H)
		}
	}
	if pending {
		frames = append(frames, current)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(frames, func(a, b int) bool { return frames[a].Time < frames[b].Time })
	return frames, nil
}

// probeSize reads the track's pixel size, used when the manifest is built before any PNG is inspected.
func (s *SubtitleImageStore) probeSize(ctx context.Context, input *os.File, streamIndex int) (int, int) {
	command := exec.CommandContext(ctx, s.probe,
		"-v", "error", "-select_streams", strconv.Itoa(streamIndex),
		"-show_entries", "stream=width,height", "-of", "csv=p=0", "/proc/self/fd/3",
	)
	command.ExtraFiles = []*os.File{input}
	output, err := command.Output()
	if err != nil {
		return 0, 0
	}
	fields := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(fields) != 2 {
		return 0, 0
	}
	width, firstErr := strconv.Atoi(strings.TrimSpace(fields[0]))
	height, secondErr := strconv.Atoi(strings.TrimSpace(fields[1]))
	if firstErr != nil || secondErr != nil {
		return 0, 0
	}
	return width, height
}

// pngSize is a cheap size probe for a cue image (config only, no pixel decode). Kept next to the store because the
// manifest fallback needs it when ffprobe is unavailable.
func pngSize(path string) (int, int) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}
	return config.Width, config.Height
}
