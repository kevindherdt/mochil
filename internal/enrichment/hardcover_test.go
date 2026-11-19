package enrichment

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHardcoverSearchUsesBearerTokenAndPreservesSearchOrder(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.String() != hardcoverEndpoint || request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer private-token" {
			t.Fatalf("unexpected request: %s", request.URL)
		}
		body, _ := io.ReadAll(request.Body)
		var response string
		if strings.Contains(string(body), "SearchBooks") {
			if !strings.Contains(string(body), `Dune Frank Herbert`) {
				t.Fatalf("author was not included in search: %s", body)
			}
			response = `{"data":{"search":{"ids":[2,1]}}}`
		} else {
			response = `{"data":{"books":[{"id":1,"title":"Dune Messiah","release_year":1969,"slug":"dune-messiah","contributions":[{"author":{"name":"Frank Herbert"}}]},{"id":2,"title":"Dune","release_year":1965,"slug":"dune","description":"Desert politics.","cached_image":{"url":"https://assets.hardcover.app/dune.jpg"},"contributions":[{"author":{"name":"Frank Herbert"}}]}]}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	provider := NewHardcover("private-token", client)
	for range 2 {
		results, err := provider.Search(context.Background(), Query{Title: "Dune", Author: "Frank Herbert"})
		if err != nil || len(results) != 2 || results[0].Title != "Dune" || results[0].Year != 1965 || results[0].Authors[0] != "Frank Herbert" || results[0].CoverURL == "" {
			t.Fatalf("results=%#v err=%v", results, err)
		}
	}
	if calls != 2 {
		t.Fatalf("requests=%d, want two provider calls followed by cache hit", calls)
	}
}

func TestHardcoverRejectsMissingTokenAndGraphQLErrors(t *testing.T) {
	if _, err := NewHardcover("", nil).Search(context.Background(), Query{Title: "Book"}); err == nil {
		t.Fatal("missing token was accepted")
	}
	provider := NewHardcover("token", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"errors":[{"message":"no"}]}`))}, nil
	})})
	if _, err := provider.Search(context.Background(), Query{Title: "Book"}); err == nil {
		t.Fatal("GraphQL error was accepted")
	}
}

func TestHardcoverConnectionCheckUsesCurrentUser(t *testing.T) {
	provider := NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), "ConnectionStatus") || request.Header.Get("Authorization") != "Bearer private-token" {
			t.Fatalf("unexpected connection request: %s", body)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":{"me":[{"id":7}]}}`))}, nil
	})})
	if err := provider.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// hardcoverShelf plays Hardcover for the shelf tests: one book (id 5), and whatever is already on the user's shelf.
