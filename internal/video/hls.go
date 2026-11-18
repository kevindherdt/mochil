package video

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mochil/internal/jobs"
	"mochil/internal/safely"
	"mochil/internal/transcode"
)

var hlsSegmentName = transcode.SegmentName

const hlsSegmentSeconds = 2

type HLSQuality string

const (
	HLSQualityMaximum HLSQuality = "maximum"
	HLSQualityHigh    HLSQuality = "high"
	HLSQualityMedium  HLSQuality = "medium"
	HLSQualityLow     HLSQuality = "low"
)

type hlsProfile struct {
	name         HLSQuality
	height       int
	videoBitrate string
	audioBitrate string
}

func ParseHLSQuality(value string) (HLSQuality, bool) {
	switch HLSQuality(value) {
	case HLSQualityMaximum, HLSQualityHigh, HLSQualityMedium, HLSQualityLow:
		return HLSQuality(value), true
	default:
		return HLSQualityHigh, value == ""
	}
}

func (s *HLSStore) profile(quality HLSQuality) hlsProfile {
	if parsed, ok := ParseHLSQuality(string(quality)); ok {
		quality = parsed
	} else {
		quality = HLSQualityHigh
	}
	height, bitrate, audioBitrate := s.maxHeight, "", "192k"
	switch quality {
	case HLSQualityHigh:
		height, bitrate = boundedHeight(s.maxHeight, 1080), "8M"
	case HLSQualityMedium:
		height, bitrate, audioBitrate = boundedHeight(s.maxHeight, 720), "4M", "128k"
	case HLSQualityLow:
		height, bitrate, audioBitrate = boundedHeight(s.maxHeight, 480), "1500k", "80k"
	}
	return hlsProfile{name: quality, height: height, videoBitrate: bitrate, audioBitrate: audioBitrate}
}

func boundedHeight(configured, limit int) int {
	if configured <= 0 || configured > limit {
		return limit
	}
	return configured
}

type HLSStore struct {
	root        string
	command     string
	maxBytes    int64
	maxHeight   int
	vaapiDevice string
	vaapi       bool
	pool        *jobs.Pool
	sequence    atomic.Uint64
	mu          sync.Mutex
	sessions    map[string]*hlsSession
	idleAfter   time.Duration
	readyAfter  time.Duration

	statsMu      sync.Mutex
	statsReady   bool
	statsTotal   int64
	statsEntries map[string]hlsCacheStat
}

type hlsSession struct {
	key        string
	directory  string
	restartMu  sync.Mutex
	mu         sync.Mutex
	process    *exec.Cmd
	jobKey     string
	lastUsed   time.Time
	failed     error
	failures   int
	retryAfter time.Time
	done       bool
	start      int

	notifyMu sync.Mutex
	notifyCh chan struct{}
}

// wake notifies every goroutine blocked in waitChannel that something about
// this session changed (a segment landed, or the job finished/failed), so
// Open() can react immediately instead of waiting for its next poll tick.
func (h *hlsSession) wake() {
	h.notifyMu.Lock()
	old := h.notifyCh
	h.notifyCh = nil
	h.notifyMu.Unlock()
	if old != nil {
		close(old)
	}
}

// waitChannel returns a channel that is closed the next time wake() runs.
func (h *hlsSession) waitChannel() chan struct{} {
	h.notifyMu.Lock()
	defer h.notifyMu.Unlock()
	if h.notifyCh == nil {
		h.notifyCh = make(chan struct{})
	}
	return h.notifyCh
}

// watchHLSSegments runs alongside an active ffmpeg encode. ffmpeg writes
// segment files directly to disk outside of our process, so we cannot be
// notified of a new segment the way we would for an in-process writer;
// instead this single goroutine polls tightly on behalf of every current
// and future waiter, and wakes them all the moment it observes a change.
// This replaces N concurrent Open() callers each sleeping on their own fixed
// interval with one lightweight watcher per active session.
func watchHLSSegments(stop <-chan struct{}, session *hlsSession) {
	// ffmpeg publishes each segment only when it is complete (temp_file + rename),
	// so a new name is the change worth waking for. Counting names keeps the tick
	// cheap: a feature-length title leaves thousands of segments in the directory,
	// and stat-ing every one of them ten times a second cost a full core per stream.
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	var lastCount int
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}
		entries, err := os.ReadDir(session.directory)
		if err != nil {
			continue
		}
		count := 0
		for _, entry := range entries {
			if hlsSegmentName.MatchString(entry.Name()) {
				count++
			}
		}
		if count != lastCount {
			lastCount = count
			session.wake()
		}
	}
}

