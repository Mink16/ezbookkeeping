package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func TestValidateCustomLlmProfileFields(t *testing.T) {
	testCases := []struct {
		name        string
		provider    string
		baseURL     string
		hasAPIKey   bool
		modelID     string
		expectedErr *errs.Error
	}{
		{"openai with key and model", "openai", "", true, "gpt-4o", nil},
		{"openai without key", "openai", "", false, "gpt-4o", errs.ErrLlmProfileMissingRequiredField},
		{"openrouter with key and model", "openrouter", "", true, "some/model", nil},
		{"openrouter without key", "openrouter", "", false, "some/model", errs.ErrLlmProfileMissingRequiredField},
		{"google_ai with key and model", "google_ai", "", true, "gemini-2.0-flash", nil},
		{"google_ai without key", "google_ai", "", false, "gemini-2.0-flash", errs.ErrLlmProfileMissingRequiredField},
		{"anthropic with key and model", "anthropic", "", true, "claude-sonnet-4-6", nil},
		{"anthropic without key", "anthropic", "", false, "claude-sonnet-4-6", errs.ErrLlmProfileMissingRequiredField},
		{"openai_compatible with all", "openai_compatible", "https://api.example.com/v1/", true, "model", nil},
		{"openai_compatible without base url", "openai_compatible", "", true, "model", errs.ErrLlmProfileMissingRequiredField},
		{"openai_compatible without key", "openai_compatible", "https://api.example.com/v1/", false, "model", errs.ErrLlmProfileMissingRequiredField},
		{"anthropic_compatible with base url", "anthropic_compatible", "https://api.example.com/v1/", false, "model", nil},
		{"anthropic_compatible without base url", "anthropic_compatible", "", true, "model", errs.ErrLlmProfileMissingRequiredField},
		{"ollama with server url", "ollama", "http://127.0.0.1:11434/", false, "gemma4:latest", nil},
		{"ollama without server url", "ollama", "", false, "gemma4:latest", errs.ErrLlmProfileMissingRequiredField},
		{"lm_studio with server url without token", "lm_studio", "http://127.0.0.1:1234/", false, "model", nil},
		{"lm_studio without server url", "lm_studio", "", true, "model", errs.ErrLlmProfileMissingRequiredField},
		{"empty model id", "ollama", "http://127.0.0.1:11434/", false, "", errs.ErrLlmProfileMissingRequiredField},
		{"unknown provider", "unknown_provider", "http://127.0.0.1/", true, "model", errs.ErrLlmProfileProviderInvalid},
		{"empty provider", "", "", true, "model", errs.ErrLlmProfileProviderInvalid},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actualErr := validateCustomLlmProfileFields(testCase.provider, testCase.baseURL, testCase.hasAPIKey, testCase.modelID)
			assert.Equal(t, testCase.expectedErr, actualErr)
		})
	}
}

func TestBuildLLMConfigFromCustomProfileFieldMapping(t *testing.T) {
	testCases := []struct {
		provider    string
		assertField func(t *testing.T, llmConfig *settings.LLMConfig)
	}{
		{settings.OpenAILLMProvider, func(t *testing.T, llmConfig *settings.LLMConfig) {
			assert.Equal(t, "decrypted-key", llmConfig.OpenAIAPIKey)
			assert.Equal(t, "test-model", llmConfig.OpenAIModelID)
			assert.Empty(t, llmConfig.OllamaServerURL)
			assert.Empty(t, llmConfig.AnthropicAPIKey)
		}},
		{settings.OpenAICompatibleLLMProvider, func(t *testing.T, llmConfig *settings.LLMConfig) {
			assert.Equal(t, "https://base.example.com/", llmConfig.OpenAICompatibleBaseURL)
			assert.Equal(t, "decrypted-key", llmConfig.OpenAICompatibleAPIKey)
			assert.Equal(t, "test-model", llmConfig.OpenAICompatibleModelID)
			assert.Empty(t, llmConfig.OpenAIAPIKey)
		}},
		{settings.AnthropicLLMProvider, func(t *testing.T, llmConfig *settings.LLMConfig) {
			assert.Equal(t, "decrypted-key", llmConfig.AnthropicAPIKey)
			assert.Equal(t, "test-model", llmConfig.AnthropicModelID)
			assert.Equal(t, uint32(4096), llmConfig.AnthropicMaxTokens)
			assert.Empty(t, llmConfig.OpenAIAPIKey)
		}},
		{settings.AnthropicCompatibleLLMProvider, func(t *testing.T, llmConfig *settings.LLMConfig) {
			assert.Equal(t, "https://base.example.com/", llmConfig.AnthropicCompatibleBaseURL)
			assert.Equal(t, "2023-06-01", llmConfig.AnthropicCompatibleAPIVersion)
			assert.Equal(t, "decrypted-key", llmConfig.AnthropicCompatibleAPIKey)
			assert.Equal(t, "test-model", llmConfig.AnthropicCompatibleModelID)
			assert.Equal(t, uint32(4096), llmConfig.AnthropicCompatibleMaxTokens)
		}},
		{settings.OpenRouterLLMProvider, func(t *testing.T, llmConfig *settings.LLMConfig) {
			assert.Equal(t, "decrypted-key", llmConfig.OpenRouterAPIKey)
			assert.Equal(t, "test-model", llmConfig.OpenRouterModelID)
		}},
		{settings.OllamaLLMProvider, func(t *testing.T, llmConfig *settings.LLMConfig) {
			assert.Equal(t, "https://base.example.com/", llmConfig.OllamaServerURL)
			assert.Equal(t, "test-model", llmConfig.OllamaModelID)
			assert.Empty(t, llmConfig.LMStudioServerURL)
		}},
		{settings.LMStudioLLMProvider, func(t *testing.T, llmConfig *settings.LLMConfig) {
			assert.Equal(t, "https://base.example.com/", llmConfig.LMStudioServerURL)
			assert.Equal(t, "decrypted-key", llmConfig.LMStudioToken)
			assert.Equal(t, "test-model", llmConfig.LMStudioModelID)
		}},
		{settings.GoogleAILLMProvider, func(t *testing.T, llmConfig *settings.LLMConfig) {
			assert.Equal(t, "decrypted-key", llmConfig.GoogleAIAPIKey)
			assert.Equal(t, "test-model", llmConfig.GoogleAIModelID)
		}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.provider, func(t *testing.T) {
			profile := &models.CustomLlmProfile{
				Provider:   testCase.provider,
				BaseURL:    "https://base.example.com/",
				APIVersion: "2023-06-01",
				ModelID:    "test-model",
				MaxTokens:  4096,
			}

			llmConfig := buildLLMConfigFromCustomProfile(profile, "decrypted-key", nil)

			assert.Equal(t, testCase.provider, llmConfig.LLMProvider)
			testCase.assertField(t, llmConfig)
		})
	}
}

