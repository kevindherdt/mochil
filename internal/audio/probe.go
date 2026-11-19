package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Metadata struct {
	Title, Artist, AlbumArtist, Album, Series, Genre, Narrator string
	Track, Disc, Year                                          int
	Bitrate                                                    int64
	HasArtwork                                                 bool
	Codec                                                      string
	SampleRate, BitDepth                                       int
	SeriesIndex                                                float64
	Duration                                                   float64
	ReplayTrack, ReplayAlbum, PeakTrack, PeakAlbum             float64
	Chapters                                                   []Chapter
}
type Prober interface {
	Probe(string) (Metadata, error)
}
type FFProber struct {
	command string
	timeout time.Duration
}

func NewFFProber(command string, timeout time.Duration) (*FFProber, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &FFProber{resolved, timeout}, nil
}

func (p *FFProber) Probe(path string) (Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	command := exec.CommandContext(ctx, p.command, "-v", "error", "-print_format", "json", "-show_entries", "format=duration,bit_rate:format_tags=title,artist,author,album_artist,albumartist,album,series,grouping,series-part,series_part,seriesindex,series_index,part,genre,date,year,originaldate,narrator,reader,read_by,narrated_by,performer,composer,track,disc,replaygain_track_gain,replaygain_album_gain,replaygain_track_peak,replaygain_album_peak,r128_track_gain,r128_album_gain:stream=codec_type,codec_name,sample_rate,bits_per_raw_sample:stream_disposition=attached_pic:chapter=start_time,end_time:chapter_tags=title", "--", path)
	var output, stderr bytes.Buffer
	command.Stdout = &output
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return Metadata{}, ctx.Err()
		}
		return Metadata{}, errors.New(strings.TrimSpace(stderr.String()))
	}
	if output.Len() > 1<<20 {
		return Metadata{}, errors.New("audio probe output exceeded limit")
	}
	return parseProbeOutput(output.Bytes())
}

func parseProbeOutput(output []byte) (Metadata, error) {
	var payload struct {
		Format struct {
			Duration string            `json:"duration"`
			Bitrate  string            `json:"bit_rate"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Chapters []struct {
			Start string            `json:"start_time"`
			End   string            `json:"end_time"`
			Tags  map[string]string `json:"tags"`
		} `json:"chapters"`
		Streams []struct {
			CodecType   string `json:"codec_type"`
			Codec       string `json:"codec_name"`
			SampleRate  string `json:"sample_rate"`
			BitDepth    string `json:"bits_per_raw_sample"`
			Duration    string `json:"duration"`
			Disposition struct {
				AttachedPicture int `json:"attached_pic"`
			} `json:"disposition"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return Metadata{}, err
	}
	tags := map[string]string{}
	for key, value := range payload.Format.Tags {
		tags[strings.ToLower(key)] = strings.TrimSpace(value)
	}
	duration, _ := strconv.ParseFloat(payload.Format.Duration, 64)
	if duration <= 0 {
		// ffprobe leaves format.duration out for unfinalised containers while still
		// reporting the stream length. Without the fallback the item was published
		// with a zero duration, and the playlist offered a single two-second segment.
		for _, stream := range payload.Streams {
			if stream.CodecType != "audio" {
				continue
			}
			if value, err := strconv.ParseFloat(stream.Duration, 64); err == nil && value > duration {
				duration = value
			}
		}
	}
	result := Metadata{Title: tags["title"], Artist: firstTag(tags, "artist", "author"), AlbumArtist: firstTag(tags, "album_artist", "albumartist"), Album: tags["album"], Series: firstTag(tags, "series", "grouping"), Genre: tags["genre"], Track: parseNumber(tags["track"]), Disc: parseNumber(tags["disc"]), Duration: duration}
	result.Year = parseAudioYear(firstTag(tags, "date", "year", "originaldate"))
	result.Bitrate = parseAudioBitrate(payload.Format.Bitrate)
	audioMetadataSet := false
	for _, stream := range payload.Streams {
		if stream.Disposition.AttachedPicture == 1 {
			result.HasArtwork = true
		}
		if stream.CodecType != "audio" || audioMetadataSet {
			continue
		}
		result.Codec = strings.ToLower(strings.TrimSpace(stream.Codec))
		result.SampleRate = parseBoundedInt(stream.SampleRate, 1_000, 768_000)
		result.BitDepth = parseBoundedInt(stream.BitDepth, 1, 64)
		audioMetadataSet = true
	}
	result.SeriesIndex, _ = parseSeriesIndex(firstTag(tags, "series-part", "series_part", "seriesindex", "series_index", "part"))
	var trackGainOK, albumGainOK bool
	result.ReplayTrack, trackGainOK = parseReplayGain(tags["replaygain_track_gain"])
	result.ReplayAlbum, albumGainOK = parseReplayGain(tags["replaygain_album_gain"])
	if !trackGainOK {
		result.ReplayTrack, _ = parseR128Gain(tags["r128_track_gain"])
	}
	if !albumGainOK {
		result.ReplayAlbum, _ = parseR128Gain(tags["r128_album_gain"])
	}
	result.PeakTrack, _ = parseReplayPeak(tags["replaygain_track_peak"])
	result.PeakAlbum, _ = parseReplayPeak(tags["replaygain_album_peak"])
	result.Narrator = firstTag(tags, "narrator", "reader", "read_by", "narrated_by", "performer", "composer")
	for index, chapter := range payload.Chapters {
		start, _ := strconv.ParseFloat(chapter.Start, 64)
		end, _ := strconv.ParseFloat(chapter.End, 64)
		title := strings.TrimSpace(chapter.Tags["title"])
		if title == "" {
			title = "Chapter " + strconv.Itoa(index+1)
		}
		if end > start {
			result.Chapters = append(result.Chapters, Chapter{Title: title, Start: start, End: end})
		}
	}
	return result, nil
}

func parseBoundedInt(value string, minimum, maximum int) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < minimum || number > maximum {
		return 0
	}
	return number
}

func parseAudioYear(value string) int {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return 0
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil || year < 1000 || year > 2100 {
		return 0
	}
	return year
}

func parseAudioBitrate(value string) int64 {
	bitrate, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || bitrate < 1_000 || bitrate > 1_000_000_000 {
		return 0
	}
	return bitrate
}

func parseSeriesIndex(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	for len(value) > 0 && (value[0] < '0' || value[0] > '9') && value[0] != '.' {
		value = value[1:]
	}
	end := 0
	for end < len(value) && ((value[end] >= '0' && value[end] <= '9') || value[end] == '.') {
		end++
	}
	if end == 0 {
		return 0, false
	}
	number, err := strconv.ParseFloat(strings.TrimRight(value[:end], "."), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 || number > 1_000_000 {
		return 0, false
	}
	return number, true
}

func firstTag(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		if tags[key] != "" {
			return tags[key]
		}
	}
	return ""
}

func parseReplayGain(value string) (float64, bool) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return 0, false
	}
	gain, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || math.IsNaN(gain) || math.IsInf(gain, 0) || gain < -60 || gain > 30 {
		return 0, false
	}
	return gain, true
}

func parseR128Gain(value string) (float64, bool) {
	raw, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, false
	}
	gain := float64(raw) / 256
	return gain, gain >= -60 && gain <= 30
}

func parseReplayPeak(value string) (float64, bool) {
	peak, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(peak) || math.IsInf(peak, 0) || peak <= 0 || peak > 100 {
		return 0, false
	}
	return peak, true
}
