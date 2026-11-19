package video

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Item struct {
	ID           string           `json:"id"`
	LibraryID    string           `json:"library_id"`
	Path         string           `json:"path"`
	Title        string           `json:"title"`
	Show         string           `json:"show,omitempty"`
	Season       int              `json:"season,omitempty"`
	Episode      int              `json:"episode,omitempty"`
	EpisodeTitle string           `json:"episode_title,omitempty"`
	Container    string           `json:"container"`
	MediaType    string           `json:"media_type"`
	DirectPlay   bool             `json:"direct_play"`
	Size         int64            `json:"size"`
	Modified     time.Time        `json:"modified"`
	Revision     string           `json:"revision"`
	Duration     float64          `json:"duration_seconds,omitempty"`
	Video        []VideoStream    `json:"video_streams,omitempty"`
	Audio        []AudioStream    `json:"audio_streams,omitempty"`
	Subtitles    []SubtitleStream `json:"subtitle_streams,omitempty"`
}

type VideoStream struct {
	Index  int    `json:"index"`
	Codec  string `json:"codec,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type AudioStream struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec,omitempty"`
	Language string `json:"language,omitempty"`
	Title    string `json:"title,omitempty"`
	Channels int    `json:"channels,omitempty"`
	Default  bool   `json:"default,omitempty"`
}

type SubtitleStream struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec,omitempty"`
	Language string `json:"language,omitempty"`
	Title    string `json:"title,omitempty"`
	Default  bool   `json:"default,omitempty"`
	Forced   bool   `json:"forced,omitempty"`
	Text     bool   `json:"text,omitempty"`
}

func cloneItem(item Item) Item {
	item.Video = append([]VideoStream(nil), item.Video...)
	item.Audio = append([]AudioStream(nil), item.Audio...)
	item.Subtitles = append([]SubtitleStream(nil), item.Subtitles...)
	return item
}

func cloneItems(items []Item) []Item {
	out := make([]Item, len(items))
	for index := range items {
		out[index] = cloneItem(items[index])
	}
	return out
}

type format struct {
	mediaType  string
	directPlay bool
}

var formats = map[string]format{
	".mp4": {"video/mp4", true}, ".m4v": {"video/mp4", true},
	".mov": {"video/quicktime", true}, ".webm": {"video/webm", true},
	".ogv": {"video/ogg", true}, ".mkv": {"video/x-matroska", false},
}

func SupportedExtensions() []string {
	extensions := make([]string, 0, len(formats))
	for extension := range formats {
		extensions = append(extensions, extension)
	}
	sort.Strings(extensions)
	return extensions
}

var videoTitleSeparators = strings.NewReplacer("_", " ", ".", " ")

func Identify(libraryID, relativePath string, size int64, modified time.Time) (Item, bool) {
	relativePath = filepath.ToSlash(filepath.Clean(relativePath))
	if size < 0 || relativePath == "." || relativePath == ".." || strings.HasPrefix(relativePath, "../") || filepath.IsAbs(relativePath) {
		return Item{}, false
	}
	extension := strings.ToLower(filepath.Ext(relativePath))
	format, supported := formats[extension]
	if !supported {
		return Item{}, false
	}
	identity := sha256.Sum256([]byte(libraryID + "\x00" + relativePath))
	revision := sha256.Sum256([]byte(relativePath + "\x00" + strconv.FormatInt(size, 10) + "\x00" + modified.UTC().Format(time.RFC3339Nano)))
	name := strings.TrimSuffix(filepath.Base(relativePath), filepath.Ext(relativePath))
	name = strings.Join(strings.Fields(videoTitleSeparators.Replace(name)), " ")
	if name == "" {
		name = "Untitled video"
	}
	return Item{
		ID: hex.EncodeToString(identity[:16]), LibraryID: libraryID, Path: relativePath,
		Title: name, Container: strings.TrimPrefix(extension, "."), MediaType: format.mediaType,
		DirectPlay: format.directPlay, Size: size, Modified: modified,
		Revision: hex.EncodeToString(revision[:12]),
	}, true
}
