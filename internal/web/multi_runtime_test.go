package web

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestMultiRuntimeRejectsInvalidComposition(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	valid := Library{ID: "books", Name: "Books", Store: staticBackend{}}
	tests := []struct {
		name      string
		cfg       MultiConfig
		libraries []Library
		want      string
	}{
		{name: "partial credentials", cfg: MultiConfig{Username: "reader"}, libraries: []Library{valid}, want: "configured together"},
		{name: "missing identity", libraries: []Library{{Name: "Books", Store: staticBackend{}}}, want: "requires an ID, name, and store"},
		{name: "missing store", libraries: []Library{{ID: "books", Name: "Books"}}, want: "requires an ID, name, and store"},
		{name: "duplicate identity", libraries: []Library{valid, valid}, want: `duplicate library ID "books"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewMulti(test.cfg, test.libraries, logger)
			if err == nil || handler != nil {
				t.Fatalf("NewMulti() = (%T, %v), want a construction error", handler, err)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestMultiRuntimeAllowsFirstRunWithoutLibraries(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewMulti() first-run handler: %v", err)
	}
	if handler == nil {
		t.Fatal("NewMulti() first-run handler is nil")
	}
	if closer, ok := handler.(interface{ Close() }); ok {
		closer.Close()
	}
}
