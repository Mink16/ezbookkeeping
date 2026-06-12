package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
)

func TestNormalizeGeocodingText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "trim and collapse consecutive whitespace",
			text:     "  セブン-イレブン　　渋谷店  ",
			expected: "セブン-イレブン 渋谷店",
		},
		{
			name:     "convert full width alphanumeric to half width",
			text:     "ＡＢＣマート １２３号店",
			expected: "ABCマート 123号店",
		},
		{
			name:     "full width hyphen is kept as is",
			text:     "１−２−３",
			expected: "1−2−3",
		},
		{
			name:     "remove from TEL marker on",
			text:     "東京都渋谷区道玄坂2-10 TEL:03-1234-5678",
			expected: "東京都渋谷区道玄坂2-10",
		},
		{
			name:     "remove from full width TEL marker on",
			text:     "東京都渋谷区道玄坂２−１０ ＴＥＬ０３−１２３４−５６７８",
			expected: "東京都渋谷区道玄坂2−10",
		},
		{
			name:     "remove from phone marker on",
			text:     "東京都渋谷区道玄坂2-10 電話 03-1234-5678",
			expected: "東京都渋谷区道玄坂2-10",
		},
		{
			name:     "lower case tel marker",
			text:     "渋谷店 tel 03-1234-5678",
			expected: "渋谷店",
		},
		{
			name:     "tel marker directly followed by digits",
			text:     "渋谷店TEL03-1234-5678",
			expected: "渋谷店",
		},
		{
			name:     "tel marker with full width colon",
			text:     "渋谷店 TEL：03-1234-5678",
			expected: "渋谷店",
		},
		{
			name:     "phone number marker with 番号 suffix",
			text:     "東京都渋谷区道玄坂2-10 電話番号:03-1234-5678",
			expected: "東京都渋谷区道玄坂2-10",
		},
		{
			name:     "hotel is not a phone number marker",
			text:     "APA HOTEL 新宿",
			expected: "APA HOTEL 新宿",
		},
		{
			name:     "hotel followed by a number is not a phone number marker",
			text:     "Hotel 1899 Tokyo",
			expected: "Hotel 1899 Tokyo",
		},
		{
			name:     "telecom is not a phone number marker",
			text:     "Telecom Center Building",
			expected: "Telecom Center Building",
		},
		{
			name:     "tel marker without a following number is kept",
			text:     "渋谷店 TEL",
			expected: "渋谷店 TEL",
		},
		{
			name:     "empty text",
			text:     "",
			expected: "",
		},
		{
			name:     "whitespace only",
			text:     " 　 ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeGeocodingText(tt.text))
		})
	}
}

func TestBuildGeocodingCacheKey(t *testing.T) {
	key := buildGeocodingCacheKey("セブン-イレブン", "渋谷駅前店", "東京都渋谷区道玄坂2-10")

	assert.Len(t, key, 64)
	assert.Equal(t, key, buildGeocodingCacheKey("セブン-イレブン", "渋谷駅前店", "東京都渋谷区道玄坂2-10"))

	// the separator must prevent collisions between different field boundaries
	assert.NotEqual(t, buildGeocodingCacheKey("ab", "c", ""), buildGeocodingCacheKey("a", "bc", ""))
	assert.NotEqual(t, key, buildGeocodingCacheKey("セブン-イレブン", "渋谷駅前店", ""))
}

func TestBuildNominatimGeocodingQuery(t *testing.T) {
	tests := []struct {
		name           string
		merchantName   string
		merchantBranch string
		address        string
		expected       string
	}{
		{
			name:           "name branch and extractable city",
			merchantName:   "セブン-イレブン",
			merchantBranch: "渋谷駅前店",
			address:        "東京都渋谷区道玄坂2-10",
			expected:       "セブン-イレブン 渋谷駅前店 東京都渋谷区",
		},
		{
			name:           "name only",
			merchantName:   "セブン-イレブン",
			merchantBranch: "",
			address:        "",
			expected:       "セブン-イレブン",
		},
		{
			name:           "branch only",
			merchantName:   "",
			merchantBranch: "渋谷駅前店",
			address:        "",
			expected:       "渋谷駅前店",
		},
		{
			name:           "city not extractable from address",
			merchantName:   "セブン-イレブン",
			merchantBranch: "渋谷駅前店",
			address:        "1-2-3",
			expected:       "セブン-イレブン 渋谷駅前店",
		},
		{
			name:           "no name and no branch skips nominatim",
			merchantName:   "",
			merchantBranch: "",
			address:        "東京都渋谷区道玄坂2-10",
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildNominatimGeocodingQuery(tt.merchantName, tt.merchantBranch, tt.address))
		})
	}
}

func TestExtractCityFromAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "prefecture and ward",
			address:  "東京都渋谷区道玄坂2-10",
			expected: "東京都渋谷区",
		},
		{
			name:     "prefecture and city",
			address:  "北海道札幌市中央区北1条西2丁目",
			expected: "北海道札幌市",
		},
		{
			name:     "three character prefecture",
			address:  "神奈川県横浜市西区みなとみらい1-1",
			expected: "神奈川県横浜市",
		},
		{
			name:     "city without prefecture",
			address:  "札幌市北区北10条西5丁目",
			expected: "札幌市",
		},
		{
			name:     "kyoto city without prefecture",
			address:  "京都市下京区烏丸通",
			expected: "京都市",
		},
		{
			name:     "no municipality",
			address:  "1-2-3",
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
			assert.Equal(t, tt.expected, extractCityFromAddress(tt.address))
		})
	}
}

func TestTruncateGeocodingCachedQuery(t *testing.T) {
	shortQuery := "セブン-イレブン\n渋谷駅前店\n東京都渋谷区道玄坂2-10"
	assert.Equal(t, shortQuery, truncateGeocodingCachedQuery(shortQuery))

	longQueryRunes := make([]rune, 300)

	for i := 0; i < len(longQueryRunes); i++ {
		longQueryRunes[i] = 'あ'
	}

	truncated := truncateGeocodingCachedQuery(string(longQueryRunes))
	assert.Equal(t, customGeocodingMaxCachedQueryLength, len([]rune(truncated)))
}

func TestCustomGeocodingServiceLoadEnvConfig_DisabledWithoutEnv(t *testing.T) {
	t.Setenv(customGeocodingEnableEnvName, "")

	service := &CustomGeocodingService{}
	service.loadEnvConfig(core.NewNullContext())

	assert.False(t, service.envEnabled)
	assert.Nil(t, service.envProviderChain)
}

func TestCustomGeocodingServiceLoadEnvConfig_EnabledWithDefaultProviders(t *testing.T) {
	t.Setenv(customGeocodingEnableEnvName, "true")
	t.Setenv(customGeocodingProvidersEnvName, "")
	t.Setenv(customGeocodingRequestTimeoutEnvName, "")
	t.Setenv(customGeocodingProxyEnvName, "")

	service := &CustomGeocodingService{}
	service.loadEnvConfig(core.NewNullContext())

	assert.True(t, service.envEnabled)
	assert.NotNil(t, service.envProviderChain)
}

func TestCustomGeocodingServiceLoadEnvConfig_EnabledWithoutValidProvider(t *testing.T) {
	t.Setenv(customGeocodingEnableEnvName, "true")
	t.Setenv(customGeocodingProvidersEnvName, "unknown-provider")

	service := &CustomGeocodingService{}
	service.loadEnvConfig(core.NewNullContext())

	assert.False(t, service.envEnabled)
	assert.Nil(t, service.envProviderChain)
}
