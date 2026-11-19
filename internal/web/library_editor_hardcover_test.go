package web

import (
	"net/http"
	"testing"
)

func TestPickedBooksCanBeLeftOutOfTheHardcoverSyncAndPutBack(t *testing.T) {
	server := editorTestServer(t)
	server.hardcoverMatches.put(hardcoverMatch{Library: "books", Book: "a1", Title: "Dune", HardcoverID: "1", HardcoverTitle: "Dune", How: "matched", Result: "added", At: 1})
	state := func(id string) string {
		for _, row := range listEditor(t, server, "").Items {
			if row.ID == id {
				return row.Hardcover
			}
		}
		return "missing"
	}
	call := func(body string) (int, string) {
		response := editorCall(server, http.MethodPost, "/api/metadata/editor/hardcover-sync", body)
		return response.Code, response.Body.String()
	}
	if code, body := call(`{"library":"books","ids":["a1","a2","n1"],"ignore":true}`); code != http.StatusOK || body != "{\"changed\":3}\n" {
		t.Fatalf("leave out: %d %s", code, body)
	}
	if state("a1") != "ignored" || state("n1") != "ignored" || state("u1") != "" {
		t.Fatalf("states: a1=%q n1=%q u1=%q", state("a1"), state("n1"), state("u1"))
	}
	if _, body := call(`{"library":"books","ids":["a1","a2"],"ignore":true}`); body != "{\"changed\":0}\n" {
		t.Fatalf("leaving out books that are already out changes nothing: %s", body)
	}
	if _, body := call(`{"library":"books","ids":["a1","u1"],"ignore":false}`); body != "{\"changed\":1}\n" {
		t.Fatalf("putting back counts only the books that were out: %s", body)
	}
	if state("a1") == "ignored" {
		t.Fatalf("a1 is still left out")
	}
	if match, _ := server.hardcoverMatches.get("books", "a1"); match.Result != "waiting to be sent" {
		t.Fatalf("putting a book back lets the sync add it to the shelf again: result = %q", match.Result)
	}
	if code, _ := call(`{"library":"books","ids":[],"ignore":true}`); code != http.StatusBadRequest {
		t.Fatalf("no books: %d", code)
	}
	server.cfg.Modules = map[string]bool{"hardcover": false}
	if code, _ := call(`{"library":"books","ids":["a1"],"ignore":true}`); code != http.StatusServiceUnavailable {
		t.Fatalf("module off: %d", code)
	}
}
