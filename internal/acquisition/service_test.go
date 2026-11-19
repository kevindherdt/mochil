package acquisition

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"mochil/internal/catalog"
)

type backend struct{ acquisition catalog.Acquisition }

func (backend) Scan() (catalog.Stats, error)     { return catalog.Stats{}, nil }
func (backend) Books() []catalog.Book            { return nil }
func (backend) Book(string) (catalog.Book, bool) { return catalog.Book{}, false }
func (b backend) Acquire(context.Context, catalog.Book) (catalog.Acquisition, error) {
	return b.acquisition, nil
}

func TestReadAllSupportsBackendReaderAndEnforcesLimit(t *testing.T) {
	service := Service{}
	data, err := service.ReadAll(context.Background(), backend{acquisition: catalog.Acquisition{Reader: reader("book")}}, catalog.Book{}, 4)
	if err != nil || string(data) != "book" {
		t.Fatalf("read = %q, %v", data, err)
	}
	if _, err := service.ReadAll(context.Background(), backend{acquisition: catalog.Acquisition{Reader: reader("large")}}, catalog.Book{}, 4); err == nil {
		t.Fatal("expected byte limit error")
	}
}

func TestReadAllFollowsTemporaryRedirectTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("remote")) }))
	defer server.Close()
	data, err := (Service{Client: server.Client()}).ReadAll(context.Background(), backend{acquisition: catalog.Acquisition{RedirectURL: server.URL}}, catalog.Book{}, 16)
	if err != nil || string(data) != "remote" {
		t.Fatalf("read = %q, %v", data, err)
	}
}

func TestOpenForwardsRangeAndPreservesRemoteStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Range") != "bytes=10-19" {
			t.Errorf("range = %q", request.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", "bytes 10-19/100")
		w.WriteHeader(http.StatusPartialContent)
	}))
	defer server.Close()
	opened, err := (Service{Client: server.Client()}).Open(context.Background(), backend{acquisition: catalog.Acquisition{RedirectURL: server.URL}}, catalog.Book{}, "bytes=10-19")
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if opened.Response == nil || opened.Response.StatusCode != http.StatusPartialContent || opened.Response.Header.Get("Content-Range") != "bytes 10-19/100" {
		t.Fatalf("opened = %#v", opened)
	}
}

func reader(value string) catalog.ReadSeekCloser {
	return readSeekCloser{Reader: bytes.NewReader([]byte(value))}
}

type readSeekCloser struct{ *bytes.Reader }

func (readSeekCloser) Close() error { return nil }
