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
func ShouldOfferCandidate(local Position, remote SyncProgress, now time.Time, maxAge time.Duration) bool {
	return remote.Timestamp > local.KoSyncSeenAt && now.Unix()-remote.Timestamp < int64(maxAge.Seconds())
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
