package video

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const maximumSubtitleBytes = 16 << 20

type SubtitleExtractor interface {
	Extract(context.Context, *os.File, int) ([]byte, error)
}

type FFmpegSubtitleExtractor struct {
	command string
	timeout time.Duration
}

func NewFFmpegSubtitleExtractor(command string, timeout time.Duration) (*FFmpegSubtitleExtractor, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	return &FFmpegSubtitleExtractor{command: resolved, timeout: timeout}, nil
}

type boundedWriter struct {
	bytes.Buffer
	remaining int
}

func (w *boundedWriter) Write(data []byte) (int, error) {
	if len(data) > w.remaining {
		return 0, errors.New("subtitle output exceeded limit")
	}
	w.remaining -= len(data)
	return w.Buffer.Write(data)
}

func (e *FFmpegSubtitleExtractor) Extract(parent context.Context, input *os.File, streamIndex int) ([]byte, error) {
	if input == nil || streamIndex < 0 {
		return nil, errors.New("invalid subtitle input")
	}
	ctx, cancel := context.WithTimeout(parent, e.timeout)
	defer cancel()
	output := &boundedWriter{remaining: maximumSubtitleBytes}
	stderr := &limitedBuffer{remaining: 64 << 10}
	command := exec.CommandContext(ctx, e.command, "-v", "error", "-nostdin", "-i", "/proc/self/fd/3", "-map", "0:"+strconv.Itoa(streamIndex), "-f", "webvtt", "pipe:1")
	command.ExtraFiles = []*os.File{input}
	command.Stdout, command.Stderr = output, stderr
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
	return append([]byte(nil), output.Bytes()...), nil
}
