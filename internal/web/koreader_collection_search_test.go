package web

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestKoReaderCollectionSearchFiltersBeforePaging(t *testing.T) {
	server, token := browseTestServer(t)
	collection, err := server.collections.create("reader", "Picks", "books", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a1", "a2"} {
		if err := server.collections.member("reader", collection.ID, collectionMember{Type: "book", LibraryID: "books", BookID: id}, true); err != nil {
			t.Fatal(err)
		}
	}
	base := "collection=" + url.QueryEscape(collection.ID)
	cases := []struct {
		query string
		total int
		ids   []string
	}{
		{"&q=MeSsIaH", 1, []string{"a2"}},
		{"&q=DuNe", 2, []string{"a1", "a2"}},
		{"&q=HERBERT&limit=1&offset=1", 2, []string{"a2"}},
		{"&q=Neuromancer", 0, nil},
		{"&q=+&limit=1&offset=1", 2, []string{"a2"}},
		{"&q=dune&limit=1&offset=2", 2, nil},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			got := browseRequest(t, server, token, base+tc.query)
			if got.Total != tc.total || len(got.Books) != len(tc.ids) {
				t.Fatalf("got %+v, want total %d ids %v", got, tc.total, tc.ids)
			}
			for i, id := range tc.ids {
				if got.Books[i].ID != id {
					t.Fatalf("got %+v, want ids %v", got, tc.ids)
				}
			}
		})
	}
	private, err := server.collections.create("someone-else", "Private", "books", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct {
		path, token string
		status      int
	}{
		{fmt.Sprintf("/api/koreader/v1/browse?collection=%s&q=dune", collection.ID), "", http.StatusUnauthorized},
		{fmt.Sprintf("/api/koreader/v1/browse?collection=%s&q=dune", private.ID), token, http.StatusBadRequest},
	} {
		resp := pluginUpdateRequest(server, http.MethodGet, target.path, target.token)
		if resp.Code != target.status {
			t.Errorf("%s: got %d want %d", target.path, resp.Code, target.status)
		}
	}
}
