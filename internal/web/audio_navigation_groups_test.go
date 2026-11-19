package web

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"mochil/internal/audio"
	"mochil/internal/catalog"
)

func TestAudioNavigationMatchesGroupedPagination(t *testing.T) {
	root := t.TempDir()
	metadata := albumDownloadProber{
		"one.flac":   {Title: "Track-only query", Album: "Alpha", Artist: "Z performer", AlbumArtist: "A artist"},
		"two.flac":   {Title: "Second", Album: "Alpha", Artist: "A performer", AlbumArtist: "Z artist"},
		"three.flac": {Title: "Third", Album: "Beta", Artist: "M performer", AlbumArtist: "M artist"},
	}
	for _, name := range []string{"one.flac", "two.flac", "three.flac"} {
		file := filepath.Join(root, name, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := audio.NewStore("music", root, "")
	store.SetProber(metadata)
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	m := &multiServer{libraries: []Library{{ID: "music", Kind: "music", Audio: store}}}
	for _, mode := range []string{"albums", "artists"} {
		for _, query := range []string{"", "alpha", "artist", "Track-only query", "missing"} {
			t.Run(mode+"/"+query, func(t *testing.T) {
				groupResponse := httptest.NewRecorder()
				m.androidAudioGroups(groupResponse, httptest.NewRequest("GET", "/?library=music&group_type="+mode+"&q="+url.QueryEscape(query), nil))
				var groups []androidAudioGroup
				if err := json.Unmarshal(groupResponse.Body.Bytes(), &groups); err != nil {
					t.Fatal(err)
				}
				if query == "" && len(groups) != 3 {
					t.Fatalf("expected three distinct groups: %+v", groups)
				}
				if query == "Track-only query" && len(groups) != 0 {
					t.Fatal("group search must not match hidden track titles")
				}
				expected := []audioNavigationView{}
				for index, group := range groups {
					bucket := catalog.Bucket(group.Name)
					if len(expected) == 0 || expected[len(expected)-1].Bucket != bucket {
						expected = append(expected, audioNavigationView{Bucket: bucket, Offset: index})
					}
					expected[len(expected)-1].Count++
				}
				response := httptest.NewRecorder()
				m.apiAudioNavigation(response, httptest.NewRequest("GET", "/?library=music&mode="+mode+"&q="+url.QueryEscape(query), nil))
				var actual []audioNavigationView
				if err := json.Unmarshal(response.Body.Bytes(), &actual); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(expected, actual) {
					t.Fatalf("groups=%+v navigation=%+v want=%+v", groups, actual, expected)
				}
				// A stale offset after a rescan must terminate, not replay page one.
				for _, offset := range []int{len(groups), len(groups) + 100} {
					page := httptest.NewRecorder()
					m.androidAudioGroups(page, httptest.NewRequest("GET", "/?library=music&group_type="+mode+"&q="+url.QueryEscape(query)+"&limit=1&offset="+strconv.Itoa(offset), nil))
					if page.Code != 200 || page.Body.String() != "[]\n" || page.Header().Get("X-Next-Offset") != "" || page.Header().Get("X-Total-Count") != strconv.Itoa(len(groups)) {
						t.Fatalf("offset %d did not terminate: status=%d headers=%v body=%s", offset, page.Code, page.Header(), page.Body.String())
					}
				}
				// Equal album names have stable IDs so pagination cannot swap editions.
				for index, group := range groups {
					page := httptest.NewRecorder()
					m.androidAudioGroups(page, httptest.NewRequest("GET", "/?library=music&group_type="+mode+"&q="+url.QueryEscape(query)+"&limit=1&offset="+strconv.Itoa(index), nil))
					var single []androidAudioGroup
					if err := json.Unmarshal(page.Body.Bytes(), &single); err != nil {
						t.Fatal(err)
					}
					if len(single) != 1 || single[0].ID != group.ID {
						t.Fatalf("unstable page %d", index)
					}
				}
			})
		}
	}
}
