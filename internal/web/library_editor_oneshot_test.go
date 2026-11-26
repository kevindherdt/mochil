package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// The editor's whole-library view asks for `limit=all` so the server builds, filters and sorts the library once
// instead of once per 1 000-row chunk. Measured on the 5 544-book `test` library before this existed: six requests at
// ~80 ms each, about 400 ms of the ~485 ms the screen took to load, every one of them rebuilding an identical list.
func TestAskingForEveryRowReturnsThemAllInOnePage(t *testing.T) {
	server := editorTestServer(t)

	oneShot := listEditor(t, server, "&limit=all")
	chunked := listEditor(t, server, "&limit=1000")
	if oneShot.Total != chunked.Total || len(oneShot.Items) != len(chunked.Items) {
		t.Fatalf("limit=all returned %d of %d, chunked returned %d of %d", len(oneShot.Items), oneShot.Total, len(chunked.Items), chunked.Total)
	}
	// Same rows, same order: the one-shot path must not become a different query.
	for i := range oneShot.Items {
		if oneShot.Items[i].ID != chunked.Items[i].ID {
			t.Fatalf("row %d differs: %q vs %q", i, oneShot.Items[i].ID, chunked.Items[i].ID)
		}
	}
}

// A library larger than the payload cap must not silently truncate: the response says how many rows exist, and what it
// sent is the first slice of them, which is exactly what the client's fallback compares to decide to keep chunking.
func TestOneShotIsCappedButStillReportsTheRealTotal(t *testing.T) {
	server := editorTestServer(t)

	// The cap is a package constant; assert the invariant rather than the number, so raising it cannot quietly break
	// the contract the client depends on.
	all := listEditor(t, server, "&limit=all")
	if len(all.Items) > editorOneShotMaximum {
		t.Fatalf("sent %d rows, cap is %d", len(all.Items), editorOneShotMaximum)
	}
	if all.Total < len(all.Items) {
		t.Fatalf("total %d is less than the %d rows sent", all.Total, len(all.Items))
	}
	// And a bigger cap than the library changes nothing: every row, in one page.
	if len(all.Items) != all.Total {
		t.Fatalf("a %d-book library should fit the %d-row cap, got %d rows", all.Total, editorOneShotMaximum, len(all.Items))
	}
}

// `limit=all` must not turn into "no limit" for a bad value: anything unparseable still gets the default page, and the
// offset still slices, so a client cannot ask for row 9 000 of a 5-row library and get a panic.
func TestAnOffsetPastTheEndIsEmptyRatherThanOutOfRange(t *testing.T) {
	server := editorTestServer(t)
	response := editorCall(server, http.MethodGet, fmt.Sprintf("/api/metadata/editor/books?library=books&limit=all&offset=%d", 9999), "")
	if response.Code != http.StatusOK {
		t.Fatalf("offset past the end = %d, want 200", response.Code)
	}
	var out editorList
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Items) != 0 || out.Total != 5 {
		t.Fatalf("got %d rows of total %d, want 0 of 5", len(out.Items), out.Total)
	}
}
