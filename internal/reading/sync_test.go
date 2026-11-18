package reading

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSyncCandidatePolicy(t *testing.T) {
	p := Position{KoSyncDocument: "LINK"}
	got := SyncCandidates(p, "binary", "link")
	if !reflect.DeepEqual(got, []string{"link", "binary"}) {
		t.Fatalf("candidates: %#v", got)
	}
	if SyncMatchKind("same", "same", "same") != "filename" {
		t.Fatal("filename compatibility priority changed")
	}
	remote := SyncProgress{Timestamp: 100}
	if !ShouldOfferCandidate(Position{KoSyncSeenAt: 99}, remote, time.Unix(110, 0), 15*time.Minute) {
		t.Fatal("new candidate not offered")
	}
	if ShouldOfferCandidate(Position{KoSyncSeenAt: 100}, remote, time.Unix(110, 0), 15*time.Minute) {
		t.Fatal("seen candidate offered")
	}
}
func TestDevicePositionBoundsLegacyAnchorWithoutMutatingStoreValue(t *testing.T) {
	text := strings.Repeat("word ", 30)
	original := Position{WebAnchor: &Anchor{Version: 1, Text: text}}
	projected := DevicePosition(original)
	if len([]rune(projected.WebAnchor.Text)) > 80 {
		t.Fatal("legacy anchor not bounded")
	}
	if original.WebAnchor.Text != text {
		t.Fatal("source position mutated")
	}
}
func TestSyncCoordinatorRejectsInvalidProtocolState(t *testing.T) {
	store, err := Open(t.TempDir() + "/reading.json")
	if err != nil {
		t.Fatal(err)
	}
	sync := NewSyncCoordinator(store)
	if _, err = sync.Record("reader", "books", "one", SyncProgress{Document: "doc", Progress: "x", Percentage: 2, Timestamp: 1}, nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wanted invalid, got %v", err)
	}
}
