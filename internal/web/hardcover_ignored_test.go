package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mochil/internal/reading"
)

func hardcoverIgnoreCall(server *multiServer, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.apiHardcover(response, request)
	return response
}

func TestABookRemovedFromTheHardcoverShelfIsNotAddedAgain(t *testing.T) {
	server, _ := browseTestServer(t)
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder) // its shelf is always empty, as if the reader deleted every entry
	if err := server.hardcoverIgnored.open(""); err != nil {
		t.Fatal(err)
	}
	job := hardcoverJob{account: "reader", event: reading.Reading, library: "books", book: "a1"}
	if sent, err := server.sendToHardcover(job); !sent || err != nil {
		t.Fatalf("first time: sent=%v err=%v", sent, err)
	}
	if got := recorder.got(); len(got) != 1 {
		t.Fatalf("the first sync should add the book: %v", got)
	}
	if sent, err := server.sendToHardcover(job); !sent || err != nil {
		t.Fatalf("second time: sent=%v err=%v", sent, err)
	}
	if got := recorder.got(); len(got) != 1 {
		t.Fatalf("a book that was on the shelf and is now gone must not be added again: %v", got)
	}
	if !server.hardcoverIgnored.has("books", "a1") || server.hardcoverIgnored.list()[0].Reason != ignoredRemoved {
		t.Fatalf("it should be remembered as removed: %+v", server.hardcoverIgnored.list())
	}
	if sent, _ := server.sendToHardcover(job); sent {
		t.Fatal("an ignored book costs no request")
	}
	if sent, _ := server.sendProgressToHardcover(hardcoverJob{account: "reader", library: "books", book: "a1", progress: .5, isProgress: true}); sent {
		t.Fatal("an ignored book gets no progress either")
	}
}

func TestStoppingAndResumingSyncOfOneBook(t *testing.T) {
	server, _ := browseTestServer(t)
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	if err := server.hardcoverIgnored.open(""); err != nil {
		t.Fatal(err)
	}
	if response := hardcoverIgnoreCall(server, "/api/metadata/hardcover/ignore", `{"library":"books","book":"a1","ignore":true}`); response.Code != http.StatusOK {
		t.Fatalf("stop = %d %s", response.Code, response.Body.String())
	}
	job := hardcoverJob{account: "reader", event: reading.Reading, library: "books", book: "a1"}
	if sent, _ := server.sendToHardcover(job); sent || len(recorder.got()) != 0 {
		t.Fatal("a book the reader stopped syncing was sent")
	}
	if response := hardcoverIgnoreCall(server, "/api/metadata/hardcover/ignore", `{"library":"books","book":"a1","ignore":false}`); response.Code != http.StatusOK {
		t.Fatalf("resume = %d", response.Code)
	}
	if sent, err := server.sendToHardcover(job); !sent || err != nil || len(recorder.got()) != 1 {
		t.Fatalf("resumed sync should add the book: sent=%v err=%v writes=%v", sent, err, recorder.got())
	}
	if response := hardcoverIgnoreCall(server, "/api/metadata/hardcover/ignore", `{"library":"books","book":"nope","ignore":true}`); response.Code != http.StatusNotFound {
		t.Fatalf("an unknown book should be refused: %d", response.Code)
	}
	if !strings.Contains(hardcoverIgnoreCall(server, "/api/metadata/hardcover/ignore", `{"library":"books","book":"a1","ignore":true}`).Body.String(), "true") {
		t.Fatal("stopping again should report ignored")
	}
}
