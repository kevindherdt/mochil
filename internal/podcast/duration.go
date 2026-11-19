package podcast

import (
	"math"
	"strconv"
	"strings"
)

// DurationSeconds reads feed durations expressed as seconds or colon-separated
// time. Invalid feed data means unknown duration, never a non-JSON number.
func DurationSeconds(value string) float64 {
	var total float64
	for _, part := range strings.Split(strings.TrimSpace(value), ":") {
		number, err := strconv.ParseFloat(part, 64)
		if err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
			return 0
		}
		total = total*60 + number
		if math.IsInf(total, 0) {
			return 0
		}
	}
	return total
}
