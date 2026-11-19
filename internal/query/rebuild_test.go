package query

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"mochil/internal/catalog"
)

// TestConcurrentReadersAfterAVersionBumpBuildTheIndexOnce is the point of the
// build lock: without it every request that saw the new catalogue version
// rebuilt the whole index and all but one result was thrown away.
func TestConcurrentReadersAfterAVersionBumpBuildTheIndexOnce(t *testing.T) {
	backend := &testBackend{}
	books := make([]catalog.Book, 500)
	for index := range books {
		books[index] = catalog.Book{ID: fmt.Sprintf("b%04d", index), Title: fmt.Sprintf("Book %04d", index)}
	}
	backend.replace(books)
	service := New([]Library{{ID: "books", Name: "Books", Store: backend}})
	if result := service.Books(BooksRequest{Limit: 10}); len(result.Items) != 10 {
		t.Fatalf("warm-up returned %d books", len(result.Items))
	}

	var builds atomic.Int32
	buildTestHook = func() { builds.Add(1) }
	defer func() { buildTestHook = nil }()

	backend.replace(append(books, catalog.Book{ID: "new", Title: "New Arrival"}))

	const readers = 24
	start := make(chan struct{})
	done := make(chan string, readers)
	var wait sync.WaitGroup
	for reader := 0; reader < readers; reader++ {
		wait.Add(1)
		go func(offset int) {
			defer wait.Done()
			<-start
			result := service.Books(BooksRequest{Limit: 5, Offset: offset})
			if len(result.Items) != 5 || result.Total != 501 {
				done <- fmt.Sprintf("offset %d got %d items, total %d", offset, len(result.Items), result.Total)
			}
			// A catalogue read must keep working after the rebuild, through the
			// cached index rather than a second build.
			if result := service.Libraries(); len(result) != 1 || result[0].Count != 501 {
				done <- fmt.Sprintf("libraries listed %+v", result)
			}
		}(reader % 4)
	}
	close(start)
	wait.Wait()
	close(done)
	for problem := range done {
		t.Fatal(problem)
	}
	if got := builds.Load(); got != 1 {
		t.Fatalf("index rebuilt %d times for one version bump, want 1", got)
	}
}

// TestSequentialReadsDoNotRebuildTheIndex proves the cheaper check that runs
// before the build lock: a matching version returns the published index.
func TestSequentialReadsDoNotRebuildTheIndex(t *testing.T) {
	backend := &testBackend{}
	backend.replace([]catalog.Book{{ID: "one", Title: "One"}})
	service := New([]Library{{ID: "books", Store: backend}})
	service.Books(BooksRequest{Limit: 10})

	var builds atomic.Int32
	buildTestHook = func() { builds.Add(1) }
	defer func() { buildTestHook = nil }()
	for range 5 {
		service.Books(BooksRequest{Limit: 10})
		service.Version("")
	}
	if got := builds.Load(); got != 0 {
		t.Fatalf("unchanged catalogue rebuilt the index %d times", got)
	}
}
