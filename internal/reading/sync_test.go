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

// A device an hour ahead of the server used to make a week-old position look fresh forever, because the
// age test subtracted the device's stamp from the server's clock.
func TestShouldOfferCandidateIgnoresADeviceClockThatIsAhead(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	local := Position{KoSyncSeenAt: now.Add(-30 * 24 * time.Hour).Unix()}
	future := SyncProgress{Timestamp: now.Add(time.Hour).Unix()}
	if ShouldOfferCandidate(local, future, now, 15*time.Minute) {
		t.Fatal("a stamp from the future was treated as a fresh position")
	}
}

// And a device a minute behind had every genuine save refused, because the save looked old on arrival.
func TestShouldOfferCandidateAcceptsASaveFromASlowClock(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	local := Position{KoSyncSeenAt: now.Add(-time.Hour).Unix()}
	slow := SyncProgress{Timestamp: now.Add(-90 * time.Second).Unix(), SeenAt: now.Unix()}
	if !ShouldOfferCandidate(local, slow, now, 15*time.Minute) {
		t.Fatal("a save that arrived just now was refused because the device's clock runs behind")
	}
}

// A position older than the one we already hold is not an offer, however recently it arrived.
func TestShouldOfferCandidateIgnoresAPositionWeAlreadyHave(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	local := Position{KoSyncSeenAt: now.Unix()}
	older := SyncProgress{Timestamp: now.Add(-time.Hour).Unix(), SeenAt: now.Unix()}
	if ShouldOfferCandidate(local, older, now, 15*time.Minute) {
		t.Fatal("an older position was offered")
	}
}
