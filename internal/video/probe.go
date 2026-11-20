package video

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Prober interface {
	Probe(string) (ProbeResult, error)
}

type ProbeResult struct {
	Duration  float64
	Video     []VideoStream
	Audio     []AudioStream
	Subtitles []SubtitleStream
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
	return &FFProber{command: resolved, timeout: timeout}, nil
}

type limitedBuffer struct {
	bytes.Buffer
	remaining int64
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	if int64(len(data)) > b.remaining {
		return 0, errors.New("media probe output exceeded limit")
	}
	b.remaining -= int64(len(data))
	return b.Buffer.Write(data)
}

func (p *FFProber) Probe(path string) (ProbeResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	command := exec.CommandContext(ctx, p.command, "-v", "error", "-print_format", "json", "-show_format", "-show_streams", path)
	stdout, stderr := &limitedBuffer{remaining: 1 << 20}, &limitedBuffer{remaining: 64 << 10}
	command.Stdout, command.Stderr = stdout, stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return ProbeResult{}, ctx.Err()
		}
		return ProbeResult{}, errors.New(strings.TrimSpace(stderr.String()))
	}
	var payload struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			Index       int               `json:"index"`
			CodecType   string            `json:"codec_type"`
			Codec       string            `json:"codec_name"`
			Width       int               `json:"width"`
			Height      int               `json:"height"`
			Channels    int               `json:"channels"`
			Duration    string            `json:"duration"`
			Tags        map[string]string `json:"tags"`
			Disposition map[string]int    `json:"disposition"`
		} `json:"streams"`
	}
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(stdout.Bytes()), 1<<20))
	if err := decoder.Decode(&payload); err != nil {
		return ProbeResult{}, err
	}
	result := ProbeResult{}
	result.Duration, _ = strconv.ParseFloat(payload.Format.Duration, 64)
	if result.Duration <= 0 {
		// Some containers report no format duration while still reporting stream
		// lengths; a zero here becomes a one-segment, two-second playlist.
		for _, stream := range payload.Streams {
			if stream.CodecType != "video" && stream.CodecType != "audio" {
				continue
			}
			if value, err := strconv.ParseFloat(stream.Duration, 64); err == nil && value > result.Duration {
				result.Duration = value
			}
		}
	}
	for _, stream := range payload.Streams {
		language, title := stream.Tags["language"], stream.Tags["title"]
		switch stream.CodecType {
		case "video":
			result.Video = append(result.Video, VideoStream{Index: stream.Index, Codec: stream.Codec, Width: stream.Width, Height: stream.Height})
		case "audio":
			result.Audio = append(result.Audio, AudioStream{Index: stream.Index, Codec: stream.Codec, Language: language, Title: title, Channels: stream.Channels, Default: stream.Disposition["default"] != 0})
		case "subtitle":
			text := map[string]bool{"subrip": true, "srt": true, "ass": true, "ssa": true, "webvtt": true, "mov_text": true}[stream.Codec]
			result.Subtitles = append(result.Subtitles, SubtitleStream{Index: stream.Index, Codec: stream.Codec, Language: language, Title: title, Default: stream.Disposition["default"] != 0, Forced: stream.Disposition["forced"] != 0, Text: text})
		}
	}
	return result, nil
}