func NewHLSStore(root, command string, maxBytes int64, workers, maxHeight int, vaapiDevice string) (*HLSStore, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return nil, err
	}
	if maxBytes <= 0 {
		maxBytes = 4 << 30
	}
	if workers < 1 || workers > 8 {
		workers = 2
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	if maxHeight < 0 || maxHeight > 4320 {
		maxHeight = 1080
	}
	store := &HLSStore{root: root, command: resolved, maxBytes: maxBytes, maxHeight: maxHeight, vaapiDevice: vaapiDevice, pool: jobs.New(workers, 32), sessions: map[string]*hlsSession{}, idleAfter: 75 * time.Second, readyAfter: 20 * time.Second}
	store.vaapi = store.detectVAAPI()
	return store, nil
}

func (s *HLSStore) Key(item Item, audioIndex int, quality HLSQuality) string {
	encoder := "software"
	if s.vaapi {
		encoder = "vaapi"
	}
	profile := s.profile(quality)
	return transcode.Key(transcode.Source{LibraryID: item.LibraryID, ItemID: item.ID, Revision: item.Revision, Duration: item.Duration, Kind: "video"}, strconv.Itoa(audioIndex)+"\x00"+string(profile.name)+"\x00"+strconv.Itoa(profile.height)+"\x00"+profile.videoBitrate+"\x00"+encoder+"\x00vod5-ts")
}

func (s *HLSStore) Start(file *os.File, item Item, audioIndex int, quality HLSQuality) (string, error) {
	return s.start(file, item, audioIndex, quality, 0, false)
}

func (s *HLSStore) start(file *os.File, item Item, audioIndex int, quality HLSQuality, startSegment int, force bool) (string, error) {
	if file == nil || audioIndex < 0 || !containsAudioStream(item.Audio, audioIndex) {
		if file != nil {
			_ = file.Close()
		}
		return "", errors.New("invalid HLS source")
	}
	profile := s.profile(quality)
	key := s.Key(item, audioIndex, profile.name)
	directory := filepath.Join(s.root, key)
	s.mu.Lock()
	session := s.sessions[key]
	if session == nil {
		session = &hlsSession{key: key, directory: directory, lastUsed: time.Now()}
		s.sessions[key] = session
	}
	s.mu.Unlock()

	session.restartMu.Lock()
	defer session.restartMu.Unlock()
	session.mu.Lock()
	session.lastUsed = time.Now()
	if !force && completePlaylist(filepath.Join(directory, "_ffmpeg.m3u8")) {
		session.done, session.failed, session.failures, session.retryAfter = true, nil, 0, time.Time{}
		session.mu.Unlock()
		_ = file.Close()
		return key, nil
	}
	if session.failed != nil && time.Now().Before(session.retryAfter) {
		retryAfter := session.retryAfter
		session.mu.Unlock()
		_ = file.Close()
		return "", fmt.Errorf("HLS conversion retry is unavailable until %s", retryAfter.UTC().Format(time.RFC3339))
	}
	if session.jobKey != "" && !force {
		session.mu.Unlock()
		_ = file.Close()
		return key, nil
	}
	if force && session.jobKey != "" {
		previousJob := session.jobKey
		session.mu.Unlock()
		s.pool.Cancel(previousJob)
		waitContext, cancel := context.WithTimeout(context.Background(), s.readyAfter)
		waitErr := s.pool.Wait(waitContext, previousJob)
		cancel()
		if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
			_ = file.Close()
			return "", fmt.Errorf("stop previous HLS conversion: %w", waitErr)
		}
		session.mu.Lock()
		if session.jobKey != "" {
			session.mu.Unlock()
			_ = file.Close()
			return "", errors.New("previous HLS conversion is still stopping")
		}
		if errors.Is(session.failed, context.Canceled) {
			session.failed, session.retryAfter = nil, time.Time{}
		}
	}
	// A server restart can leave a partial playlist and later segment files on
	// disk without a live encoder. Never mix those stale files into a fresh
	// conversion: a resumed player may otherwise accept an old segment and
	// buffer forever on a discontinuous timeline.
	if startSegment == 0 && !completePlaylist(filepath.Join(directory, "_ffmpeg.m3u8")) {
		if err := os.RemoveAll(directory); err != nil {
			session.mu.Unlock()
			_ = file.Close()
			return "", fmt.Errorf("clear incomplete HLS conversion: %w", err)
		}
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		session.mu.Unlock()
		_ = file.Close()
		return "", err
	}
	args := []string{"-v", "error", "-nostdin", "-y"}
	if s.vaapi {
		args = append(args, "-vaapi_device", s.vaapiDevice)
	}
	if startSegment > 0 {
		args = append(args, "-ss", strconv.Itoa(startSegment*hlsSegmentSeconds))
	}
	args = append(args, "-i", "/proc/self/fd/3",
		"-map", "0:v:0", "-map", "0:"+strconv.Itoa(audioIndex), "-sn",
	)
	// A seeked encoder starts its decoded timestamps at zero by default. The
	// synthetic VOD manifest still places segment N at N*hlsSegmentSeconds, so
	// preserve that absolute timeline or players seeking into a resumed video
	// will buffer forever while waiting for timestamps that never arrive.
	if startSegment > 0 {
		args = append(args, "-output_ts_offset", strconv.Itoa(startSegment*hlsSegmentSeconds))
	}
	if s.vaapi {
		filter := "format=nv12,hwupload"
		if profile.height > 0 {
			filter += ",scale_vaapi=w=" + strconv.Itoa(profile.height*16/9) + ":h=" + strconv.Itoa(profile.height) + ":force_original_aspect_ratio=decrease"
		}
		args = append(args, "-vf", filter, "-c:v", "h264_vaapi", "-qp", "23")
	} else {
		if profile.height > 0 {
			args = append(args, "-vf", "scale=-2:min("+strconv.Itoa(profile.height)+"\\,ih)")
		}
		args = append(args, "-c:v", "libx264", "-preset", "veryfast", "-crf", "22")
	}
	if profile.videoBitrate != "" {
		args = append(args, "-maxrate", profile.videoBitrate, "-bufsize", profile.videoBitrate)
	}
	args = append(args,
		"-c:a", "aac", "-ac", "2", "-b:a", profile.audioBitrate,
		"-force_key_frames", "expr:gte(t,n_forced*"+strconv.Itoa(hlsSegmentSeconds)+")",
		"-f", "hls", "-hls_time", strconv.Itoa(hlsSegmentSeconds), "-hls_list_size", "0", "-hls_segment_type", "mpegts",
		"-hls_segment_filename", filepath.Join(directory, "seg%05d.ts"),
		"-start_number", strconv.Itoa(startSegment),
		"-hls_flags", "independent_segments+temp_file", filepath.Join(directory, "_ffmpeg.m3u8"),
	)
	source := file.Name()
	_ = file.Close()
	jobKey := fmt.Sprintf("hls:%s:%d:%d", key, startSegment, s.sequence.Add(1))
	session.process, session.jobKey, session.failed, session.done, session.start = nil, jobKey, nil, false, startSegment
	session.mu.Unlock()
	if err := s.pool.SubmitWithCleanup(jobKey, func(ctx context.Context) error { return s.run(ctx, session, jobKey, source, args) }, func() {
		// Cancelled while still queued, so run() never starts: without this the
		// session keeps jobKey and every later start short-circuits on it.
		s.finish(session, jobKey, nil, context.Canceled)
		session.wake()
	}); err != nil {
		session.mu.Lock()
		if session.jobKey == jobKey {
			session.jobKey = ""
			session.failed = err
			session.retryAfter = time.Now().Add(time.Second)
		}
		session.mu.Unlock()
		return "", err
	}
	go safely.Run("video idle stop", func() { s.stopWhenIdle(session, jobKey) })
	return key, nil
}

