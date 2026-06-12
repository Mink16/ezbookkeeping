package services

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/geocoding"
	"github.com/mayswind/ezbookkeeping/pkg/httpclient"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// environment variables configuring the receipt geocoding feature (fork-local, see custom_admins.go for the convention)
const (
	customGeocodingEnableEnvName         = "EBK_CUSTOM_GEOCODING_ENABLE"
	customGeocodingProvidersEnvName      = "EBK_CUSTOM_GEOCODING_PROVIDERS"
	customGeocodingRequestTimeoutEnvName = "EBK_CUSTOM_GEOCODING_REQUEST_TIMEOUT"
	customGeocodingProxyEnvName          = "EBK_CUSTOM_GEOCODING_PROXY"
)

const (
	customGeocodingDefaultProviders      = "gsi,nominatim"
	customGeocodingDefaultRequestTimeout = uint32(10000) // milliseconds
	customGeocodingDefaultProxy          = "system"
)

// customGeocodingNegativeCacheTTL is how long a cached no-hit result suppresses re-querying (positive results never expire)
const customGeocodingNegativeCacheTTL = int64(30 * 24 * 60 * 60) // 30 days in seconds

const customGeocodingMaxCachedQueryLength = 255

// customGeocodingCityPattern matches the leading prefecture + municipality part of a japanese address
var customGeocodingCityPattern = regexp.MustCompile(`^(?:.{2,3}[都道府県])?.{1,10}?[市区町村]`)

// customGeocodingPhoneMarkerPattern matches a "TEL" / "電話" marker only when it is followed by
// something that looks like a phone number, so that substrings of regular words ("HOTEL",
// "Telecom" etc.) never truncate the merchant name or address. The latin marker additionally
// requires a word boundary before it ("\btel" does not match the "tel" inside "Hotel")
var customGeocodingPhoneMarkerPattern = regexp.MustCompile(`(?i)\btel[\s:：.．)）]*[0-9(（+]|電話(?:番号)?[\s:：.．)）]*[0-9(（+]`)

// CustomGeocodingService represents the receipt merchant location geocoding service (fork-local feature)
type CustomGeocodingService struct {
	ServiceUsingDB
	envConfigOnce    sync.Once
	envEnabled       bool
	envProviderChain *geocoding.ProviderChain
}

// Initialize a custom geocoding service singleton instance
var (
	CustomGeocoding = &CustomGeocodingService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
	}
)

// ResolveReceiptLocation resolves the merchant location extracted from a receipt to geographic coordinates.
// It returns nil when the feature is disabled (EBK_CUSTOM_GEOCODING_ENABLE is not "true"), when no provider
// hits and on any error (errors are only logged and never returned, geocoding must not affect recognition)
func (s *CustomGeocodingService) ResolveReceiptLocation(c core.Context, merchantName string, merchantBranch string, address string) *geocoding.GeocodingResult {
	s.loadEnvConfig(c)

	if !s.envEnabled || s.envProviderChain == nil {
		return nil
	}

	normalizedName := normalizeGeocodingText(merchantName)
	normalizedBranch := normalizeGeocodingText(merchantBranch)
	normalizedAddress := normalizeGeocodingText(address)

	if normalizedName == "" && normalizedBranch == "" && normalizedAddress == "" {
		return nil
	}

	queryHash := buildGeocodingCacheKey(normalizedName, normalizedBranch, normalizedAddress)
	cache, cacheExists := s.getCachedResult(c, queryHash)

	if cacheExists {
		if cache.Hit {
			return &geocoding.GeocodingResult{
				Latitude:    cache.Latitude,
				Longitude:   cache.Longitude,
				DisplayName: cache.DisplayName,
				Confidence:  geocoding.GeocodingConfidence(cache.Confidence),
				Provider:    cache.Provider,
			}
		}

		if time.Now().Unix()-cache.CreatedUnixTime < customGeocodingNegativeCacheTTL {
			return nil
		}

		// the negative cache entry expired, re-query the providers and refresh the entry
	}

	nominatimQuery := buildNominatimGeocodingQuery(normalizedName, normalizedBranch, normalizedAddress)

	result, noHitConfirmed := s.envProviderChain.Search(c, func(providerName string) string {
		switch providerName {
		case geocoding.GEOCODING_PROVIDER_GSI:
			return normalizedAddress
		case geocoding.GEOCODING_PROVIDER_NOMINATIM:
			return nominatimQuery
		default:
			return ""
		}
	})

	if result == nil && !noHitConfirmed {
		// every tried provider failed with an error (network failure, timeout, rate limiting etc.),
		// a transient failure must not be cached as a 30-day negative entry, so the next receipt
		// from this merchant re-queries the providers (a negative entry is only written when at
		// least one provider successfully confirmed that there is no hit)
		return nil
	}

	newCache := &models.CustomGeocodingCache{
		QueryHash:       queryHash,
		Query:           truncateGeocodingCachedQuery(normalizedName + "\n" + normalizedBranch + "\n" + normalizedAddress),
		CreatedUnixTime: time.Now().Unix(),
	}

	if result != nil {
		newCache.Provider = result.Provider
		newCache.Latitude = result.Latitude
		newCache.Longitude = result.Longitude
		newCache.DisplayName = result.DisplayName
		newCache.Confidence = byte(result.Confidence)
		newCache.Hit = true
	}

	s.saveCachedResult(c, newCache, cacheExists)

	return result
}

