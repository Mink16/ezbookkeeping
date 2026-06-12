package geocoding

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
)

type fakeGeocodingProvider struct {
	name    string
	result  *GeocodingResult
	err     error
	queries []string
}

func (p *fakeGeocodingProvider) Name() string {
	return p.name
}

func (p *fakeGeocodingProvider) Search(c core.Context, query string) (*GeocodingResult, error) {
	p.queries = append(p.queries, query)
	return p.result, p.err
}

func TestProviderChainSearch_FirstHitAdopted(t *testing.T) {
	firstResult := &GeocodingResult{Latitude: 35.0, Longitude: 139.0, Provider: "first"}
	secondResult := &GeocodingResult{Latitude: 36.0, Longitude: 140.0, Provider: "second"}
	firstProvider := &fakeGeocodingProvider{name: "first", result: firstResult}
	secondProvider := &fakeGeocodingProvider{name: "second", result: secondResult}
	chain := NewProviderChain([]Provider{firstProvider, secondProvider})

	actualResult, noHitConfirmed := chain.Search(core.NewNullContext(), func(providerName string) string {
		return "query for " + providerName
	})

	assert.Equal(t, firstResult, actualResult)
	assert.False(t, noHitConfirmed)
	assert.Equal(t, []string{"query for first"}, firstProvider.queries)
	assert.Empty(t, secondProvider.queries)
}

func TestProviderChainSearch_EmptyQuerySkipsProvider(t *testing.T) {
	secondResult := &GeocodingResult{Latitude: 36.0, Longitude: 140.0, Provider: "second"}
	firstProvider := &fakeGeocodingProvider{name: "first", result: &GeocodingResult{}}
	secondProvider := &fakeGeocodingProvider{name: "second", result: secondResult}
	chain := NewProviderChain([]Provider{firstProvider, secondProvider})

	actualResult, noHitConfirmed := chain.Search(core.NewNullContext(), func(providerName string) string {
		if providerName == "first" {
			return ""
		}

		return "second query"
	})

	assert.Equal(t, secondResult, actualResult)
	assert.False(t, noHitConfirmed)
	assert.Empty(t, firstProvider.queries)
	assert.Equal(t, []string{"second query"}, secondProvider.queries)
}

func TestProviderChainSearch_ErrorFallsThroughToNextProvider(t *testing.T) {
	secondResult := &GeocodingResult{Latitude: 36.0, Longitude: 140.0, Provider: "second"}
	firstProvider := &fakeGeocodingProvider{name: "first", err: errors.New("first provider error")}
	secondProvider := &fakeGeocodingProvider{name: "second", result: secondResult}
	chain := NewProviderChain([]Provider{firstProvider, secondProvider})

	actualResult, noHitConfirmed := chain.Search(core.NewNullContext(), func(providerName string) string {
		return "query"
	})

	assert.Equal(t, secondResult, actualResult)
	assert.False(t, noHitConfirmed)
	assert.Equal(t, []string{"query"}, firstProvider.queries)
	assert.Equal(t, []string{"query"}, secondProvider.queries)
}

func TestProviderChainSearch_NoHitReturnsNil(t *testing.T) {
	firstProvider := &fakeGeocodingProvider{name: "first"}
	secondProvider := &fakeGeocodingProvider{name: "second"}
	chain := NewProviderChain([]Provider{firstProvider, secondProvider})

	actualResult, noHitConfirmed := chain.Search(core.NewNullContext(), func(providerName string) string {
		return "query"
	})

	assert.Nil(t, actualResult)
	assert.True(t, noHitConfirmed)
	assert.Equal(t, []string{"query"}, firstProvider.queries)
	assert.Equal(t, []string{"query"}, secondProvider.queries)
}

func TestProviderChainSearch_NoProviders(t *testing.T) {
	chain := NewProviderChain(nil)

	actualResult, noHitConfirmed := chain.Search(core.NewNullContext(), func(providerName string) string {
		return "query"
	})

	assert.Nil(t, actualResult)
	assert.False(t, noHitConfirmed)
}

func TestProviderChainSearch_AllProvidersErrorDoesNotConfirmNoHit(t *testing.T) {
	firstProvider := &fakeGeocodingProvider{name: "first", err: errors.New("first provider error")}
	secondProvider := &fakeGeocodingProvider{name: "second", err: errors.New("second provider error")}
	chain := NewProviderChain([]Provider{firstProvider, secondProvider})

	actualResult, noHitConfirmed := chain.Search(core.NewNullContext(), func(providerName string) string {
		return "query"
	})

	assert.Nil(t, actualResult)
	assert.False(t, noHitConfirmed)
}

func TestProviderChainSearch_OneErrorAndOneCleanNoHitConfirmsNoHit(t *testing.T) {
	firstProvider := &fakeGeocodingProvider{name: "first", err: errors.New("first provider error")}
	secondProvider := &fakeGeocodingProvider{name: "second"}
	chain := NewProviderChain([]Provider{firstProvider, secondProvider})

	actualResult, noHitConfirmed := chain.Search(core.NewNullContext(), func(providerName string) string {
		return "query"
	})

	assert.Nil(t, actualResult)
	assert.True(t, noHitConfirmed)
}

func TestProviderChainSearch_AllQueriesEmptyDoesNotConfirmNoHit(t *testing.T) {
	firstProvider := &fakeGeocodingProvider{name: "first"}
	chain := NewProviderChain([]Provider{firstProvider})

	actualResult, noHitConfirmed := chain.Search(core.NewNullContext(), func(providerName string) string {
		return ""
	})

	assert.Nil(t, actualResult)
	assert.False(t, noHitConfirmed)
	assert.Empty(t, firstProvider.queries)
}
