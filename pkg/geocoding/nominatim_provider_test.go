package geocoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNominatimSearchResponse_ValidResponse(t *testing.T) {
	content := `[{"place_id":12345,"licence":"Data © OpenStreetMap contributors","lat":"35.6580339","lon":"139.7016358",` +
		`"category":"railway","type":"station","display_name":"渋谷駅, 道玄坂, 渋谷区, 東京都, 日本"}]`

	actualResult, err := parseNominatimSearchResponse([]byte(content), GEOCODING_CONFIDENCE_LOW)

	assert.Nil(t, err)
	assert.NotNil(t, actualResult)
	assert.Equal(t, 35.6580339, actualResult.Latitude)
	assert.Equal(t, 139.7016358, actualResult.Longitude)
	assert.Equal(t, "渋谷駅, 道玄坂, 渋谷区, 東京都, 日本", actualResult.DisplayName)
	assert.Equal(t, GEOCODING_CONFIDENCE_LOW, actualResult.Confidence)
	assert.Equal(t, GEOCODING_PROVIDER_NOMINATIM, actualResult.Provider)
}

func TestParseNominatimSearchResponse_GivenConfidenceIsAdopted(t *testing.T) {
	content := `[{"lat":"35.0","lon":"139.0","display_name":"東京都渋谷区道玄坂"}]`

	actualResult, err := parseNominatimSearchResponse([]byte(content), GEOCODING_CONFIDENCE_HIGH)

	assert.Nil(t, err)
	assert.NotNil(t, actualResult)
	assert.Equal(t, GEOCODING_CONFIDENCE_HIGH, actualResult.Confidence)
}

func TestParseNominatimSearchResponse_EmptyResponse(t *testing.T) {
	actualResult, err := parseNominatimSearchResponse([]byte(`[]`), GEOCODING_CONFIDENCE_LOW)

	assert.Nil(t, err)
	assert.Nil(t, actualResult)
}

func TestParseNominatimSearchResponse_InvalidJson(t *testing.T) {
	actualResult, err := parseNominatimSearchResponse([]byte(`<html>error</html>`), GEOCODING_CONFIDENCE_LOW)

	assert.NotNil(t, err)
	assert.Nil(t, actualResult)
}

func TestParseNominatimSearchResponse_InvalidLatitude(t *testing.T) {
	content := `[{"lat":"not a number","lon":"139.0","display_name":"東京都"}]`

	actualResult, err := parseNominatimSearchResponse([]byte(content), GEOCODING_CONFIDENCE_LOW)

	assert.NotNil(t, err)
	assert.Nil(t, actualResult)
}

func TestNominatimQueryKindConfidence(t *testing.T) {
	assert.Equal(t, GEOCODING_CONFIDENCE_LOW, nominatimQueryKindConfidence(NOMINATIM_QUERY_KIND_NAME))
	assert.Equal(t, GEOCODING_CONFIDENCE_HIGH, nominatimQueryKindConfidence(NOMINATIM_QUERY_KIND_ADDRESS))
}
