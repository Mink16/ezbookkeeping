package geocoding

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
)

const nominatimSearchUrl = "https://nominatim.openstreetmap.org/search"

// nominatimUserAgent is the identifiable user agent required by the nominatim usage policy
const nominatimUserAgent = "ezbookkeeping-custom/1.0 (self-hosted fork; +https://github.com/Mink16/ezbookkeeping)"

// nominatimMinRequestInterval is the minimum interval between two nominatim requests (usage policy: max 1 req/s)
const nominatimMinRequestInterval = 1100 * time.Millisecond

// package-level throttle state shared by all nominatim provider instances
var (
	nominatimThrottleMutex   sync.Mutex
	nominatimLastRequestTime time.Time
)

// NominatimQueryKind represents what kind of query is sent to nominatim,
// it determines the confidence of the hits
type NominatimQueryKind byte

// Nominatim Query Kind
const (
	NOMINATIM_QUERY_KIND_NAME    NominatimQueryKind = 0 // merchant name based query, hits are low confidence
	NOMINATIM_QUERY_KIND_ADDRESS NominatimQueryKind = 1 // address based query, hits are high confidence
)

// NominatimProvider represents the geocoding provider of OSM Nominatim search api
type NominatimProvider struct {
	httpClient *http.Client
	queryKind  NominatimQueryKind
}

// nominatimSearchResult represents one item in the nominatim jsonv2 search response
type nominatimSearchResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

// NewNominatimProvider creates a new OSM Nominatim geocoding provider
func NewNominatimProvider(httpClient *http.Client, queryKind NominatimQueryKind) *NominatimProvider {
	return &NominatimProvider{
		httpClient: httpClient,
		queryKind:  queryKind,
	}
}

// Name returns the unique name of the nominatim geocoding provider
func (p *NominatimProvider) Name() string {
	return GEOCODING_PROVIDER_NOMINATIM
}

// Search returns the best matched location of the given query, (nil, nil) means no hit,
// requests are throttled to the minimum interval required by the nominatim usage policy
func (p *NominatimProvider) Search(c core.Context, query string) (*GeocodingResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "jsonv2")
	params.Set("limit", "1")
	params.Set("countrycodes", "jp")
	params.Set("accept-language", "ja")

	req, err := http.NewRequest("GET", nominatimSearchUrl+"?"+params.Encode(), nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", nominatimUserAgent)

	waitForNominatimRequestSlot()

	resp, err := p.httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		log.Warnf(c, "[nominatim_provider.Search] failed to search \"%s\", because response code is %d", query, resp.StatusCode)
		return nil, errors.New("unexpected nominatim search response code")
	}

	return parseNominatimSearchResponse(body, nominatimQueryKindConfidence(p.queryKind))
}

// waitForNominatimRequestSlot blocks until the minimum interval since the last nominatim request has passed
func waitForNominatimRequestSlot() {
	nominatimThrottleMutex.Lock()
	defer nominatimThrottleMutex.Unlock()

	if !nominatimLastRequestTime.IsZero() {
		elapsed := time.Since(nominatimLastRequestTime)

		if elapsed < nominatimMinRequestInterval {
			time.Sleep(nominatimMinRequestInterval - elapsed)
		}
	}

	nominatimLastRequestTime = time.Now()
}

// nominatimQueryKindConfidence returns the result confidence for the given query kind,
// address query hits are high confidence and merchant name query hits are low confidence
func nominatimQueryKindConfidence(queryKind NominatimQueryKind) GeocodingConfidence {
	if queryKind == NOMINATIM_QUERY_KIND_ADDRESS {
		return GEOCODING_CONFIDENCE_HIGH
	}

	return GEOCODING_CONFIDENCE_LOW
}

// parseNominatimSearchResponse parses the nominatim jsonv2 search response array and adopts the first item
func parseNominatimSearchResponse(content []byte, confidence GeocodingConfidence) (*GeocodingResult, error) {
	var results []*nominatimSearchResult
	err := json.Unmarshal(content, &results)

	if err != nil {
		return nil, err
	}

	if len(results) < 1 || results[0] == nil {
		return nil, nil
	}

	result := results[0]
	latitude, err := strconv.ParseFloat(result.Lat, 64)

	if err != nil {
		return nil, err
	}

	longitude, err := strconv.ParseFloat(result.Lon, 64)

	if err != nil {
		return nil, err
	}

	return &GeocodingResult{
		Latitude:    latitude,
		Longitude:   longitude,
		DisplayName: result.DisplayName,
		Confidence:  confidence,
		Provider:    GEOCODING_PROVIDER_NOMINATIM,
	}, nil
}
