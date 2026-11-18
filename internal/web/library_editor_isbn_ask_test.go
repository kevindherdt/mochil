package web

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mochil/internal/appconfig"
)

// epubWithTwoEqualNumbers prints two ISBNs on the copyright page with nothing to tell them apart, so the automatic
// search is not sure which is the book's own.
func epubWithTwoEqualNumbers(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for _, file := range [][2]string{
		{"META-INF/container.xml", `<container><rootfiles><rootfile full-path="c.opf"/></rootfiles></container>`},
		{"c.opf", `<package><manifest><item id="a" href="copyright.xhtml"/></manifest><spine><itemref idref="a"/></spine><guide><reference type="copyright-page" href="copyright.xhtml"/></guide></package>`},
		{"copyright.xhtml", `<html><body><p>Example Press 2020. ISBN 978-0-306-40615-7</p><p>Distributed by Other House. ISBN 978-0-7432-7356-5</p></body></html>`},
	} {
		writer, _ := archive.Create(file[0])
		_, _ = writer.Write([]byte(file[1]))
	}
	_ = archive.Close()
	return buffer.Bytes()
}

func askServer(t *testing.T, reply string) (*multiServer, *string) {
	t.Helper()
	server := countServer(t)
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", epubBackend{staticBackend: staticBackend{books: server.libraries[0].Store.Books()}, file: epubWithTwoEqualNumbers(t)})
	var err error
	if server.isbnFound, err = newISBNStore(t); err != nil {
		t.Fatal(err)
	}
	var sent string
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		payload, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}}}})
		_, _ = w.Write(payload)
	}))
	t.Cleanup(hermes.Close)
	server.cfg.Hermes = appconfig.Hermes{URL: hermes.URL, Key: "k", HelperURL: hermes.URL, HelperModel: "m"}
	return server, &sent
}

func runISBNAsk(t *testing.T, server *multiServer) map[string]any {
	t.Helper()
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/isbn-ask", `{"library":"books"}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		var status map[string]any
		_ = json.Unmarshal(editorCall(server, http.MethodGet, "/api/metadata/editor/isbn-ask?library=books", "").Body.Bytes(), &status)
		if status["status"] == "ready" || status["status"] == "failed" {
			return status
		}
	}
	t.Fatal("asking did not finish")
	return nil
}

func TestHermesSettlesACloseCallAndTheChoiceIsOnlyAFinding(t *testing.T) {
	server, sent := askServer(t, `{"isbn":"978-0-306-40615-7","confidence":"high","reason":"printed with the publisher's own imprint"}`)
	status := runISBNAsk(t, server)
	if status["status"] != "ready" || !strings.Contains(status["message"].(string), "1 picked") {
		t.Fatalf("status = %v", status)
	}
	if !strings.Contains(*sent, "9780306406157") || !strings.Contains(*sent, "Distributed by Other House") {
		t.Fatalf("Hermes must be shown every number and the words around it: %s", *sent)
	}
	rows := listEditor(t, server, "&filter=isbn_found")
	if rows.Total != 1 || rows.Items[0].IsbnFound != "9780306406157" || !strings.Contains(rows.Items[0].IsbnNote, "picked by Hermes") {
		t.Fatalf("found = %+v", rows)
	}
	if strings.Contains(rows.Items[0].Identifiers, "isbn:") || listEditor(t, server, "&filter=edited").Total != 0 {
		t.Fatal("the choice must not be written into the book")
	}
}

func TestAnInventedISBNIsNeverTaken(t *testing.T) {
	// 978-0-14-103614-4 is a correct ISBN, but it is not in this book.
	server, _ := askServer(t, `{"isbn":"9780141036144","confidence":"high","reason":"I recognise this title"}`)
	status := runISBNAsk(t, server)
	if !strings.Contains(status["message"].(string), "0 picked, 1 still unsure") {
		t.Fatalf("status = %v", status)
	}
	if rows := listEditor(t, server, "&filter=isbn_found"); rows.Total != 0 {
		t.Fatalf("an invented number must not become a finding: %+v", rows)
	}
	if rows := listEditor(t, server, "&filter=isbn_unsure"); rows.Total != 1 {
		t.Fatalf("the book stays unsure: %+v", rows)
	}
}

func TestALowConfidenceAnswerIsNotTaken(t *testing.T) {
	server, _ := askServer(t, `{"isbn":"9780306406157","confidence":"low","reason":"maybe"}`)
	if status := runISBNAsk(t, server); !strings.Contains(status["message"].(string), "0 picked") {
		t.Fatalf("status = %v", status)
	}
}

func TestAskingNeedsHermesAndTheAbility(t *testing.T) {
	server, _ := askServer(t, `{}`)
	server.cfg.Hermes.Features = map[string]bool{"isbn": false}
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/isbn-ask", `{"library":"books"}`); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("switched off = %d", response.Code)
	}
	server.cfg.Hermes = appconfig.Hermes{}
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/isbn-ask", `{"library":"books"}`); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("not set up = %d", response.Code)
	}
}

func TestABookHermesCouldNotSettleIsNotAskedAgainOnTheNextRun(t *testing.T) {
	server, sent := askServer(t, `{"isbn":"9780306406157","confidence":"low","reason":"maybe"}`)
	if status := runISBNAsk(t, server); !strings.Contains(status["message"].(string), "0 picked, 1 still unsure") {
		t.Fatalf("first run = %v", status)
	}
	*sent = ""
	status := runISBNAsk(t, server)
	if *sent != "" {
		t.Fatal("Hermes was asked about the same book again; a second run must move on to other books")
	}
	if !strings.Contains(status["message"].(string), "asked before") {
		t.Fatalf("second run = %v", status)
	}
}
