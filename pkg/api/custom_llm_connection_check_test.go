package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionCheckImageIsEmbedded(t *testing.T) {
	assert.NotEmpty(t, connectionCheckImage)
	// png signature
	assert.Equal(t, []byte{0x89, 0x50, 0x4e, 0x47}, connectionCheckImage[0:4])
}

func TestEvaluateConnectionCheckResponse(t *testing.T) {
	testCases := []struct {
		name           string
		content        string
		expectedPassed bool
		expectedText   string
	}{
		{"exact match", `{"text": "EZBK-TEST 1234"}`, true, "EZBK-TEST 1234"},
		{"lower case with extra spaces", `{"text": " ezbk-test  1234 "}`, true, " ezbk-test  1234 "},
		{"token only", `{"text": "1234"}`, true, "1234"},
		{"surrounding whitespace in json", "  {\"text\": \"EZBK-TEST 1234\"}\n", true, "EZBK-TEST 1234"},
		{"wrong token", `{"text": "EZBK-TEST 5678"}`, false, "EZBK-TEST 5678"},
		{"empty text", `{"text": ""}`, false, ""},
		{"missing text field", `{"message": "EZBK-TEST 1234"}`, false, ""},
		{"not json", `EZBK-TEST 1234`, false, ""},
		{"empty content", ``, false, ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			passed, recognizedText := evaluateConnectionCheckResponse(testCase.content)
			assert.Equal(t, testCase.expectedPassed, passed)
			assert.Equal(t, testCase.expectedText, recognizedText)
		})
	}
}
