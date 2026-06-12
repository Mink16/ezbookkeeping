// Package geocoding provides forward geocoding providers used to resolve
// receipt merchant locations to geographic coordinates (fork-local feature)
package geocoding

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
)

// GeocodingConfidence represents how reliable a geocoding result is
type GeocodingConfidence byte

// Geocoding Confidence
const (
	GEOCODING_CONFIDENCE_LOW  GeocodingConfidence = 0
	GEOCODING_CONFIDENCE_HIGH GeocodingConfidence = 1
)

// Geocoding provider names
const (
	GEOCODING_PROVIDER_GSI       = "gsi"
	GEOCODING_PROVIDER_NOMINATIM = "nominatim"
)

// GeocodingResult represents a resolved geographic location
type GeocodingResult struct {
	Latitude    float64
	Longitude   float64
	DisplayName string
	Confidence  GeocodingConfidence
	Provider    string
}

// Provider defines the structure of a forward geocoding provider
type Provider interface {
	// Name returns the unique name of the geocoding provider
	Name() string

	// Search returns the best matched location of the given query, (nil, nil) means no hit
	Search(c core.Context, query string) (*GeocodingResult, error)
}

// ProviderChain executes geocoding providers in the configured order and adopts the first hit
type ProviderChain struct {
	providers []Provider
}

// NewProviderChain creates a new geocoding provider chain with the given providers
func NewProviderChain(providers []Provider) *ProviderChain {
	return &ProviderChain{
		providers: providers,
	}
}

// Search tries each provider in the configured order with the provider-specific query built by buildQuery
// (an empty query skips that provider) and returns the first hit, provider errors are logged and the next
// provider is tried, nil is returned when no provider hits. The second return value reports whether the
// no-hit result is definitive: it is true only when at least one provider was tried and every tried
// provider completed without error, so that a transient failure of any provider (network error, timeout,
// rate limiting etc.) never gets treated as a confirmed no-hit by the caller's negative cache
func (p *ProviderChain) Search(c core.Context, buildQuery func(providerName string) string) (*GeocodingResult, bool) {
	tried := false
	failed := false

	for i := 0; i < len(p.providers); i++ {
		provider := p.providers[i]
		query := buildQuery(provider.Name())

		if query == "" {
			continue
		}

		result, err := provider.Search(c, query)

		if err != nil {
			log.Warnf(c, "[geocoding.Search] provider \"%s\" failed to search \"%s\", because %s", provider.Name(), query, err.Error())
			failed = true
			continue
		}

		if result != nil {
			return result, false
		}

		tried = true
	}

	return nil, tried && !failed
}
