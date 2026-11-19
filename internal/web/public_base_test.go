package web

import (
	"math"
	"net/http/httptest"
	"testing"

	"mochil/internal/catalog"
)

// A forwarded "http" must not be believed: it would downgrade a link the server is serving over TLS.
func TestPublicBaseNeverDowngradesToCleartext(t *testing.T) {
	server := &multiServer{}
	tlsRequest := httptest.NewRequest("GET", "https://books.example/pair", nil)
	tlsRequest.Host = "books.example"
	tlsRequest.Header.Set("X-Forwarded-Proto", "http")
	if got := server.publicBase(tlsRequest); got != "https://books.example" {
		t.Fatalf("publicBase = %q, want https", got)
	}
	plain := httptest.NewRequest("GET", "http://books.local:8099/pair", nil)
	plain.Host = "books.local:8099"
	plain.Header.Set("X-Forwarded-Proto", "https")
	if got := server.publicBase(plain); got != "https://books.local:8099" {
		t.Fatalf("publicBase = %q, want the forwarded https", got)
	}
}

// A Host that is not a bare host:port would be echoed into a link a person is asked to open.
func TestPublicBaseRejectsAHostThatIsNotAHost(t *testing.T) {
	server := &multiServer{}
	for _, host := range []string{"", "evil.example/steal", "user:pass@evil.example", "evil.example?x=1", "a b"} {
		request := httptest.NewRequest("GET", "http://localhost/pair", nil)
		request.Host = host
		if got := server.publicBase(request); got != "" {
			t.Fatalf("publicBase(%q) = %q, want empty", host, got)
		}
	}
}

// A NaN index makes every comparison false, so the series order is not total and there is no honest
// neighbour to offer.
func TestSeriesNeighboursIgnoreANonFiniteIndex(t *testing.T) {
	nan := math.NaN()
	book := catalog.Book{ID: "x", Series: "Dune", SeriesIndex: &nan}
	other := 2.0
	previous, next, ok := seriesNeighbours([]catalog.Book{{ID: "a", Series: "Dune", SeriesIndex: &other}}, book)
	if ok || previous.ID != "" || next.ID != "" {
		t.Fatalf("expected no neighbours, got %q and %q (ok=%v)", previous.ID, next.ID, ok)
	}
}
