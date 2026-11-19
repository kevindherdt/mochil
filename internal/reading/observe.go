package reading

// Event is something worth telling other parts of the server about, such as a shelf service.
type Event int

const (
	// Reading: progress was saved on a book that is not finished. It repeats with every save, so listeners
	// that only care about "this book is being read" should remember what they have already handled.
	Reading Event = iota + 1
	// Finished: a book that was not finished now is, whichever app or device did it.
	Finished
)

// Observe registers one function to hear about events. It runs while the store is locked after the change is
// saved, so it must only hand the event on (a queue, a channel) and return at once.
func (s *Store) Observe(fn func(account string, event Event, position Position)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observer = fn
}

func (s *Store) notifyLocked(account string, before, after Position) {
	if s.observer == nil {
		return
	}
	switch {
	case after.Finished && !before.Finished:
		s.observer(account, Finished, clonePosition(after))
	case !after.Finished && after.Fraction > 0:
		s.observer(account, Reading, clonePosition(after))
	}
}
