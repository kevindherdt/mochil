package metadata

import (
	"strings"
	"testing"

	"mochil/internal/catalog"
)

func TestInspectExplainsMissingPDFAuthor(t *testing.T) {
	report := Inspect([]catalog.Book{{ID: "pdf", Title: "Scanned book", Authors: []string{"Unknown"}, Format: "pdf"}})
	if len(report.Issues) != 1 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	issue := report.Issues[0]
	if issue.Format != "pdf" || !strings.Contains(issue.Reason, "PDF") {
		t.Fatalf("issue = %#v", issue)
	}
}

func TestInspectFindsReviewableIssues(t *testing.T) {
	one := 1.0
	report := Inspect([]catalog.Book{
		{ID: "a", Title: "First", Authors: []string{"Unknown"}, Series: "Saga", SeriesIndex: &one},
		{ID: "b", Title: "Second", Authors: []string{"Writer"}, Series: "Saga", SeriesIndex: &one},
		{ID: "c", Title: "Third", Authors: []string{"Writer"}, Series: "Saga"},
	})
	if report.BooksInspected != 3 || report.IssueCount != 3 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if report.Counts["missing_author"] != 1 || report.Counts["duplicate_series_index"] != 2 || report.Counts["missing_series_index"] != 0 {
		t.Fatalf("unexpected counts: %#v", report.Counts)
	}
}

func TestInspectOnlyFlagsActionableMissingVolumeNumbers(t *testing.T) {
	report := Inspect([]catalog.Book{
		{ID: "complete", Title: "A Story - Complete", Authors: []string{"Writer"}, Series: "A Story"},
		{ID: "fanbook", Title: "A Story Fanbook 01", Authors: []string{"Writer"}, Series: "A Story"},
		{ID: "range", Title: "A Story - Volume 01-02", Authors: []string{"Writer"}, Series: "A Story"},
		{ID: "volume", Title: "A Story - Volume 03", Authors: []string{"Writer"}, Series: "A Story"},
	})
	if report.Counts["missing_series_index"] != 1 || len(report.Issues) != 1 || report.Issues[0].BookID != "volume" {
		t.Fatalf("non-actionable volume issues were retained: %#v", report.Issues)
	}
}

func TestInspectDoesNotFlagGoodStandaloneBook(t *testing.T) {
	report := Inspect([]catalog.Book{{ID: "a", Title: "Book", Authors: []string{"Writer"}}})
	if report.IssueCount != 0 {
		t.Fatalf("unexpected issues: %#v", report.Issues)
	}
}

func TestInspectIgnoresParallelEditionsWithSameExplicitVolume(t *testing.T) {
	one := 1.0
	report := Inspect([]catalog.Book{
		{ID: "a", Title: "Slayers vol.01: The Ruby Eye", Authors: []string{"Writer"}, Series: "Slayers", SeriesIndex: &one},
		{ID: "b", Title: "Slayers: Volume 01", Authors: []string{"Writer"}, Series: "Slayers", SeriesIndex: &one},
	})
	if report.Counts["duplicate_series_index"] != 0 {
		t.Fatalf("parallel editions were reported as a volume error: %#v", report.Issues)
	}
}

func TestInspectKeepsExplicitVolumeMismatch(t *testing.T) {
	ten := 10.0
	report := Inspect([]catalog.Book{
		{ID: "a", Title: "Series Vol. 09", Authors: []string{"Writer"}, Series: "Series", SeriesIndex: &ten},
		{ID: "b", Title: "Series Vol. 10", Authors: []string{"Writer"}, Series: "Series", SeriesIndex: &ten},
	})
	if report.Counts["duplicate_series_index"] != 2 {
		t.Fatalf("actionable mismatch was hidden: %#v", report.Issues)
	}
}

func TestInspectExcludesSpecialPublicationsFromVolumeCollision(t *testing.T) {
	one := 1.0
	report := Inspect([]catalog.Book{
		{ID: "main", Title: "Saga Volume 01", Authors: []string{"Writer"}, Series: "Saga", SeriesIndex: &one},
		{ID: "stories", Path: "Saga/!Side Stories/Saga Short Stories.epub", Title: "Saga Short Stories", Authors: []string{"Writer"}, Series: "Saga", SeriesIndex: &one},
		{ID: "fanbook", Title: "Saga Fanbook", Authors: []string{"Writer"}, Series: "Saga", SeriesIndex: &one},
	})
	if report.Counts["duplicate_series_index"] != 0 {
		t.Fatalf("special publications were treated as reading-order conflicts: %#v", report.Issues)
	}
}

func TestInspectDoesNotProposeNumericOrderForSpecialPublication(t *testing.T) {
	report := Inspect([]catalog.Book{{
		ID: "stories", Title: "Saga Side Stories Volume 1", Path: "Saga/Side Stories.epub",
		Authors: []string{"Writer"}, Series: "Saga", PublicationRole: catalog.PublicationSideStory,
	}})
	if report.Counts["missing_series_index"] != 0 {
		t.Fatalf("special publication received order proposal: %#v", report.Issues)
	}
}

func TestInspectReportsExactDuplicateEditionsSeparately(t *testing.T) {
	one := 1.0
	report := Inspect([]catalog.Book{
		{ID: "scan", Title: "The Tale", Authors: []string{"Writer"}, Series: "Saga", SeriesIndex: &one},
		{ID: "ocr", Title: "The Tale", Authors: []string{"Writer"}, Series: "Saga", SeriesIndex: &one},
	})
	if report.Counts["duplicate_edition"] != 2 || report.Counts["duplicate_series_index"] != 0 {
		t.Fatalf("duplicate editions were not classified separately: %#v", report.Issues)
	}
}

func TestTitleVolumeNumberRecognizesBookMarker(t *testing.T) {
	if number, ok := titleVolumeNumber("Demon Realm (Series Book 2)"); !ok || number != 2 {
		t.Fatalf("book marker = %v, %v", number, ok)
	}
}

func TestTitleVolumeNumberPrefersVolumeOverPart(t *testing.T) {
	if number, ok := titleVolumeNumber("Part 1 Daughter of a Soldier Volume 02"); !ok || number != 2 {
		t.Fatalf("preferred marker = %v, %v", number, ok)
	}
}
