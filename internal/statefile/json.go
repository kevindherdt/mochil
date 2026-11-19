// Package statefile owns strict, bounded decoding of trusted-on-write but
// potentially damaged persistent JSON files.
package statefile

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

const MaximumCatalogueBytes int64 = 128 << 20

type boundedWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *boundedWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > w.remaining {
		return 0, errors.New("state file exceeds its byte limit")
	}
	written, err := w.writer.Write(data)
	w.remaining -= int64(written)
	return written, err
}

func EncodeJSON(writer io.Writer, maximumBytes int64, value any, indent bool) error {
	if maximumBytes <= 0 {
		return errors.New("state file byte limit must be positive")
	}
	encoder := json.NewEncoder(&boundedWriter{writer: writer, remaining: maximumBytes})
	if indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(value)
}

func DecodeJSON(path string, maximumBytes int64, target any) error {
	if maximumBytes <= 0 {
		return errors.New("state file byte limit must be positive")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maximumBytes {
		return errors.New("state file is invalid or too large")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maximumBytes+1))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("state file contains trailing data")
	}
	return nil
}
