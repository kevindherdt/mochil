package main

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/web"
)

func scanLibrary(logger *slog.Logger, library web.Library) bool {
	return scanLibraryContext(context.Background(), logger, library)
}

// scanLibraryContext reports whether the library is ready. A panic while reading
// one library is that library's failure; it must not stop the server booting.
func scanLibraryContext(ctx context.Context, logger *slog.Logger, library web.Library) (ready bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("library scan panicked", "library", library.ID, "panic", recovered, "stack", string(debug.Stack()))
			ready = false
		}
	}()
	if library.Video != nil {
		stats, err := library.Video.Scan()
		if err != nil {
			logger.Error("library scan failed", "library", library.ID, "error", err)
			return false
		}
		logger.Info("library ready", "library", library.ID, "videos", stats.Items, "reused", stats.Reused, "errors", stats.Errors)
		return true
	}
	if library.Audio != nil {
		stats, err := library.Audio.Scan()
		if err != nil {
			logger.Error("library scan failed", "library", library.ID, "error", err)
			return false
		}
		logger.Info("library ready", "library", library.ID, "audio", stats.Items, "reused", stats.Reused, "errors", stats.Errors)
		return true
	}
	if library.Documents != nil {
		stats, err := library.Documents.Scan()
		if err != nil {
			logger.Error("library scan failed", "library", library.ID, "error", err)
			return false
		}
		logger.Info("library ready", "library", library.ID, "documents", stats.Items, "reused", stats.Reused, "errors", stats.Errors)
		return true
	}
	var stats catalog.Stats
	var err error
	if scanner, ok := library.Store.(interface {
		ScanContext(context.Context) (catalog.Stats, error)
	}); ok {
		stats, err = scanner.ScanContext(ctx)
	} else {
		stats, err = library.Store.Scan()
	}
	if err != nil {
		logger.Error("library scan failed", "library", library.ID, "error", err)
		return false
	}
	logger.Info("library ready", "library", library.ID, "books", stats.Books, "read", stats.Read, "reused", stats.Reused, "errors", stats.Errors)
	if stats.FirstError != "" {
		logger.Warn("library scan had errors", "library", library.ID, "first_error", stats.FirstError)
	}
	return true
}

func rescanLoop(ctx context.Context, interval time.Duration, schedule func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			schedule()
		case <-ctx.Done():
			return
		}
	}
}
