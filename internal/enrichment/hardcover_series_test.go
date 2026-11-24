package enrichment

import (
	"encoding/json"
	"testing"
)

// Hardcover files one book under several series listings — the main one, a publication-order one, a comics line — and
// the row it stores first is not the one a person means. "The Witcher (Publication order)" is a real answer the API
// gives for The Last Wish, and taking it would rename every book in the series to a listing nobody browses.
func TestHardcoverSeriesTakesTheFeaturedListing(t *testing.T) {
	rows := func(raw string) []hardcoverSeriesRow {
		var decoded struct {
			Rows []hardcoverSeriesRow `json:"rows"`
		}
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Fatal(err)
		}
		return decoded.Rows
	}
	name, index := hardcoverSeries(rows(`{"rows":[{"position":1,"featured":false,"series":{"name":"The Witcher (Publication order)"}},{"position":0.5,"featured":true,"series":{"name":"The Witcher"}}]}`))
	if name != "The Witcher" || index == nil || *index != 0.5 {
		t.Fatalf("name=%q index=%v, want the featured listing", name, index)
	}
	if name, index := hardcoverSeries(rows(`{"rows":[{"position":3,"series":{"name":"Saga"}}]}`)); name != "Saga" || index == nil || *index != 3 {
		t.Fatalf("a book with no featured row keeps its only listing: %q %v", name, index)
	}
	// 0 is what Hardcover sends for "not placed", and writing it would make a book the zeroth of its series.
	if name, index := hardcoverSeries(rows(`{"rows":[{"position":0,"featured":true,"series":{"name":"Unplaced"}}]}`)); name != "Unplaced" || index != nil {
		t.Fatalf("a zero position must not become an index: %q %v", name, index)
	}
	if name, index := hardcoverSeries(rows(`{"rows":[{"series":{"name":""}},{"featured":true}]}`)); name != "" || index != nil {
		t.Fatalf("a nameless row is not a series: %q %v", name, index)
	}
}
