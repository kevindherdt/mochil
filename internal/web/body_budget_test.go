package web

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStalledJSONBodyIsReleasedWithinItsBudget(t *testing.T) {
	previous := smallBodyBudget
	smallBodyBudget = 300 * time.Millisecond
	defer func() { smallBodyBudget = previous }()
	released := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value map[string]any
		released <- decodeJSONRequest(w, r, 1<<10, &value, false)
	}))
	defer server.Close()
	connection, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	// Complete headers promptly, promise a body, then send only part of it and stall.
	w := bufio.NewWriter(connection)
	_, _ = w.WriteString("POST / HTTP/1.1\r\nHost: x\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{\"a\":")
	_ = w.Flush()
	select {
	case err := <-released:
		if err == nil {
			t.Fatal("a stalled body must fail")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the handler was still waiting for the body after its budget")
	}
}