// EnsureSegment redirects an active encoder to a distant uncached part of the
// movie. Stable segment numbers let every seek share the same persistent cache.
func (s *HLSStore) EnsureSegment(file *os.File, item Item, audioIndex int, quality HLSQuality, segment int) error {
	if segment < 0 || item.Duration > 0 && float64(segment*hlsSegmentSeconds) >= item.Duration+hlsSegmentSeconds {
		_ = file.Close()
		return errors.New("invalid HLS segment") // past the end: never start an encoder there
	}
	key := s.Key(item, audioIndex, quality)
	directory := filepath.Join(s.root, key)
	if info, err := os.Stat(filepath.Join(directory, fmt.Sprintf("seg%05d.ts", segment))); err == nil && info.Size() > 0 {
		_ = file.Close()
		return nil
	}
	s.mu.Lock()
	session := s.sessions[key]
	s.mu.Unlock()
	if session != nil {
		session.mu.Lock()
		running, start := session.jobKey != "", session.start
		session.mu.Unlock()
		if running {
			highest := highestContiguousSegment(directory, start)
			if segment >= start && segment <= highest+10 {
				_ = file.Close()
				return nil
			}
		}
	}
	_, err := s.start(file, item, audioIndex, quality, segment, true)
	return err
}

// Playlist advertises the complete timeline. Segment requests block briefly
// while the requested region is generated on demand.
func (s *HLSStore) Playlist(item Item) []byte {
	return transcode.TransportStreamManifest(item.Duration, hlsSegmentSeconds)
}

