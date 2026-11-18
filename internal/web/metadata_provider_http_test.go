package web

import "testing"

func TestFirstKnownAuthorSkipsEmptyAndUnknownValues(t *testing.T) {
	t.Parallel()

	if got := firstKnownAuthor([]string{" ", "UNKNOWN", "Ursula K. Le Guin", "Other"}); got != "Ursula K. Le Guin" {
		t.Fatalf("firstKnownAuthor() = %q, want first usable author", got)
	}
	if got := firstKnownAuthor([]string{"", " unknown "}); got != "" {
		t.Fatalf("firstKnownAuthor() = %q, want empty fallback", got)
	}
}
