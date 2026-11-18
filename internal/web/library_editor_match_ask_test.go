package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mochil/internal/appconfig"
)

// matchAskServer has Neuromancer (n1) found by title with two near-equal candidates, and a helper model that gives reply.
func matchAskServer(t *testing.T, reply string) (*multiServer, *string, *atomic.Int32) {
	t.Helper()
	server, _ := matchServer(t)
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/matches", `{"library":"books","ids":["n1"]}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	waitForMatches(t, server)
	var sent string
	calls := &atomic.Int32{}
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		calls.Add(1)
		payload, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}}}})
		_, _ = w.Write(payload)
	}))
	t.Cleanup(helper.Close)
	server.cfg.Hermes = appconfig.Hermes{URL: helper.URL, Key: "k", HelperURL: helper.URL, HelperModel: "m"}
	return server, &sent, calls
}

func runMatchAsk(t *testing.T, server *multiServer) map[string]any {
	t.Helper()
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/match-ask", `{"library":"books"}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		var status map[string]any
		_ = json.Unmarshal(editorCall(server, http.MethodGet, "/api/metadata/editor/match-ask?library=books", "").Body.Bytes(), &status)
		if status["status"] == "ready" || status["status"] == "failed" {
			return status
		}
	}
	t.Fatal("asking did not finish")
	return nil
}

func TestHermesSettlesAnUnsureMatchAndItIsOnlyAFinding(t *testing.T) {
	server, sent, _ := matchAskServer(t, `{"pick":1,"confidence":"high","reason":"the original novel, not the graphic adaptation"}`)
	status := runMatchAsk(t, server)
	if status["status"] != "ready" || !strings.Contains(status["message"].(string), "1 picked") {
		t.Fatalf("status = %v", status)
	}
	if !strings.Contains(*sent, "Neuromancer: The Graphic Novel") || !strings.Contains(*sent, "Library book: Neuromancer") {
		t.Fatalf("the model must be shown the book and every candidate: %s", *sent)
	}
	found := listEditor(t, server, "&filter=match_found")
	var picked bool
	for _, row := range found.Items {
		if row.ID == "n1" {
			picked = row.MatchClear && strings.Contains(row.MatchNote, "Picked by Hermes") && row.Match != nil && row.Match.Title == "Neuromancer"
		}
	}
	if !picked {
		t.Fatalf("n1 should now be a clear pick: %+v", found.Items)
	}
	if listEditor(t, server, "&filter=edited").Total != 0 {
		t.Fatal("a pick must not change the book")
	}
}

func TestAnUnusablePickIsNotTakenAndTheBookIsNotAskedAgain(t *testing.T) {
	for _, reply := range []string{
		`{"pick":7,"confidence":"high","reason":"x"}`,
		`{"pick":1,"confidence":"low","reason":"maybe"}`,
		`{"pick":null,"confidence":"high","reason":"none fits"}`,
	} {
		server, _, calls := matchAskServer(t, reply)
		if status := runMatchAsk(t, server); !strings.Contains(status["message"].(string), "0 picked, 1 still unsure") {
			t.Fatalf("%s: status = %v", reply, status)
		}
		if got := listEditor(t, server, "&filter=match_unsure"); got.Total != 1 {
			t.Fatalf("%s: n1 must stay unsure: %+v", reply, got)
		}
		before := calls.Load()
		if status := runMatchAsk(t, server); !strings.Contains(status["message"].(string), "Nothing to ask") || calls.Load() != before {
			t.Fatalf("%s: a judged book must not be asked again: %v", reply, status)
		}
	}
}

func TestMatchAskNeedsTheHelperAndTheSwitch(t *testing.T) {
	server, _, _ := matchAskServer(t, `{}`)
	server.cfg.Hermes.Features = map[string]bool{"match": false}
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/match-ask", `{"library":"books"}`); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("switched off = %d", response.Code)
	}
}