func hardcoverShelf(t *testing.T, shelf string, calls *[]string) *Hardcover {
	t.Helper()
	return NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "SearchBooks"):
			response = `{"data":{"search":{"ids":[5]}}}`
		case strings.Contains(body, "BookDetails"):
			response = `{"data":{"books":[{"id":5,"title":"Dune","slug":"dune","contributions":[{"author":{"name":"Frank Herbert"}}]}]}}`
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ExistingShelfEntry"):
			response = `{"data":{"user_books":` + shelf + `}}`
		case strings.Contains(body, "AddToShelf"):
			*calls = append(*calls, "insert "+body)
			response = `{"data":{"insert_user_book":{"error":null}}}`
		case strings.Contains(body, "ChangeShelfEntry"):
			*calls = append(*calls, "update "+body)
			response = `{"data":{"update_user_book":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
}

func TestHardcoverShelfAddsABookNotOnTheShelfYet(t *testing.T) {
	var calls []string
	_, err := hardcoverShelf(t, `[]`, &calls).SetStatus(context.Background(), "Dune", []string{"Frank Herbert"}, HardcoverReading)
	if err != nil || len(calls) != 1 || !strings.HasPrefix(calls[0], "insert") || !strings.Contains(calls[0], `"book":5`) || !strings.Contains(calls[0], `"status":2`) {
		t.Fatalf("err=%v calls=%v", err, calls)
	}
}

func TestHardcoverShelfMovesAnExistingEntryAndNeverBackwards(t *testing.T) {
	var calls []string
	reading := `[{"id":11,"status_id":2}]`
	if _, err := hardcoverShelf(t, reading, &calls).SetStatus(context.Background(), "Dune", []string{"Frank Herbert"}, HardcoverRead); err != nil || len(calls) != 1 || !strings.HasPrefix(calls[0], "update") || !strings.Contains(calls[0], `"id":11`) || !strings.Contains(calls[0], `"status":3`) {
		t.Fatalf("finishing a book being read: err=%v calls=%v", err, calls)
	}
	calls = nil
	if _, err := hardcoverShelf(t, `[{"id":11,"status_id":3}]`, &calls).SetStatus(context.Background(), "Dune", []string{"Frank Herbert"}, HardcoverReading); err != nil || len(calls) != 0 {
		t.Fatalf("a read book was moved back to reading: err=%v calls=%v", err, calls)
	}
	if _, err := hardcoverShelf(t, reading, &calls).SetStatus(context.Background(), "Dune", []string{"Frank Herbert"}, HardcoverReading); err != nil || len(calls) != 0 {
		t.Fatalf("an unchanged status was written again: err=%v calls=%v", err, calls)
	}
}

func TestHardcoverShelfSkipsABookItCannotMatchClearly(t *testing.T) {
	var calls []string
	provider := hardcoverShelf(t, `[]`, &calls)
	for _, book := range []struct {
		title   string
		authors []string
	}{{"Dune Messiah", []string{"Frank Herbert"}}, {"Dune", []string{"Someone Else"}}} {
		if _, err := provider.SetStatus(context.Background(), book.title, book.authors, HardcoverRead); !errors.Is(err, ErrHardcoverNoMatch) {
			t.Fatalf("%s by %v: err=%v, want no match", book.title, book.authors, err)
		}
	}
	if len(calls) != 0 {
		t.Fatalf("wrote to Hardcover for an unmatched book: %v", calls)
	}
	if _, err := provider.SetStatus(context.Background(), "dune:", nil, HardcoverRead); err != nil {
		t.Fatalf("a book with no author and a punctuation-only difference should match: %v", err)
	}
}

func TestHardcoverShelfFindsTheBookByTitleWhenTheAuthorSearchIsCrowdedOut(t *testing.T) {
	var calls []string
	provider := NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "SearchBooks") && strings.Contains(body, "Gibson"):
			response = `{"data":{"search":{"ids":[9]}}}` // with the author in the text, only a bundle comes back
		case strings.Contains(body, "SearchBooks"):
			response = `{"data":{"search":{"ids":[5]}}}`
		case strings.Contains(body, "BookDetails") && strings.Contains(body, "9"):
			response = `{"data":{"books":[{"id":9,"title":"Neuromancer Collection 4 Books Bundle","slug":"b","contributions":[{"author":{"name":"William Gibson"}}]}]}}`
		case strings.Contains(body, "BookDetails"):
			response = `{"data":{"books":[{"id":5,"title":"Neuromancer","slug":"n","contributions":[{"author":{"name":"William Gibson"}}]}]}}`
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ExistingShelfEntry"):
			response = `{"data":{"user_books":[]}}`
		case strings.Contains(body, "AddToShelf"):
			calls = append(calls, body)
			response = `{"data":{"insert_user_book":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
	if _, err := provider.SetStatus(context.Background(), "Neuromancer", []string{"William Gibson"}, HardcoverReading); err != nil || len(calls) != 1 || !strings.Contains(calls[0], `"book":5`) {
		t.Fatalf("err=%v calls=%v, want the real novel (5), not the bundle (9)", err, calls)
	}
}

func TestHardcoverShelfUsesAKnownIDWithoutSearching(t *testing.T) {
	var calls []string
	searched := false
	provider := NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "SearchBooks"):
			searched = true
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ExistingShelfEntry"):
			response = `{"data":{"user_books":[]}}`
		case strings.Contains(body, "AddToShelf"):
			calls = append(calls, body)
			response = `{"data":{"insert_user_book":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
	if _, err := provider.SetStatusByID(context.Background(), "1643933", HardcoverRead); err != nil || searched || len(calls) != 1 || !strings.Contains(calls[0], `"book":1643933`) {
		t.Fatalf("err=%v searched=%v calls=%v", err, searched, calls)
	}
	if _, err := provider.SetStatusByID(context.Background(), "", HardcoverRead); !errors.Is(err, ErrHardcoverNoMatch) {
		t.Fatalf("an empty id: err=%v", err)
	}
}

func TestHardcoverShelfReportsWhatItActuallyDid(t *testing.T) {
	var calls []string
	ctx := context.Background()
	if result, _ := hardcoverShelf(t, `[]`, &calls).SetStatus(ctx, "Dune", []string{"Frank Herbert"}, HardcoverReading); result.Result != ShelfAdded || result.Book.ProviderID != "5" || result.Book.Title != "Dune" {
		t.Fatalf("new entry: %+v", result)
	}
	if result, _ := hardcoverShelf(t, `[{"id":1,"status_id":2}]`, &calls).SetStatus(ctx, "Dune", []string{"Frank Herbert"}, HardcoverRead); result.Result != ShelfChanged {
		t.Fatalf("reading -> read: %+v", result)
	}
	if result, _ := hardcoverShelf(t, `[{"id":1,"status_id":2}]`, &calls).SetStatus(ctx, "Dune", []string{"Frank Herbert"}, HardcoverReading); result.Result != ShelfUnchanged {
		t.Fatalf("already reading: %+v", result)
	}
}

func TestVolumeNumbersCompareTheWayHardcoverWritesThem(t *testing.T) {
	for _, same := range [][2]string{
		{"Rebuild World - Volume 01.2", "Rebuild World: Volume 1 Part 2"},
		{"Rebuild World - Volume 03.1", "Rebuild World: Volume 3 Part 1"},
		{"Min-Maxing My TRPG Build in Another World - Volume 01", "Min-Maxing My TRPG Build in Another World: Volume 1"},
		{"Series, Vol. 2", "Series: Volume 2"},
	} {
		if !sameTitle(same[0], same[1]) {
			t.Errorf("%q and %q should be the same book", same[0], same[1])
		}
	}
	for _, different := range [][2]string{
		{"Rebuild World - Volume 01.2", "Rebuild World: Volume 1 Part 1"},
		{"Rebuild World - Volume 01.2", "Rebuild World: Volume 1"},
		{"Rebuild World - Volume 01", "Rebuild World: Volume 10"},
		{"Rebuild World - Volume 01.2", "Rebuild World: Volume 2 Part 2"},
	} {
		if sameTitle(different[0], different[1]) {
			t.Errorf("%q and %q are different books", different[0], different[1])
		}
	}
}

func TestHardcoverBookByIDDescribesTheBook(t *testing.T) {
	provider := NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(raw), "BookByID") || !strings.Contains(string(raw), `"id":1643933`) {
			t.Fatalf("unexpected request: %s", raw)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":{"books":[{"id":1643933,"title":"Rebuild World: Volume 1 Part 2","release_year":2021,"slug":"rebuild-world-volume-1-part-2","contributions":[{"author":{"name":"Nahuse"}}]}]}}`))}, nil
	})})
	book, err := provider.BookByID(context.Background(), "1643933")
	if err != nil || book.Title != "Rebuild World: Volume 1 Part 2" || book.Authors[0] != "Nahuse" || book.SourceURL != "https://hardcover.app/books/rebuild-world-volume-1-part-2" || book.ProviderID != "1643933" {
		t.Fatalf("book=%+v err=%v", book, err)
	}
	if _, err := provider.BookByID(context.Background(), "x"); !errors.Is(err, ErrHardcoverNoMatch) {
		t.Fatalf("a bad id: err=%v", err)
	}
}

