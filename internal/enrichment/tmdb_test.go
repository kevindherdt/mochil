package enrichment

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTMDBSearchKeepsTokenServerSideAndCaches(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.Host != "api.themoviedb.org" || request.URL.Path != "/3/search/movie" || request.URL.Query().Get("query") != "Arrival" {
			t.Fatalf("unexpected request: %s", request.URL)
		}
		if request.Header.Get("Authorization") != "Bearer private-token" || strings.Contains(request.URL.String(), "private-token") {
			t.Fatal("token was not confined to the authorization header")
		}
		body := `{"results":[{"id":329865,"title":"Arrival","release_date":"2016-11-10","overview":"A linguist investigates.","poster_path":"/poster.jpg"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	provider := NewTMDB("private-token", client)
	for range 2 {
		results, err := provider.Search(context.Background(), TMDBQuery{Title: "Arrival", Kind: "movies"})
		if err != nil || len(results) != 1 || results[0].Year != 2016 || results[0].ProviderID != "329865" || results[0].MediaType != "movie" {
			t.Fatalf("results=%#v err=%v", results, err)
		}
	}
	if calls != 1 {
		t.Fatalf("requests=%d, want one cached request", calls)
	}
}

func TestTMDBUsesTVSearchAndRejectsInvalidInputs(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/3/search/tv" || request.URL.Query().Get("include_adult") != "false" {
			t.Fatalf("unexpected request: %s", request.URL)
		}
		body := `{"results":[{"id":1,"name":"A Show","first_air_date":"2020-01-01"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	provider := NewTMDB("token", client)
	if results, err := provider.Search(context.Background(), TMDBQuery{Title: "A Show", Kind: "anime"}); err != nil || len(results) != 1 || results[0].MediaType != "tv" {
		t.Fatalf("results=%#v err=%v", results, err)
	}
	if _, err := provider.Search(context.Background(), TMDBQuery{Title: "Book", Kind: "books"}); err == nil {
		t.Fatal("publication kind was accepted")
	}
	if _, err := NewTMDB("", client).Search(context.Background(), TMDBQuery{Title: "A Show", Kind: "tv"}); err == nil {
		t.Fatal("missing token was accepted")
	}
}

func TestTMDBConnectionCheckUsesAuthenticationEndpoint(t *testing.T) {
	provider := NewTMDB("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/3/authentication" || request.Header.Get("Authorization") != "Bearer private-token" {
			t.Fatalf("unexpected connection request: %s", request.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"success":true}`))}, nil
	})})
	if err := provider.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
}
