package ingest

import "testing"

func TestPreviewPublicationsIsFillOnlyAndPaginated(t *testing.T) {
	publications := []CurrentPublication{
		{ID: "keep", Path: "Wrong/Folder/Series - Vol 02 - File Title.epub", Title: "Embedded Title", Authors: []string{"Embedded Author"}, Series: "Embedded Series"},
		{ID: "fill", Path: "Ursula K. Le Guin/Earthsea/Earthsea - Vol 01 - A Wizard of Earthsea [isbn=9780547773742].epub"},
	}
	preview := PreviewPublications(publications, 0, 2)
	if preview.Inspected != 2 || preview.Matched != 2 || len(preview.Items) != 2 {
		t.Fatalf("unexpected preview summary: %#v", preview)
	}
	for _, item := range preview.Items {
		if item.ID == "keep" {
			for _, change := range item.Changes {
				if change.Field == "title" || change.Field == "authors" || change.Field == "series" {
					t.Fatalf("existing metadata would be replaced: %#v", item)
				}
			}
		}
	}
	page := PreviewPublications(publications, 1, 1)
	if len(page.Items) != 1 || page.Offset != 1 || page.Next != 0 {
		t.Fatalf("unexpected final page: %#v", page)
	}
}

func TestPreviewLimitIsBounded(t *testing.T) {
	items := make([]CurrentPublication, 140)
	for i := range items {
		items[i] = CurrentPublication{ID: string(rune(i + 1)), Path: "Author/Series/Series - Vol 01 - Title.epub"}
	}
	preview := PreviewPublications(items, 0, 1000)
	if preview.Limit != MaximumPreviewItems || len(preview.Items) != MaximumPreviewItems || preview.Next != MaximumPreviewItems {
		t.Fatalf("preview was not bounded: %#v", preview)
	}
}
