package externallibrary

import (
	"errors"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMembershipAndProgressPersistWithStableRemoteIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "external-library.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return time.UnixMilli(1000) }
	title, err := store.Put("reader", Title{ProviderID: "opds", SourceID: "catalogue", RemoteTitleID: "urn:book:one", Name: "One", Authors: []string{"Writer"}, ExecutionLocation: ExecutionDevice})
	if err != nil || title.ID != StableID("opds", "catalogue", "urn:book:one") {
		t.Fatalf("put=%+v err=%v", title, err)
	}
	progress, err := store.PutProgress("reader", title.ID, Progress{RemoteChapterID: "/chapter/1", Locator: "page:4", Fraction: .5, DeviceID: "phone"}, 0)
	if err != nil || progress.Revision != 1 {
		t.Fatalf("progress=%+v err=%v", progress, err)
	}
	if _, err := store.PutProgress("reader", title.ID, Progress{RemoteChapterID: "/chapter/1", Fraction: .6}, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	reloaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.List("reader"); len(got) != 1 || got[0].Name != "One" {
		t.Fatalf("titles=%+v", got)
	}
	if got, ok := reloaded.Progress("reader", title.ID, "/chapter/1"); !ok || got.Locator != "page:4" {
		t.Fatalf("progress=%+v ok=%v", got, ok)
	}
	if len(reloaded.List("other")) != 0 {
		t.Fatal("account data leaked")
	}
}

func TestConcurrentProgressWritesRequireOneAuthoritativeRevision(t *testing.T) {
	store, _ := Open("")
	title, _ := store.Put("reader", Title{ProviderID: "opds", SourceID: "catalog", RemoteTitleID: "urn:one", Name: "One", ExecutionLocation: ExecutionDevice})
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, locator := range []string{"page:4", "page:9"} {
		workers.Add(1)
		go func(locator string) {
			defer workers.Done()
			<-start
			_, err := store.PutProgress("reader", title.ID, Progress{RemoteChapterID: "chapter", Locator: locator, Fraction: .5}, 0)
			results <- err
		}(locator)
	}
	close(start)
	workers.Wait()
	close(results)
	var succeeded, conflicted int
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrConflict):
			conflicted++
		default:
			t.Fatalf("unexpected write error: %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("success=%d conflicts=%d", succeeded, conflicted)
	}
	if value, ok := store.Progress("reader", title.ID, "chapter"); !ok || value.Revision != 1 || (value.Locator != "page:4" && value.Locator != "page:9") {
		t.Fatalf("authoritative progress=%+v exists=%v", value, ok)
	}
}