// loadEnvConfig reads the geocoding environment configuration and builds the provider chain (read once and cached)
func (s *CustomGeocodingService) loadEnvConfig(c core.Context) {
	s.envConfigOnce.Do(func() {
		if strings.TrimSpace(os.Getenv(customGeocodingEnableEnvName)) != "true" {
			return
		}

		requestTimeout := customGeocodingDefaultRequestTimeout
		requestTimeoutValue := strings.TrimSpace(os.Getenv(customGeocodingRequestTimeoutEnvName))

		if requestTimeoutValue != "" {
			parsedTimeout, err := strconv.ParseUint(requestTimeoutValue, 10, 32)

			if err != nil || parsedTimeout < 1 {
				log.Warnf(c, "[custom_geocoding.loadEnvConfig] invalid %s value \"%s\", using default %d", customGeocodingRequestTimeoutEnvName, requestTimeoutValue, customGeocodingDefaultRequestTimeout)
			} else {
				requestTimeout = uint32(parsedTimeout)
			}
		}

		proxy := strings.TrimSpace(os.Getenv(customGeocodingProxyEnvName))

		if proxy == "" {
			proxy = customGeocodingDefaultProxy
		}

		providersValue := strings.TrimSpace(os.Getenv(customGeocodingProvidersEnvName))

		if providersValue == "" {
			providersValue = customGeocodingDefaultProviders
		}

		httpClient := httpclient.NewHttpClient(requestTimeout, proxy, false, core.GetOutgoingUserAgent(), false)
		providerNames := strings.Split(providersValue, ",")
		providers := make([]geocoding.Provider, 0, len(providerNames))

		for i := 0; i < len(providerNames); i++ {
			providerName := strings.TrimSpace(providerNames[i])

			switch providerName {
			case "":
				continue
			case geocoding.GEOCODING_PROVIDER_GSI:
				providers = append(providers, geocoding.NewGsiProvider(httpClient))
			case geocoding.GEOCODING_PROVIDER_NOMINATIM:
				providers = append(providers, geocoding.NewNominatimProvider(httpClient, geocoding.NOMINATIM_QUERY_KIND_NAME))
			default:
				log.Warnf(c, "[custom_geocoding.loadEnvConfig] unknown geocoding provider \"%s\" is ignored", providerName)
			}
		}

		if len(providers) < 1 {
			log.Warnf(c, "[custom_geocoding.loadEnvConfig] geocoding is enabled but no valid provider is configured")
			return
		}

		s.envEnabled = true
		s.envProviderChain = geocoding.NewProviderChain(providers)
	})
}

