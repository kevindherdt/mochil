package web

import "testing"

func TestCursorStalenessOnlyAppliesToARequestThatContinues(t *testing.T) {
	cases := []struct {
		name        string
		kind        string
		anchorFound bool
		seen        uint64
		current     uint64
		stale       bool
	}{
		// The regression: a scan commits between the two version reads while a *first* page is being served. There is no
		// anchor to lose, so this must be answered with the page, not "reload the shelf".
		{"first page while the catalogue moves", "", false, 7, 8, false},
		{"first page, nothing moved", "", true, 8, 8, false},
		{"cursor page, generation moved", "title", true, 7, 8, true},
		{"cursor page, anchor gone", "title", false, 8, 8, true},
		{"cursor page, anchor and version agree", "title", true, 8, 8, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := cursorStale(test.kind, test.anchorFound, test.seen, test.current); got != test.stale {
				t.Fatalf("cursorStale(%q, %v, %d, %d) = %v, want %v", test.kind, test.anchorFound, test.seen, test.current, got, test.stale)
			}
		})
	}
}
