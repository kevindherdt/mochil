package ingest

import "testing"

// ParseFloat accepts NaN and ±Inf, and neither can be written as JSON: one crafted
// EPUB used to make the import listing answer 200 with no valid body at all.
func TestNonFiniteSeriesIndicesAreDropped(t *testing.T) {
	for _, value := range []string{"NaN", "Inf", "+Inf", "-Inf", "infinity"} {
		if got := parseIndex(value); got != nil {
			t.Fatalf("parseIndex(%q) = %v, want nil", value, *got)
		}
	}
	if got := parseIndex("2.5"); got == nil || *got != 2.5 {
		t.Fatalf("parseIndex(2.5) = %v", got)
	}
}