// getCachedResult returns the cached geocoding entry of the given query hash, the second return value indicates existence
func (s *CustomGeocodingService) getCachedResult(c core.Context, queryHash string) (*models.CustomGeocodingCache, bool) {
	cache := &models.CustomGeocodingCache{}
	has, err := s.UserDB().NewSession(c).Where("query_hash=?", queryHash).Get(cache)

	if err != nil {
		log.Warnf(c, "[custom_geocoding.getCachedResult] failed to get cached geocoding result, because %s", err.Error())
		return nil, false
	}

	if !has {
		return nil, false
	}

	return cache, true
}

// saveCachedResult inserts or updates the cached geocoding entry, errors are only logged
func (s *CustomGeocodingService) saveCachedResult(c core.Context, cache *models.CustomGeocodingCache, exists bool) {
	err := s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		if exists {
			_, err := sess.Where("query_hash=?", cache.QueryHash).AllCols().Update(cache)
			return err
		}

		_, err := sess.Insert(cache)
		return err
	})

	if err != nil {
		log.Warnf(c, "[custom_geocoding.saveCachedResult] failed to save cached geocoding result, because %s", err.Error())
	}
}

// normalizeGeocodingText normalizes a text extracted from a receipt for geocoding:
// full-width alphanumeric characters are converted to half-width, everything from a
// "TEL" / "電話" phone-number marker on is removed, and consecutive whitespace is collapsed
func normalizeGeocodingText(text string) string {
	converted := convertFullWidthAlphanumericToHalfWidth(text)
	converted = cutTextAtPhoneNumberMarker(converted)

	return strings.Join(strings.Fields(converted), " ")
}

// convertFullWidthAlphanumericToHalfWidth converts full-width digits and latin letters to their half-width forms
func convertFullWidthAlphanumericToHalfWidth(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))

	for _, ch := range text {
		if ('０' <= ch && ch <= '９') || ('Ａ' <= ch && ch <= 'Ｚ') || ('ａ' <= ch && ch <= 'ｚ') {
			// the full-width forms block is a contiguous offset of 0xFEE0 from the ascii block
			ch -= 0xFEE0
		}

		builder.WriteRune(ch)
	}

	return builder.String()
}

// cutTextAtPhoneNumberMarker removes everything from the first "TEL" / "電話" marker on,
// the marker is only recognized when it is followed by something that looks like a phone number
func cutTextAtPhoneNumberMarker(text string) string {
	if loc := customGeocodingPhoneMarkerPattern.FindStringIndex(text); loc != nil {
		return text[:loc[0]]
	}

	return text
}

// buildGeocodingCacheKey returns the SHA-256 hex of the normalized merchant name, branch and address
func buildGeocodingCacheKey(merchantName string, merchantBranch string, address string) string {
	hash := sha256.Sum256([]byte(merchantName + "\n" + merchantBranch + "\n" + address))
	return hex.EncodeToString(hash[:])
}

// buildNominatimGeocodingQuery builds the merchant name based nominatim query
// "{name} {branch} {city}" where the city is extracted from the address when possible,
// an empty string is returned when there is no merchant name nor branch (nominatim is skipped)
func buildNominatimGeocodingQuery(merchantName string, merchantBranch string, address string) string {
	if merchantName == "" && merchantBranch == "" {
		return ""
	}

	parts := make([]string, 0, 3)

	if merchantName != "" {
		parts = append(parts, merchantName)
	}

	if merchantBranch != "" {
		parts = append(parts, merchantBranch)
	}

	city := extractCityFromAddress(address)

	if city != "" {
		parts = append(parts, city)
	}

	return strings.Join(parts, " ")
}

// extractCityFromAddress extracts the leading prefecture + municipality part from a japanese address,
// an empty string is returned when the address does not start with a recognizable municipality
func extractCityFromAddress(address string) string {
	return customGeocodingCityPattern.FindString(address)
}

// truncateGeocodingCachedQuery truncates the cached query text to the column length
func truncateGeocodingCachedQuery(query string) string {
	runes := []rune(query)

	if len(runes) <= customGeocodingMaxCachedQueryLength {
		return query
	}

	return string(runes[:customGeocodingMaxCachedQueryLength])
}
