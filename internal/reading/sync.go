package reading

import (
	"strings"
	"time"
)

type SyncCoordinator struct{ store *Store }

func NewSyncCoordinator(store *Store) *SyncCoordinator { return &SyncCoordinator{store: store} }
func (c *SyncCoordinator) FindDocument(account, document string) (Position, bool) {
	return c.store.FindKoSyncDocument(account, strings.ToLower(document))
}
func (c *SyncCoordinator) Link(account, libraryID, bookID, document string) (Position, error) {
	if account == "" || libraryID == "" || bookID == "" || strings.TrimSpace(document) == "" {
		return Position{}, ErrInvalid
	}
	return c.store.LinkKoSync(account, libraryID, bookID, strings.ToLower(document))
}
func (c *SyncCoordinator) Record(account, libraryID, bookID string, p SyncProgress, a *Anchor) (Position, error) {
	if account == "" || libraryID == "" || bookID == "" || strings.TrimSpace(p.Document) == "" || strings.TrimSpace(p.Progress) == "" || p.Timestamp <= 0 || !ValidFraction(p.Percentage) || !ValidAnchor(a) {
		return Position{}, ErrInvalid
	}
	return c.store.RecordKoSync(account, libraryID, bookID, p, a)
}
func (c *SyncCoordinator) Accept(account, libraryID, bookID string, timestamp int64, p Position) (Position, error) {
	if account == "" || libraryID == "" || bookID == "" || timestamp <= 0 || !ValidFraction(p.Fraction) || !ValidAnchor(p.WebAnchor) {
		return Position{}, ErrInvalid
	}
	return c.store.AcceptKoSync(account, libraryID, bookID, timestamp, p)
}
func (c *SyncCoordinator) MarkSeen(account, libraryID, bookID string, timestamp int64) (Position, error) {
	if account == "" || libraryID == "" || bookID == "" || timestamp <= 0 {
		return Position{}, ErrInvalid
	}
	return c.store.MarkKoSyncSeen(account, libraryID, bookID, timestamp)
}

func SyncCandidates(position Position, binary, filename string) []string {
	values := []string{position.KoSyncDocument, binary, filename}
	out := make([]string, 0, 3)
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
func SyncMatchKind(candidate, linked, filename string) string {
	candidate = strings.ToLower(candidate)
	if candidate == strings.ToLower(filename) {
		return "filename"
	}
	if candidate == strings.ToLower(linked) {
		return "linked"
	}
	return "binary"
}

// ShouldOfferCandidate reports whether another reader's position is worth offering as "sync to here".
//
// The two questions it asks come from different clocks, so each is asked of the clock that can answer it.
// "Is this newer than what we already have?" is an ordering question between two device stamps, and only
// the device's clock needs to agree with itself. "Is this recent?" cannot be answered by a device stamp at
// all: comparing it with our own time meant an e-reader a minute behind was never offered (its save looked
// old the moment it arrived) and one running ahead kept a week-old position looking fresh forever. When the
// server recorded the arrival, that is used; a stored position from before that was recorded falls back to
// the stamp, which is at least what it was judged by until now.
func ShouldOfferCandidate(local Position, remote SyncProgress, now time.Time, maxAge time.Duration) bool {
	if remote.Timestamp <= local.KoSyncSeenAt {
		return false
	}
	if maxAge <= 0 {
		return true
	}
	receivedAt := remote.SeenAt
	if receivedAt == 0 {
		// Nothing recorded when this arrived (a position stored before that existed, or a position read
		// from a foreign KoSync server). The stamp is all there is: it can still say the position is
		// older than the window, but a stamp from the future cannot be vouched for at all.
		if remote.Timestamp > now.Unix() {
			return false
		}
		receivedAt = remote.Timestamp
	}
	return now.Unix()-receivedAt < int64(maxAge.Seconds())
}

func DevicePosition(position Position) Position {
	if position.WebAnchor == nil || position.WebAnchor.Version >= 2 || len([]rune(position.WebAnchor.Text)) <= 80 {
		return position
	}
	anchor := *position.WebAnchor
	anchor.Text = strings.TrimSpace(string([]rune(anchor.Text)[:80]))
	if boundary := strings.LastIndexByte(anchor.Text, ' '); boundary >= 48 {
		anchor.Text = anchor.Text[:boundary]
	}
	position.WebAnchor = &anchor
	return position
}
