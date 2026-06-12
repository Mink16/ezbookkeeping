package geocoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGsiAddressSearchResponse_ValidResponse(t *testing.T) {
	// the coordinates are in [longitude, latitude] order in the GSI GeoJSON response
	content := `[{"geometry":{"coordinates":[139.700464,35.658034],"type":"Point"},"type":"Feature","properties":{"addressCode":"13113","title":"東京都渋谷区道玄坂二丁目"}},` +
		`{"geometry":{"coordinates":[135.0,34.0],"type":"Point"},"type":"Feature","properties":{"addressCode":"","title":"二番目の候補"}}]`

	actualResult, err := parseGsiAddressSearchResponse([]byte(content))

	assert.Nil(t, err)
	assert.NotNil(t, actualResult)
	assert.Equal(t, 35.658034, actualResult.Latitude)
	assert.Equal(t, 139.700464, actualResult.Longitude)
	assert.Equal(t, "東京都渋谷区道玄坂二丁目", actualResult.DisplayName)
	assert.Equal(t, GEOCODING_CONFIDENCE_HIGH, actualResult.Confidence)
	assert.Equal(t, GEOCODING_PROVIDER_GSI, actualResult.Provider)
}

func TestParseGsiAddressSearchResponse_EmptyResponse(t *testing.T) {
	actualResult, err := parseGsiAddressSearchResponse([]byte(`[]`))

	assert.Nil(t, err)
	assert.Nil(t, actualResult)
}

func TestParseGsiAddressSearchResponse_InvalidJson(t *testing.T) {
	actualResult, err := parseGsiAddressSearchResponse([]byte(`{ not json`))

	assert.NotNil(t, err)
	assert.Nil(t, actualResult)
}

func TestParseGsiAddressSearchResponse_MissingCoordinates(t *testing.T) {
	content := `[{"geometry":{"coordinates":[],"type":"Point"},"type":"Feature","properties":{"title":"東京都渋谷区"}}]`

	actualResult, err := parseGsiAddressSearchResponse([]byte(content))

	assert.NotNil(t, err)
	assert.Nil(t, actualResult)
}

func TestTrimLastBlockNumberSegment(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "drop last hyphen separated segment",
			address:  "東京都渋谷区道玄坂1-2-3",
			expected: "東京都渋谷区道玄坂1-2",
		},
		{
			name:     "drop last segment after chome",
			address:  "東京都渋谷区道玄坂1丁目2-3",
			expected: "東京都渋谷区道玄坂1丁目2",
		},
		{
			name:     "drop whole token when no hyphen",
			address:  "東京都渋谷区道玄坂1",
			expected: "東京都渋谷区道玄坂",
		},
		{
			name:     "full width digits and hyphens",
			address:  "道玄坂１−２−３",
			expected: "道玄坂１−２",
		},
		{
			name:     "katakana long vowel mark used as hyphen",
			address:  "道玄坂1ー2",
			expected: "道玄坂1",
		},
		{
			name:     "trailing spaces are ignored",
			address:  "道玄坂1-2-3 ",
			expected: "道玄坂1-2",
		},
		{
			name:     "block number only address keeps leading segments",
			address:  "1-2-3",
			expected: "1-2",
		},
		{
			name:     "no trailing block number",
			address:  "東京都渋谷区道玄坂",
			expected: "",
		},
		{
			name:     "single digit only",
			address:  "3",
			expected: "",
		},
		{
			name:     "empty address",
			address:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, trimLastBlockNumberSegment(tt.address))
		})
	}
}
