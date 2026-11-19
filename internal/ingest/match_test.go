package ingest

import "testing"

func TestRankStrongIdentifierWinsAndExplainsScore(t *testing.T) {
	index := 1.0
	subject := Identity{Title: "The Final Empire", Authors: []string{"Brandon Sanderson"}, Series: "Mistborn", SeriesIndex: &index, Identifiers: map[string]string{"isbn": "978-0-7653-1178-8"}}
	results := Rank(subject, []Identity{
		{ID: "title-only", Title: "The Final Empire", Authors: []string{"Someone Else"}},
		{ID: "isbn", Title: "Mistborn: The Final Empire", Authors: []string{"Brandon Sanderson"}, Identifiers: map[string]string{"isbn": "9780765311788"}},
	})
	if len(results) != 2 || results[0].Candidate.ID != "isbn" || results[0].Confidence != High || results[0].Score != 85 {
		t.Fatalf("unexpected ranking: %#v", results)
	}
	if len(results[0].Contributions) != 2 {
		t.Fatalf("score is not explainable: %#v", results[0])
	}
}

func TestRankRejectsConflictingStrongIdentifier(t *testing.T) {
	results := Rank(Identity{Title: "Same", Identifiers: map[string]string{"tmdbid": "123"}}, []Identity{{ID: "wrong", Title: "Same", Identifiers: map[string]string{"tmdbid": "456"}}})
	if len(results) != 1 || !results[0].Rejected || results[0].Score != 0 || results[0].Contributions[0].Points != -1000 {
		t.Fatalf("identifier conflict was not rejected: %#v", results)
	}
}

func TestRankIsDeterministicAndDoesNotTreatWeakTitleAsCertain(t *testing.T) {
	results := Rank(Identity{Title: "Rebuild World"}, []Identity{{ID: "b", Title: "Rebuild World"}, {ID: "a", Title: "Rebuild World"}})
	if len(results) != 2 || results[0].Candidate.ID != "a" || results[0].Confidence != Low || results[0].Score != 25 {
		t.Fatalf("unexpected weak ranking: %#v", results)
	}
}
