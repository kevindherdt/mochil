package video

import "testing"

func TestInspectMovieQualityReportsUnmatchedTitles(t *testing.T) {
	items := []Item{{ID: "one", Path: "Arrival (2016).mkv", Title: "Arrival"}, {ID: "two", Path: "Matched.mp4", Title: "Matched"}}
	report := InspectQuality("movies", items, func(item Item) bool { return item.ID == "two" }, nil)
	if report.Items != 2 || report.Groups != 2 || report.Counts["unmatched_title"] != 1 || len(report.Issues) != 1 || report.Issues[0].ItemID != "one" {
		t.Fatalf("report=%+v", report)
	}
}

func TestInspectEpisodeQualityGroupsShowsAndFindsIdentityProblems(t *testing.T) {
	items := []Item{
		{ID: "one", Path: "Show/Season 01/Show.S01E01.mkv", Title: "Show S01E01", Show: "Show", Season: 1, Episode: 1},
		{ID: "copy", Path: "Show/Season 01/copy.S01E01.mkv", Title: "Copy", Show: "Show", Season: 1, Episode: 1},
		{ID: "unknown", Path: "misc.mkv", Title: "Misc"},
	}
	report := InspectQuality("tvshows", items, nil, func(show string) bool { return show == "Matched" })
	if report.Items != 3 || report.Groups != 1 || report.Counts["unmatched_show"] != 1 || report.Counts["duplicate_episode"] != 1 || report.Counts["unidentified_episode"] != 1 {
		t.Fatalf("report=%+v", report)
	}
}
