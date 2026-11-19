package reading

import "testing"

type seen struct {
	account string
	event   Event
	book    string
}

func observed(t *testing.T) (*Store, *[]seen) {
	t.Helper()
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	var events []seen
	store.Observe(func(account string, event Event, position Position) {
		events = append(events, seen{account, event, position.BookID})
	})
	return store, &events
}

func TestObserverHearsAboutReadingAndFinishingOnce(t *testing.T) {
	store, events := observed(t)
	put := func(fraction float64, base int64) {
		if _, err := store.Put("kevin", Position{LibraryID: "books", BookID: "dune", Fraction: fraction}, base); err != nil {
			t.Fatal(err)
		}
	}
	put(0.1, 0)
	put(0.4, 1)
	put(0.99, 2)
	put(0.995, 3)
	want := []seen{{"kevin", Reading, "dune"}, {"kevin", Reading, "dune"}, {"kevin", Finished, "dune"}}
	if len(*events) != len(want) {
		t.Fatalf("events = %v, want %v (nothing more once it is finished)", *events, want)
	}
	for i := range want {
		if (*events)[i] != want[i] {
			t.Fatalf("events = %v, want %v", *events, want)
		}
	}
}

func TestObserverHearsFromKOReaderAndTheFinishButton(t *testing.T) {
	store, events := observed(t)
	if _, err := store.RecordKoSync("kevin", "books", "a", SyncProgress{Percentage: .2, Timestamp: 10}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordKoSync("kevin", "books", "a", SyncProgress{Percentage: .99, Timestamp: 20}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetFinished("kevin", "books", "b", true); err != nil {
		t.Fatal(err)
	}
	if len(*events) != 3 || (*events)[0].event != Reading || (*events)[1].event != Finished || (*events)[2] != (seen{"kevin", Finished, "b"}) {
		t.Fatalf("events = %v", *events)
	}
}

func TestObserverStaysQuietWhenNothingChanged(t *testing.T) {
	store, events := observed(t)
	_, _ = store.Put("kevin", Position{LibraryID: "books", BookID: "x", Fraction: 0}, 0) // opened, not read yet: not "reading"
	_, _ = store.SetFinished("kevin", "books", "x", true)
	_, _ = store.SetFinished("kevin", "books", "x", true) // already finished
	if len(*events) != 1 || (*events)[0].event != Finished {
		t.Fatalf("events = %v", *events)
	}
}
