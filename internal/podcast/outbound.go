package podcast

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var errUnsafePodcastURL = errors.New("podcast URL must use a public HTTP or HTTPS address")

func validatePodcastURL(raw string, publicOnly bool) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errUnsafePodcastURL
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" || publicOnly && (host == "localhost" || strings.HasSuffix(host, ".localhost")) {
		return nil, errUnsafePodcastURL
	}
	if address, err := netip.ParseAddr(host); err == nil && publicOnly && !publicPodcastAddress(address.Unmap()) {
		return nil, errUnsafePodcastURL
	}
	return parsed, nil
}

func publicPodcastAddress(address netip.Addr) bool {
	return address.IsValid() && !address.IsPrivate() && !address.IsLoopback() &&
		!address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() &&
		!address.IsMulticast() && !address.IsUnspecified() &&
		!address.IsInterfaceLocalMulticast() && !address.Is4In6() &&
		!netip.MustParsePrefix("100.64.0.0/10").Contains(address) &&
		!netip.MustParsePrefix("192.0.0.0/24").Contains(address) &&
		!netip.MustParsePrefix("198.18.0.0/15").Contains(address) &&
		!netip.MustParsePrefix("2001:db8::/32").Contains(address)
}

// NewPublicHTTPClient returns a redirect-aware client that resolves and dials
// only public addresses. It protects feed, artwork, and enclosure fetches from
// loopback, private-network, link-local, and DNS-rebinding targets.
func NewPublicHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errUnsafePodcastURL
		}
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		for _, candidate := range addresses {
			candidate = candidate.Unmap()
			if !publicPodcastAddress(candidate) {
				continue
			}
			connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
			if dialErr == nil {
				return connection, nil
			}
			err = dialErr
		}
		if err != nil {
			return nil, err
		}
		return nil, errUnsafePodcastURL
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			// Podcast CDNs commonly chain attribution, ad measurement, and the
			// final enclosure host. Match net/http's conventional bound while
			// still validating every hop against local/private destinations.
			if len(via) >= 10 {
				return errors.New("too many podcast feed redirects")
			}
			_, err := validatePodcastURL(request.URL.String(), true)
			return err
		},
	}
}