func TestBuildLLMConfigFromCustomProfileDefaultMaxTokens(t *testing.T) {
	profile := &models.CustomLlmProfile{
		Provider:  settings.AnthropicLLMProvider,
		ModelID:   "test-model",
		MaxTokens: 0,
	}

	llmConfig := buildLLMConfigFromCustomProfile(profile, "key", nil)

	assert.Equal(t, uint32(1024), llmConfig.AnthropicMaxTokens)
}

func TestInheritCommonLLMConfigFields(t *testing.T) {
	envConfig := &settings.LLMConfig{
		LargeLanguageModelAPIRequestTimeout: 120000,
		LargeLanguageModelAPIProxy:          "http://proxy.example.com/",
		LargeLanguageModelAPISkipTLSVerify:  true,
	}

	target := &settings.LLMConfig{}
	inheritCommonLLMConfigFields(target, envConfig)

	assert.Equal(t, uint32(120000), target.LargeLanguageModelAPIRequestTimeout)
	assert.Equal(t, "http://proxy.example.com/", target.LargeLanguageModelAPIProxy)
	assert.True(t, target.LargeLanguageModelAPISkipTLSVerify)

	target = &settings.LLMConfig{}
	inheritCommonLLMConfigFields(target, nil)

	assert.Equal(t, uint32(60000), target.LargeLanguageModelAPIRequestTimeout)
	assert.Equal(t, "system", target.LargeLanguageModelAPIProxy)
	assert.False(t, target.LargeLanguageModelAPISkipTLSVerify)
}

func TestBuildDefaultLlmProfileInfoResponse(t *testing.T) {
	resp := buildDefaultLlmProfileInfoResponse(nil, true)

	assert.Equal(t, int64(0), resp.Id)
	assert.Empty(t, resp.Provider)
	assert.True(t, resp.IsActive)
	assert.True(t, resp.IsDefault)
	assert.False(t, resp.Editable)

	envConfig := &settings.LLMConfig{
		LLMProvider:     settings.OllamaLLMProvider,
		OllamaServerURL: "http://host.docker.internal:11434/",
		OllamaModelID:   "gemma4:latest",
	}

	resp = buildDefaultLlmProfileInfoResponse(envConfig, false)

	assert.Equal(t, settings.OllamaLLMProvider, resp.Provider)
	assert.Equal(t, "http://host.docker.internal:11434/", resp.BaseURL)
	assert.Equal(t, "gemma4:latest", resp.ModelID)
	assert.False(t, resp.HasAPIKey)
	assert.False(t, resp.IsActive)
	assert.True(t, resp.IsDefault)
	assert.False(t, resp.Editable)

	envConfig = &settings.LLMConfig{
		LLMProvider:        settings.AnthropicLLMProvider,
		AnthropicAPIKey:    "sk-ant-xxx",
		AnthropicModelID:   "claude-sonnet-4-6",
		AnthropicMaxTokens: 1024,
	}

	resp = buildDefaultLlmProfileInfoResponse(envConfig, true)

	assert.Equal(t, settings.AnthropicLLMProvider, resp.Provider)
	assert.True(t, resp.HasAPIKey)
	assert.Equal(t, "claude-sonnet-4-6", resp.ModelID)
	assert.Equal(t, uint32(1024), resp.MaxTokens)
}