func TestHardcoverMatchIsFlaggedWhenSeveralBooksFitEquallyWell(t *testing.T) {
	var calls []string
	provider := NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "SearchBooks"):
			response = `{"data":{"search":{"ids":[5,6]}}}`
		case strings.Contains(body, "BookDetails"):
			response = `{"data":{"books":[{"id":5,"title":"Dune","slug":"dune","contributions":[{"author":{"name":"Frank Herbert"}}]},{"id":6,"title":"Dune","slug":"dune-2","contributions":[{"author":{"name":"Frank Herbert"}}]}]}}`
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ExistingShelfEntry"):
			response = `{"data":{"user_books":[]}}`
		case strings.Contains(body, "AddToShelf"):
			calls = append(calls, body)
			response = `{"data":{"insert_user_book":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
	outcome, err := provider.SetStatus(context.Background(), "Dune", []string{"Frank Herbert"}, HardcoverReading)
	if err != nil || !outcome.Ambiguous || outcome.Book.ProviderID != "5" || len(calls) != 1 {
		t.Fatalf("outcome=%+v err=%v calls=%d, want the top-ranked book (5), flagged", outcome, err, len(calls))
	}
	var plain []string
	if outcome, err := hardcoverShelf(t, `[]`, &plain).SetStatus(context.Background(), "Dune", []string{"Frank Herbert"}, HardcoverReading); err != nil || outcome.Ambiguous {
		t.Fatalf("a single fitting book was flagged: %+v %v", outcome, err)
	}
}

func TestHardcoverShelfDoesNotReAddABookWhenAddingIsNotAllowed(t *testing.T) {
	var calls []string
	result, err := hardcoverShelf(t, `[]`, &calls).SetStatusAllowing(context.Background(), "Dune", []string{"Frank Herbert"}, HardcoverReading, false)
	if err != nil || result.Result != ShelfAbsent || len(calls) != 0 {
		t.Fatalf("result=%v err=%v calls=%v", result.Result, err, calls)
	}
	if result, err := hardcoverShelf(t, `[{"id":11,"status_id":2}]`, &calls).SetStatusAllowing(context.Background(), "Dune", []string{"Frank Herbert"}, HardcoverRead, false); err != nil || result.Result != ShelfChanged || len(calls) != 1 {
		t.Fatalf("a book still on the shelf is updated as usual: %v %v %v", result.Result, err, calls)
	}
}
