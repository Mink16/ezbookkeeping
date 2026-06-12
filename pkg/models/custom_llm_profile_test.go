package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomLlmProfileToCustomLlmProfileInfoResponse(t *testing.T) {
	profile := &CustomLlmProfile{
		ProfileId:       12345,
		Name:            "Test Profile",
		Provider:        "anthropic",
		BaseURL:         "",
		APIKey:          "encrypted-key-value",
		APIVersion:      "",
		ModelID:         "claude-sonnet-4-6",
		MaxTokens:       2048,
		IsActive:        true,
		CreatedUnixTime: 1000,
		UpdatedUnixTime: 2000,
	}

	resp := profile.ToCustomLlmProfileInfoResponse()

	assert.Equal(t, int64(12345), resp.Id)
	assert.Equal(t, "Test Profile", resp.Name)
	assert.Equal(t, "anthropic", resp.Provider)
	assert.True(t, resp.HasAPIKey)
	assert.Equal(t, "claude-sonnet-4-6", resp.ModelID)
	assert.Equal(t, uint32(2048), resp.MaxTokens)
	assert.True(t, resp.IsActive)
	assert.False(t, resp.IsDefault)
	assert.True(t, resp.Editable)
	assert.Equal(t, int64(1000), resp.CreatedTime)
	assert.Equal(t, int64(2000), resp.UpdatedTime)
}

func TestCustomLlmProfileInfoResponseNeverContainsApiKey(t *testing.T) {
	profile := &CustomLlmProfile{
		ProfileId: 1,
		Name:      "Key Leak Check",
		Provider:  "openai",
		APIKey:    "encrypted-super-secret-key",
		ModelID:   "gpt-4o",
	}

	resp := profile.ToCustomLlmProfileInfoResponse()
	serialized, err := json.Marshal(resp)

	assert.Nil(t, err)
	assert.NotContains(t, string(serialized), "encrypted-super-secret-key")
	assert.NotContains(t, string(serialized), "apiKey")
	assert.Contains(t, string(serialized), "\"hasApiKey\":true")
}

func TestCustomLlmProfileToCustomLlmProfileInfoResponseWithoutApiKey(t *testing.T) {
	profile := &CustomLlmProfile{
		ProfileId: 2,
		Name:      "Ollama Profile",
		Provider:  "ollama",
		BaseURL:   "http://127.0.0.1:11434/",
		ModelID:   "gemma4:latest",
	}

	resp := profile.ToCustomLlmProfileInfoResponse()

	assert.False(t, resp.HasAPIKey)
	assert.Equal(t, "http://127.0.0.1:11434/", resp.BaseURL)
}
