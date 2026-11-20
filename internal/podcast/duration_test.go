package podcast

import "testing"

func TestDurationSeconds(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  float64
	}{
		{"125", 125}, {" 125.5 ", 125.5}, {"03:50", 230}, {"01:02:03", 3723},
		{"0", 0}, {"", 0}, {"unknown", 0}, {"1::2", 0},
		{"NaN", 0}, {"Inf", 0}, {"+Inf", 0}, {"-Inf", 0},
		{"1:NaN", 0}, {"1:Inf", 0}, {"-1", 0}, {"2:-10", 0},
		{"1e309", 0}, {"1e308:00", 0},
	} {
		t.Run(tc.value, func(t *testing.T) {
			if got := DurationSeconds(tc.value); got != tc.want {
				t.Fatalf("DurationSeconds(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}
