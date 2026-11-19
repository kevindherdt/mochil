// Package acquisition owns bounded access to publication bytes across storage
// backends. It does not interpret formats or know about HTTP application routes.
package acquisition

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"mochil/internal/catalog"
)

type Service struct {
	Client *http.Client
}

type Opened struct {
	Reader   catalog.ReadSeekCloser
	Response *http.Response
}

func (o *Opened) Close() error {
	if o == nil {
		return nil
	}
	if o.Reader != nil {
		return o.Reader.Close()
	}
	if o.Response != nil && o.Response.Body != nil {
		return o.Response.Body.Close()
	}
	return nil
}

// Open resolves a backend reader or opens its temporary remote URL. Range is
// forwarded verbatim; interpreting HTTP range/status semantics remains the
// delivery adapter's responsibility.
func (s Service) Open(ctx context.Context, backend catalog.Backend, book catalog.Book, byteRange string) (*Opened, error) {
	if backend == nil {
		return nil, errors.New("acquisition backend is unavailable")
	}
	acquired, err := backend.Acquire(ctx, book)
	if err != nil {
		return nil, err
	}
	if acquired.Reader != nil {
		return &Opened{Reader: acquired.Reader}, nil
	}
	if acquired.RedirectURL == "" {
		return nil, errors.New("acquisition returned neither a reader nor a redirect")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, acquired.RedirectURL, nil)
	if err != nil {
		return nil, err
	}
	if byteRange != "" {
		request.Header.Set("Range", byteRange)
	}
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	return &Opened{Response: response}, nil
}

// ReadAll resolves either a backend reader or a temporary redirect and returns
// at most maximum bytes. Callers must choose a limit appropriate to their use.
func (s Service) ReadAll(ctx context.Context, backend catalog.Backend, book catalog.Book, maximum int64) ([]byte, error) {
	if maximum <= 0 {
		return nil, errors.New("acquisition byte limit must be positive")
	}
	opened, err := s.Open(ctx, backend, book, "")
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	var reader io.ReadCloser
	if opened.Reader != nil {
		reader = opened.Reader
	} else {
		if opened.Response.StatusCode < 200 || opened.Response.StatusCode >= 300 {
			return nil, fmt.Errorf("acquisition returned HTTP %d", opened.Response.StatusCode)
		}
		reader = opened.Response.Body
	}
	data, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum {
		return nil, fmt.Errorf("acquisition exceeds %d byte limit", maximum)
	}
	return data, nil
}
