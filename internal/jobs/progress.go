package jobs

import (
	"context"
	"unicode/utf8"
)

type progressReporter func(current, total int64, message string)
type progressContextKey struct{}

const (
	// maximumProgressMessageBytes and maximumStateErrorBytes are the byte limits
	// the persisted-state loader enforces. Truncation must land on a rune boundary:
	// a cut inside a multi-byte character re-encodes wider when the state file is
	// written, which makes the loader reject a file this package just produced.
	maximumProgressMessageBytes = 120
	maximumStateErrorBytes      = 4096
)

// Report updates bounded, public-safe progress for the running job associated
// with ctx. It returns false outside a Pool worker or after cancellation.
func Report(ctx context.Context, current, total int64, message string) bool {
	if ctx == nil || current < 0 || total < 0 || (total > 0 && current > total) {
		return false
	}
	reporter, ok := ctx.Value(progressContextKey{}).(progressReporter)
	if !ok {
		return false
	}
	message = truncateUTF8(message, maximumProgressMessageBytes)
	reporter(current, total, message)
	return true
}

// truncateUTF8 cuts text to at most limit bytes without splitting a rune.
func truncateUTF8(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit]
}
