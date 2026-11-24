package web

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"mochil/internal/enrichment"
)

// ratingHardcover plays Hardcover with every book on the shelf; it remembers the rating it was last given, and the
// rating writes are recorded.
func ratingHardcover(recorder *shelfRecorder) *enrichment.Hardcover {
	var mu sync.Mutex
	current := "null"
	return enrichment.NewHardcover("token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		mu.Lock()
		defer mu.Unlock()
		switch {
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ShelfRating"):
			response = `{"data":{"user_books":[{"id":11,"rating":` + current + `}]}}`
		case strings.Contains(body, "ChangeRating"):
			recorder.add(body)
			current = "null"
			if strings.Contains(body, `"rating":4.5`) {
				current = "4.5"
			}
			response = `{"data":{"update_user_book":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
}

func ratingServer(t *testing.T) (*multiServer, *shelfRecorder) {
	t.Helper()
	server := editorTestServer(t) // Dune (a1) and Dune Messiah (a2) in library "books"
	recorder := &shelfRecorder{}
	server.hardcover = ratingHardcover(recorder)
	if err := startSync(t, server, ""); err != nil {
		t.Fatal(err)
	}
	server.hardcoverMatches.put(hardcoverMatch{Library: "books", Book: "a1", HardcoverID: "5"}) // only Dune is matched
	return server, recorder
}

func TestRatingABookInTheEditorReachesItsHardcoverShelfEntry(t *testing.T) {
	server, recorder := ratingServer(t)
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"a1":{"rating":"4.5"}}}`); response.Code != http.StatusOK {
		t.Fatalf("edit: %d %s", response.Code, response.Body)
	}
	got := waitForShelf(t, recorder, 1)
	if len(got) != 1 || !strings.Contains(got[0], `"rating":4.5`) || !strings.Contains(got[0], `"id":11`) {
		t.Fatalf("Hardcover writes = %v, want one 4.5-star rating on shelf entry 11", got)
	}
}

func TestOnlyBooksMatchedWithHardcoverAreRated(t *testing.T) {
	server, recorder := ratingServer(t)
	editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"a2":{"rating":"5"},"a1":{"title":"Dune (renamed)"}}}`)
	time.Sleep(300 * time.Millisecond)
	if got := recorder.got(); len(got) != 0 {
		t.Fatalf("a book with no stored match, and a title edit, must send nothing; sent %v", got)
	}
}

func TestRatingsStayHomeWhenTheModuleIsOff(t *testing.T) {
	server, recorder := ratingServer(t)
	server.cfg.Modules = map[string]bool{"hardcover": true, "hardcover_ratings": false}
	editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"a1":{"rating":"4"}}}`)
	time.Sleep(300 * time.Millisecond)
	if got := recorder.got(); len(got) != 0 {
		t.Fatalf("module off but sent %v", got)
	}
}

func TestUndoingARatingTakesItOffHardcoverAgain(t *testing.T) {
	server, recorder := ratingServer(t)
	response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"a1":{"rating":"4.5"}}}`)
	token := strings.TrimSpace(strings.SplitN(strings.SplitN(response.Body.String(), `"undo":"`, 2)[1], `"`, 2)[0])
	waitForShelf(t, recorder, 1)
	if undone := editorCall(server, http.MethodPost, "/api/metadata/editor/undo", `{"token":"`+token+`"}`); undone.Code != http.StatusOK {
		t.Fatalf("undo: %d %s", undone.Code, undone.Body)
	}
	got := waitForShelf(t, recorder, 2)
	if len(got) != 2 || !strings.Contains(got[1], `"rating":null`) {
		t.Fatalf("Hardcover writes = %v, want the 4.5 and then a clear", got)
	}
}

func TestRatedBooksNamesEveryBookWhoseRatingAnEditTouched(t *testing.T) {
	apply := map[string]map[string]string{"a": {"rating": "3"}, "b": {"title": "x"}}
	resets := map[string][]string{"c": {"rating"}, "d": {"series"}}
	got := ratedBooks(apply, resets, []string{"e"})
	want := map[string]bool{"a": true, "c": true, "e": true}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, id := range got {
		if !want[id] {
			t.Fatalf("got %v, want a, c and e", got)
		}
	}
}
