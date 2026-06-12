package geocoding

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
)

const gsiAddressSearchUrl = "https://msearch.gsi.go.jp/address-search/AddressSearch"

// GsiProvider represents the geocoding provider of GSI (Geospatial Information Authority of Japan) address search api
type GsiProvider struct {
	httpClient *http.Client
}

// gsiAddressSearchFeature represents one GeoJSON feature in the GSI address search response,
// the coordinates are in [longitude, latitude] order
type gsiAddressSearchFeature struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
		Type        string    `json:"type"`
	} `json:"geometry"`
	Properties struct {
		Title string `json:"title"`
	} `json:"properties"`
}

// NewGsiProvider creates a new GSI address search geocoding provider
func NewGsiProvider(httpClient *http.Client) *GsiProvider {
	return &GsiProvider{
		httpClient: httpClient,
	}
}

// Name returns the unique name of the GSI geocoding provider
func (p *GsiProvider) Name() string {
	return GEOCODING_PROVIDER_GSI
}

// Search returns the best matched location of the given address query, (nil, nil) means no hit,
// when the full address has no hit, it retries once with the last block number segment removed
func (p *GsiProvider) Search(c core.Context, query string) (*GeocodingResult, error) {
	result, err := p.searchOnce(c, query)

	if err != nil || result != nil {
		return result, err
	}

	fallbackQuery := trimLastBlockNumberSegment(query)

	if fallbackQuery == "" || fallbackQuery == query {
		return nil, nil
	}

	log.Infof(c, "[gsi_provider.Search] no hit for \"%s\", retrying with \"%s\"", query, fallbackQuery)

	return p.searchOnce(c, fallbackQuery)
}

func (p *GsiProvider) searchOnce(c core.Context, query string) (*GeocodingResult, error) {
	requestUrl := gsiAddressSearchUrl + "?q=" + url.QueryEscape(query)
	req, err := http.NewRequest("GET", requestUrl, nil)

	if err != nil {
		return nil, err
	}

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
		log.Warnf(c, "[gsi_provider.searchOnce] failed to search \"%s\", because response code is %d", query, resp.StatusCode)
		return nil, errors.New("unexpected gsi address search response code")
	}

	return parseGsiAddressSearchResponse(body)
}

// parseGsiAddressSearchResponse parses the GSI address search GeoJSON feature array,
// the first feature is adopted and its coordinates are in [longitude, latitude] order
func parseGsiAddressSearchResponse(content []byte) (*GeocodingResult, error) {
	var features []*gsiAddressSearchFeature
	err := json.Unmarshal(content, &features)

	if err != nil {
		return nil, err
	}

	if len(features) < 1 {
		return nil, nil
	}

	feature := features[0]

	if feature == nil || len(feature.Geometry.Coordinates) < 2 {
		return nil, errors.New("invalid coordinates in gsi address search response")
	}

	return &GeocodingResult{
		Latitude:    feature.Geometry.Coordinates[1],
		Longitude:   feature.Geometry.Coordinates[0],
		DisplayName: feature.Properties.Title,
		Confidence:  GEOCODING_CONFIDENCE_HIGH,
		Provider:    GEOCODING_PROVIDER_GSI,
	}, nil
}

// isBlockNumberDigit returns whether the rune is a half-width or full-width digit
func isBlockNumberDigit(ch rune) bool {
	return ('0' <= ch && ch <= '9') || ('０' <= ch && ch <= '９')
}

// isBlockNumberHyphen returns whether the rune is a hyphen-like character used in japanese block numbers
func isBlockNumberHyphen(ch rune) bool {
	switch ch {
	case '-', '‐', '‑', '–', '—', '―', '−', 'ー', '－':
		return true
	}

	return false
}

// trimLastBlockNumberSegment drops the last segment of the trailing block number token
// (half-width / full-width digits joined by hyphen-like characters) from the address,
// e.g. "道玄坂1-2-3" becomes "道玄坂1-2" and "道玄坂1" becomes "道玄坂",
// an empty string is returned when the address has no trailing block number token
func trimLastBlockNumberSegment(address string) string {
	runes := []rune(address)
	end := len(runes)

	for end > 0 && (runes[end-1] == ' ' || runes[end-1] == '　') {
		end--
	}

	tokenStart := end

	for tokenStart > 0 && (isBlockNumberDigit(runes[tokenStart-1]) || isBlockNumberHyphen(runes[tokenStart-1])) {
		tokenStart--
	}

	if tokenStart >= end {
		return ""
	}

	cutAt := tokenStart

	for i := end - 1; i > tokenStart; i-- {
		if isBlockNumberHyphen(runes[i]) {
			cutAt = i
			break
		}
	}

	trimmed := string(runes[:cutAt])

	if trimmed == "" {
		return ""
	}

	return trimmed
}
