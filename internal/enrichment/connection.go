package enrichment

import "errors"

var (
	ErrProviderNotConnected = errors.New("provider is not connected")
	ErrProviderRejected     = errors.New("provider rejected the credential")
	ErrProviderRateLimited  = errors.New("provider rate limit reached")
	ErrProviderUnavailable  = errors.New("provider is temporarily unavailable")
)
