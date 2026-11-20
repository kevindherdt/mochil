package audio

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Analysis struct {
	Revision string
	GainDB   float64
	Peak     float64
}

type Analyzer interface {
	Analyze(context.Context, *os.File) (Analysis, error)
}

type FFAnalyzer struct{ command string }

func NewFFAnalyzer(command string) (*FFAnalyzer, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return nil, err
	}
	return &FFAnalyzer{command: resolved}, nil
}

var (
	integratedLoudness = regexp.MustCompile(`(?m)^\s*I:\s*(-?[0-9]+(?:\.[0-9]+)?)\s+LUFS\s*$`)
	truePeak           = regexp.MustCompile(`(?m)^\s*Peak:\s*(-?[0-9]+(?:\.[0-9]+)?)\s+dBFS\s*$`)
)

func (a *FFAnalyzer) Analyze(ctx context.Context, file *os.File) (Analysis, error) {
	if file == nil {
		return Analysis{}, errors.New("audio source is unavailable")
	}
	defer file.Close()
	command := exec.CommandContext(ctx, a.command, "-nostdin", "-hide_banner", "-i", "/proc/self/fd/3", "-map", "0:a:0", "-filter_complex", "ebur128=peak=true", "-f", "null", "-")
	command.ExtraFiles = []*os.File{file}
	stderr := &tailBuffer{maximum: 256 << 10}
	command.Stdout = stderr
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return Analysis{}, ctx.Err()
		}
		return Analysis{}, errors.New(strings.TrimSpace(stderr.String()))
	}
	return parseAnalysis(stderr.String())
}

func parseAnalysis(output string) (Analysis, error) {
	loudness := integratedLoudness.FindAllStringSubmatch(output, -1)
	peaks := truePeak.FindAllStringSubmatch(output, -1)
	if len(loudness) == 0 || len(peaks) == 0 {
		return Analysis{}, errors.New("audio loudness summary is missing")
	}
	integrated, err1 := strconv.ParseFloat(loudness[len(loudness)-1][1], 64)
	peakDB, err2 := strconv.ParseFloat(peaks[len(peaks)-1][1], 64)
	if err1 != nil || err2 != nil || math.IsNaN(integrated) || math.IsInf(integrated, 0) || integrated > 0 || integrated < -100 || peakDB > 20 || peakDB < -100 {
		return Analysis{}, errors.New("audio loudness summary is invalid")
	}
	gain := -18 - integrated
	if gain < -60 {
		gain = -60
	}
	if gain > 30 {
		gain = 30
	}
	return Analysis{GainDB: gain, Peak: math.Pow(10, peakDB/20)}, nil
}

type tailBuffer struct {
	data    []byte
	maximum int
}

func (b *tailBuffer) Write(value []byte) (int, error) {
	written := len(value)
	b.data = append(b.data, value...)
	if len(b.data) > b.maximum {
		copy(b.data, b.data[len(b.data)-b.maximum:])
		b.data = b.data[:b.maximum]
	}
	return written, nil
}
func (b *tailBuffer) String() string { return string(b.data) }
