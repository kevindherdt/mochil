package vaultwriter

import (
	"bytes"
	"errors"
	"testing"
)

func TestReadPageContentPreservesBoundaryAndEmptyPages(t *testing.T) {
	for _, size := range []int{0, 1, MaximumPageBytes} {
		input := bytes.Repeat([]byte("x"), size)
		got, err := readPageContent(bytes.NewReader(input))
		if err != nil || !bytes.Equal(got, input) {
			t.Fatalf("size %d: length=%d error=%v", size, len(got), err)
		}
	}
}

type growingPageReader struct{ consumed int }

func (r *growingPageReader) Read(p []byte) (int, error) {
	clear(p)
	r.consumed += len(p)
	return len(p), nil
}

func TestReadPageContentBoundsGrowingFilesWithoutReturningTruncatedData(t *testing.T) {
	reader := &growingPageReader{}
	got, err := readPageContent(reader)
	if err == nil || got != nil || reader.consumed != MaximumPageBytes+1 {
		t.Fatalf("length=%d consumed=%d error=%v", len(got), reader.consumed, err)
	}
}

type failingPageReader struct{ err error }

func (r failingPageReader) Read([]byte) (int, error) { return 0, r.err }

func TestReadPageContentPreservesReadFailures(t *testing.T) {
	want := errors.New("read interrupted")
	got, err := readPageContent(failingPageReader{want})
	if got != nil || !errors.Is(err, want) {
		t.Fatalf("content=%v error=%v", got, err)
	}
}