func TestStaleEquivalentReadiumProgressAcknowledgesAuthoritativeRevision(t *testing.T) {
	store, _ := Open("")
	title, _ := store.Put("reader", Title{ProviderID: "opds", SourceID: "catalog", RemoteTitleID: "urn:one", Name: "One", ExecutionLocation: ExecutionDevice})
	firstLocator := `{"href":"chapter-2.xhtml","locations":{"totalProgression":0.41},"text":{"before":"the words before ","highlight":"same sentence","after":" after the words"}}`
	current, err := store.PutProgress("reader", title.ID, Progress{RemoteChapterID: "publication", Locator: firstLocator, Fraction: .41, DeviceID: "tablet"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	secondLocator := `{"href":"chapter-2.xhtml","locations":{"totalProgression":0.47},"text":{"before":"the words before ","highlight":"same sentence","after":" after the words"}}`
	acknowledged, err := store.PutProgress("reader", title.ID, Progress{RemoteChapterID: "publication", Locator: secondLocator, Fraction: .47, DeviceID: "phone"}, 0)
	if err != nil || acknowledged != current {
		t.Fatalf("acknowledged=%+v current=%+v err=%v", acknowledged, current, err)
	}
	stored, _ := store.Progress("reader", title.ID, "publication")
	if stored != current {
		t.Fatalf("stale acknowledgement replaced authoritative progress: %+v", stored)
	}
}

func TestStaleAmbiguousOrDifferentReadiumProgressConflicts(t *testing.T) {
	store, _ := Open("")
	title, _ := store.Put("reader", Title{ProviderID: "opds", SourceID: "catalog", RemoteTitleID: "urn:one", Name: "One", ExecutionLocation: ExecutionDevice})
	currentLocator := `{"href":"chapter-2.xhtml","text":{"before":"before","highlight":"repeated","after":"after"}}`
	_, _ = store.PutProgress("reader", title.ID, Progress{RemoteChapterID: "publication", Locator: currentLocator, Fraction: .2}, 0)
	for name, locator := range map[string]string{
		"no context":        `{"href":"chapter-2.xhtml","text":{"highlight":"repeated"}}`,
		"different href":    `{"href":"chapter-3.xhtml","text":{"before":"before","highlight":"repeated","after":"after"}}`,
		"different context": `{"href":"chapter-2.xhtml","text":{"before":"elsewhere","highlight":"repeated","after":"after"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := store.PutProgress("reader", title.ID, Progress{RemoteChapterID: "publication", Locator: locator, Fraction: .8}, 0)
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("expected conflict, got %v", err)
			}
		})
	}
}

func TestMalformedProgressNeverReachesState(t *testing.T) {
	store, _ := Open("")
	title, _ := store.Put("reader", Title{ProviderID: "opds", SourceID: "catalog", RemoteTitleID: "urn:one", Name: "One", ExecutionLocation: ExecutionDevice})
	tests := map[string]Progress{
		"missing chapter":       {Fraction: .5},
		"oversized chapter":     {RemoteChapterID: strings.Repeat("c", 2049), Fraction: .5},
		"oversized locator":     {RemoteChapterID: "locator", Locator: strings.Repeat("x", 8193), Fraction: .5},
		"oversized device":      {RemoteChapterID: "device", DeviceID: strings.Repeat("d", 161), Fraction: .5},
		"oversized device name": {RemoteChapterID: "device-name", DeviceName: strings.Repeat("d", 129), Fraction: .5},
		"negative fraction":     {RemoteChapterID: "negative", Fraction: -0.1},
		"large fraction":        {RemoteChapterID: "large", Fraction: 1.1},
		"nan fraction":          {RemoteChapterID: "nan", Fraction: math.NaN()},
		"infinite fraction":     {RemoteChapterID: "infinite", Fraction: math.Inf(1)},
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := store.PutProgress("reader", title.ID, value, 0); err == nil {
				t.Fatal("malformed progress was accepted")
			}
		})
	}
	if got := store.ListProgress("reader", title.ID); len(got) != 0 {
		t.Fatalf("malformed entries reached state: %+v", got)
	}
}

func TestRemovingTitleAlsoRemovesProgress(t *testing.T) {
	store, _ := Open("")
	title, _ := store.Put("reader", Title{ProviderID: "opds", SourceID: "catalog", RemoteTitleID: "urn:one", Name: "One", ExecutionLocation: ExecutionDevice})
	_, _ = store.PutProgress("reader", title.ID, Progress{RemoteChapterID: "chapter", Fraction: 1}, 0)
	if err := store.Remove("reader", title.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Progress("reader", title.ID, "chapter"); ok {
		t.Fatal("orphaned progress")
	}
}

func TestSharedProviderReferenceSurvivesOneAccountRemoval(t *testing.T) {
	store, _ := Open("")
	value := Title{ProviderID: "remote", SourceID: "44", RemoteTitleID: "9", Name: "Example", ExecutionLocation: ExecutionServer}
	first, _ := store.Put("one", value)
	second, _ := store.Put("two", value)
	if _, ok := store.Get("one", first.ID); !ok {
		t.Fatal("expected account one's own copy before removal")
	}
	if _, ok := store.Get("two", second.ID); !ok {
		t.Fatal("expected account two's own copy before removal")
	}
	if err := store.Remove("one", first.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get("one", first.ID); ok {
		t.Fatal("account one's copy should be gone after removal")
	}
	if _, ok := store.Get("two", second.ID); !ok {
		t.Fatal("account two's own copy must survive account one's unrelated removal: each account's saved title is independent storage, never a shared record")
	}
}
