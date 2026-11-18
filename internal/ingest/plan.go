package ingest

import (
	"sort"
	"strconv"
	"strings"
)

const MaximumPreviewItems = 100

type CurrentPublication struct {
	ID          string
	Path        string
	Title       string
	Authors     []string
	Series      string
	SeriesIndex *float64
	Identifier  string
}

type Change struct {
	Field      string     `json:"field"`
	Current    string     `json:"current,omitempty"`
	Proposed   string     `json:"proposed"`
	Source     string     `json:"source"`
	Confidence Confidence `json:"confidence"`
}

type PreviewItem struct {
	ID       string     `json:"id"`
	Path     string     `json:"path"`
	Title    string     `json:"title"`
	Changes  []Change   `json:"changes"`
	Evidence []Evidence `json:"evidence"`
}

type Preview struct {
	Inspected int           `json:"inspected"`
	Matched   int           `json:"matched"`
	Offset    int           `json:"offset"`
	Limit     int           `json:"limit"`
	Next      int           `json:"next_offset,omitempty"`
	Items     []PreviewItem `json:"items"`
}

type PublicationPreview struct {
	Inspected int
	Items     []PreviewItem
}

// PreviewPublications proposes fill-only changes. Existing non-empty metadata
// is evidence owned by the scanner and is never proposed for replacement.
func PreviewPublications(publications []CurrentPublication, offset, limit int) Preview {
	return BuildPublicationPreview(publications).Page(offset, limit)
}

// BuildPublicationPreview performs inference once. Servers can retain this
// immutable result for the lifetime of a catalogue generation.
func BuildPublicationPreview(publications []CurrentPublication) PublicationPreview {
	items := make([]PreviewItem, 0)
	for _, current := range publications {
		proposal := InferPath(Publication, current.Path)
		changes := publicationChanges(current, proposal)
		if len(changes) == 0 {
			continue
		}
		items = append(items, PreviewItem{ID: current.ID, Path: current.Path, Title: current.Title, Changes: changes, Evidence: proposal.Evidence})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Path != items[j].Path {
			return items[i].Path < items[j].Path
		}
		return items[i].ID < items[j].ID
	})
	return PublicationPreview{Inspected: len(publications), Items: items}
}

func (set PublicationPreview) Page(offset, limit int) Preview {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > MaximumPreviewItems {
		limit = MaximumPreviewItems
	}
	preview := Preview{Inspected: set.Inspected, Matched: len(set.Items), Offset: offset, Limit: limit, Items: []PreviewItem{}}
	if offset >= len(set.Items) {
		return preview
	}
	end := offset + limit
	if end > len(set.Items) {
		end = len(set.Items)
	}
	preview.Items = append(preview.Items, set.Items[offset:end]...)
	if end < len(set.Items) {
		preview.Next = end
	}
	return preview
}

func publicationChanges(current CurrentPublication, proposal Proposal) []Change {
	changes := make([]Change, 0, 5)
	add := func(field, currentValue, proposed string, confidence Confidence) {
		if strings.TrimSpace(currentValue) != "" || strings.TrimSpace(proposed) == "" {
			return
		}
		source := evidenceSource(proposal.Evidence, field)
		changes = append(changes, Change{Field: field, Current: currentValue, Proposed: proposed, Source: source, Confidence: confidence})
	}
	add("title", current.Title, proposal.Title, Medium)
	currentAuthors := strings.Join(current.Authors, " & ")
	if allUnknown(current.Authors) {
		currentAuthors = ""
	}
	add("authors", currentAuthors, strings.Join(proposal.Authors, " & "), Medium)
	add("series", current.Series, proposal.Series, Medium)
	if current.SeriesIndex == nil && proposal.SeriesIndex != nil {
		add("series_index", "", strconv.FormatFloat(*proposal.SeriesIndex, 'f', -1, 64), High)
	}
	if strings.TrimSpace(current.Identifier) == "" {
		for _, key := range []string{"isbn", "tmdbid", "tvdbid", "imdbid"} {
			if value := proposal.Identifiers[key]; value != "" {
				add("identifier", "", value, High)
				break
			}
		}
	}
	return changes
}

func allUnknown(values []string) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized != "" && normalized != "unknown" && normalized != "unknown author" {
			return false
		}
	}
	return true
}

func evidenceSource(evidence []Evidence, field string) string {
	for _, item := range evidence {
		if item.Field == field || (field == "identifier" && strings.HasPrefix(item.Field, "identifier.")) {
			return item.Source
		}
	}
	return "structured path"
}
