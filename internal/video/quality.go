package video

import (
	"fmt"
	"sort"
	"strings"
)

const qualitySampleLimit = 50

type QualityIssue struct {
	Code   string   `json:"code"`
	Title  string   `json:"title"`
	Detail string   `json:"detail"`
	ItemID string   `json:"item_id,omitempty"`
	Paths  []string `json:"paths,omitempty"`
}

type QualityReport struct {
	Items  int            `json:"items"`
	Groups int            `json:"groups"`
	Counts map[string]int `json:"counts"`
	Issues []QualityIssue `json:"issues"`
}

// InspectQuality reports catalogue identity problems without probing or
// decoding video files. Movies are reviewed individually; episodic shelves are
// reviewed by show and episode identity so one missing provider match does not
// produce a row for every episode.
func InspectQuality(kind string, items []Item, hasItemMetadata func(Item) bool, hasShowMetadata func(string) bool) QualityReport {
	report := QualityReport{Items: len(items), Counts: map[string]int{}, Issues: []QualityIssue{}}
	if kind == "movies" {
		report.Groups = len(items)
		for _, item := range items {
			if hasItemMetadata == nil || !hasItemMetadata(item) {
				report.add("unmatched_title", QualityIssue{Code: "unmatched_title", Title: item.Title, ItemID: item.ID, Detail: "No reviewed provider match is saved for this movie.", Paths: []string{item.Path}})
			}
		}
		report.sort()
		return report
	}

	shows := map[string]string{}
	episodes := map[string][]Item{}
	for _, item := range items {
		show := strings.TrimSpace(item.Show)
		if show == "" || item.Season < 0 || item.Episode <= 0 {
			report.add("unidentified_episode", QualityIssue{Code: "unidentified_episode", Title: item.Title, ItemID: item.ID, Detail: "The filename does not provide a complete show, season, and episode identity.", Paths: []string{item.Path}})
			continue
		}
		showKey := normalizedVideoValue(show)
		shows[showKey] = show
		key := fmt.Sprintf("%s\x00%d\x00%d", showKey, item.Season, item.Episode)
		episodes[key] = append(episodes[key], item)
	}
	report.Groups = len(shows)
	for _, show := range shows {
		if hasShowMetadata == nil || !hasShowMetadata(show) {
			report.add("unmatched_show", QualityIssue{Code: "unmatched_show", Title: show, Detail: "No reviewed provider match is saved for this show."})
		}
	}
	for _, duplicates := range episodes {
		if len(duplicates) < 2 {
			continue
		}
		paths := make([]string, 0, min(len(duplicates), 8))
		for _, item := range duplicates[:min(len(duplicates), 8)] {
			paths = append(paths, item.Path)
		}
		first := duplicates[0]
		report.add("duplicate_episode", QualityIssue{
			Code: "duplicate_episode", Title: fmt.Sprintf("%s · S%02dE%02d", first.Show, first.Season, first.Episode),
			ItemID: first.ID, Detail: "More than one file resolves to this episode identity.", Paths: paths,
		})
	}
	report.sort()
	return report
}

func (r *QualityReport) add(code string, issue QualityIssue) {
	previous := r.Counts[code]
	r.Counts[code] = previous + 1
	if previous < qualitySampleLimit {
		r.Issues = append(r.Issues, issue)
	}
}

func (r *QualityReport) sort() {
	sort.Slice(r.Issues, func(i, j int) bool {
		if r.Issues[i].Code != r.Issues[j].Code {
			return r.Issues[i].Code < r.Issues[j].Code
		}
		return normalizedVideoValue(r.Issues[i].Title) < normalizedVideoValue(r.Issues[j].Title)
	})
}

func normalizedVideoValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