func (s *HLSStore) detectVAAPI() bool {
	if strings.TrimSpace(s.vaapiDevice) == "" {
		return false
	}
	if info, err := os.Stat(s.vaapiDevice); err != nil || info.Mode()&os.ModeDevice == 0 {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	// Some hardware encoders (including AMD Polaris) have a 128-pixel
	// minimum dimension. Keep the probe small, but representative enough that
	// a successful result means a real video can actually be encoded.
	command := exec.CommandContext(ctx, s.command, "-v", "error", "-vaapi_device", s.vaapiDevice, "-f", "lavfi", "-i", "color=c=black:s=256x256:r=1", "-vf", "format=nv12,hwupload", "-frames:v", "1", "-c:v", "h264_vaapi", "-f", "null", "-")
	return command.Run() == nil
}

func containsAudioStream(streams []AudioStream, index int) bool {
	for _, stream := range streams {
		if stream.Index == index {
			return true
		}
	}
	return false
}

func (s *HLSStore) run(ctx context.Context, session *hlsSession, jobKey, source string, args []string) error {
	file, err := os.Open(source)
	if err != nil {
		s.finish(session, jobKey, nil, err)
		session.wake()
		return err
	}
	defer file.Close()
	stop := make(chan struct{})
	go safely.Run("video segment watch", func() { watchHLSSegments(stop, session) })
	defer close(stop)
	command := exec.CommandContext(ctx, s.command, args...)
	command.ExtraFiles = []*os.File{file}
	// A real encoder can leave a grandchild process alive after Cancel kills
	// the direct child (a forking wrapper, or a hardware-accel helper that
	// does not die with its parent). Without WaitDelay, Run() would block
	// until that straggler exits on its own, holding the stop request (and
	// the goroutine) hostage well past when the caller asked to stop.
	command.WaitDelay = 5 * time.Second
	stderr := &limitedBuffer{remaining: 128 << 10}
	command.Stdout, command.Stderr = io.Discard, stderr
	session.mu.Lock()
	if session.jobKey != jobKey {
		session.mu.Unlock()
		return context.Canceled
	}
	session.process = command
	session.mu.Unlock()
	err = command.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	} else if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		err = errors.New(message)
	}
	s.finish(session, jobKey, command, err)
	session.wake()
	s.evict(session.key)
	return err
}

func (s *HLSStore) finish(session *hlsSession, jobKey string, command *exec.Cmd, err error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.jobKey != jobKey {
		return
	}
	session.process = nil
	session.jobKey = ""
	session.done = err == nil && completePlaylist(filepath.Join(session.directory, "_ffmpeg.m3u8"))
	session.failed = err
	if err == nil {
		session.failures, session.retryAfter = 0, time.Time{}
		return
	}
	session.failures++
	delay := time.Second * time.Duration(1<<min(session.failures, 5))
	session.retryAfter = time.Now().Add(delay)
}

func (s *HLSStore) stopWhenIdle(session *hlsSession, jobKey string) {
	idleStopHLSSession(&s.idleAfter, s.pool.Cancel, session, jobKey)
}

func (s *HLSStore) Jobs() []jobs.State { return s.pool.Snapshot() }

func (s *HLSStore) Close() { s.pool.Close() }

func (s *HLSStore) Open(key, name string) (*os.File, string, error) {
	if len(key) != 32 || (name != "index.m3u8" && name != "init.mp4" && !hlsSegmentName.MatchString(name)) {
		return nil, "", os.ErrNotExist
	}
	s.mu.Lock()
	session := s.sessions[key]
	if session == nil {
		directory := filepath.Join(s.root, key)
		if completePlaylist(filepath.Join(directory, "_ffmpeg.m3u8")) {
			session = &hlsSession{key: key, directory: directory, lastUsed: time.Now(), done: true}
			s.sessions[key] = session
		}
	}
	s.mu.Unlock()
	if session == nil {
		return nil, "", os.ErrNotExist
	}
	path := filepath.Join(session.directory, name)
	deadline := time.Now().Add(s.readyAfter)
	for {
		session.mu.Lock()
		session.lastUsed = time.Now()
		failed := session.failed
		session.mu.Unlock()
		file, err := os.Open(path)
		if err == nil {
			mediaType := "video/mp2t"
			if name == "index.m3u8" {
				mediaType = "application/vnd.apple.mpegurl"
			} else if name == "init.mp4" {
				mediaType = "video/mp4"
			}
			return file, mediaType, nil
		}
		if failed != nil {
			return nil, "", fmt.Errorf("HLS conversion failed: %w", failed)
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, "", errors.New("HLS segment is not ready")
		}
		// Wait to be signaled by the watcher started alongside the active
		// encode (or by finish() on completion/failure) instead of busy
		// polling on a fixed interval; the timer only guards the overall
		// deadline.
		wait := session.waitChannel()
		timer := time.NewTimer(remaining)
		select {
		case <-wait:
		case <-timer.C:
		}
		timer.Stop()
	}
}

func RewriteHLSPlaylist(data []byte, base string, version ...string) []byte {
	value := ""
	if len(version) > 0 {
		value = version[0]
	}
	return transcode.RewriteManifestVersioned(data, base, value)
}
