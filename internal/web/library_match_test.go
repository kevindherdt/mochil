package web

import (
	"testing"

	"mochil/internal/enrichment"
)

// The match blurb is the largest field on an editor row and is read in one place only — filling a description that the
// book does not have. Sending it for a book that already has one costs about 700 bytes per row for nothing.
func TestMatchOverviewOnlyTravelsWhenItIsTheOnlyDescription(t *testing.T) {
	candidate := matchCandidate{Suggestion: enrichment.Suggestion{Title: "Dune", Overview: "A long blurb about sand."}, Score: 0.9}

	withDescription := candidate
	if got := withDescription.forList(&editorRow{Description: "What the book file says."}).Overview; got != "" {
		t.Errorf("a book with its own description still carried the match blurb: %q", got)
	}

	empty := candidate
	if got := empty.forList(&editorRow{}).Overview; got != "A long blurb about sand." {
		t.Errorf("a book with no description must keep the blurb, it is what staging a match fills in; got %q", got)
	}

	// A candidate that came back from the on-disk cache holds its blurb on this struct's own field (that is where JSON
	// writes it), with the embedded one empty. Both shapes must behave the same: dropped when the book has its own
	// description, kept when it does not — otherwise the books that most need a blurb, the ones with no description,
	// are exactly the ones that lose it.
	cached := matchCandidate{Score: 0.9}
	cached.Overview = "A long blurb about sand."
	if got := cached.forList(&editorRow{Description: "What the book file says."}).Overview; got != "" {
		t.Errorf("a cached candidate leaked its blurb onto a book that has a description: %q", got)
	}
	cachedAgain := matchCandidate{Score: 0.9}
	cachedAgain.Overview = "A long blurb about sand."
	if got := cachedAgain.forList(&editorRow{}).Overview; got != "A long blurb about sand." {
		t.Errorf("a cached candidate lost the blurb for a book with no description: %q", got)
	}
}
