package ingest

import (
	"sort"
	"strings"
	"unicode"
)

type Identity struct {
	ID          string
	Title       string
	Authors     []string
	Series      string
	SeriesIndex *float64
	Year        int
	Identifiers map[string]string
}

type Contribution struct {
	Field  string `json:"field"`
	Points int    `json:"points"`
	Reason string `json:"reason"`
}

type Match struct {
	Candidate     Identity       `json:"candidate"`
	Score         int            `json:"score"`
	Confidence    Confidence     `json:"confidence"`
	Rejected      bool           `json:"rejected,omitempty"`
	Contributions []Contribution `json:"contributions"`
}

// Rank returns candidates in a deterministic order. A conflicting value for
// the same strong identifier rejects a candidate; missing evidence never does.
func Rank(subject Identity, candidates []Identity) []Match {
	result := make([]Match, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, score(subject, candidate))
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Rejected != result[j].Rejected {
			return !result[i].Rejected
		}
		if result[i].Score != result[j].Score {
			return result[i].Score > result[j].Score
		}
		return result[i].Candidate.ID < result[j].Candidate.ID
	})
	return result
}

func score(left, right Identity) Match {
	m := Match{Candidate: right}
	for key, value := range left.Identifiers {
		other := strings.TrimSpace(right.Identifiers[key])
		if strings.TrimSpace(value) == "" || other == "" {
			continue
		}
		if normalize(value) != normalize(other) {
			m.Rejected = true
			m.Contributions = append(m.Contributions, Contribution{Field: "identifier." + key, Points: -1000, Reason: "conflicting strong identifier"})
			m.Confidence = Low
			return m
		}
		m.add("identifier."+key, 70, "exact strong identifier")
	}
	if equalText(left.Title, right.Title) {
		m.add("title", 25, "normalized title match")
	}
	if equalText(left.Series, right.Series) {
		m.add("series", 15, "normalized series match")
	}
	if left.SeriesIndex != nil && right.SeriesIndex != nil && *left.SeriesIndex == *right.SeriesIndex {
		m.add("series_index", 15, "exact reading-order match")
	}
	if authorOverlap(left.Authors, right.Authors) {
		m.add("authors", 15, "normalized contributor match")
	}
	if left.Year > 0 && right.Year > 0 {
		if left.Year == right.Year {
			m.add("year", 10, "publication year match")
		} else {
			m.add("year", -8, "publication years differ")
		}
	}
	switch {
	case m.Score >= 70:
		m.Confidence = High
	case m.Score >= 40:
		m.Confidence = Medium
	default:
		m.Confidence = Low
	}
	return m
}

func (m *Match) add(field string, points int, reason string) {
	m.Score += points
	m.Contributions = append(m.Contributions, Contribution{Field: field, Points: points, Reason: reason})
}

func equalText(a, b string) bool { return normalize(a) != "" && normalize(a) == normalize(b) }

func authorOverlap(a, b []string) bool {
	seen := make(map[string]struct{}, len(a))
	for _, value := range a {
		if value = normalize(value); value != "" {
			seen[value] = struct{}{}
		}
	}
	for _, value := range b {
		if _, ok := seen[normalize(value)]; ok {
			return true
		}
	}
	return false
}

func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}
