package api

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/llm"
	"github.com/mayswind/ezbookkeeping/pkg/llm/data"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// connectionCheckImage is a small embedded picture containing the text "EZBK-TEST 1234",
// used to verify that the configured model actually supports vision input
//
//go:embed custom_llm_connection_check_image.png
var connectionCheckImage []byte

// connectionCheckExpectedToken is the digit sequence the model must read from the embedded image,
// a text-only model cannot guess it even when it hallucinates a well-formed json response
const connectionCheckExpectedToken = "1234"

// maxConnectionCheckRequestTimeout limits the connection test duration to keep the settings page responsive,
// it matches the recognition request timeout because local models (ollama) may need tens of seconds
// to load the model on the first request
const maxConnectionCheckRequestTimeout uint32 = 60000

const connectionCheckSystemPrompt = `You are a connectivity test endpoint. ` +
	`Look at the attached image and respond with ONLY a JSON object in this exact format: ` +
	`{"text": "<the exact text you can read in the image>"}. Do not add any other text.`

type connectionCheckResponse struct {
	Text string `json:"text"`
}

// evaluateConnectionCheckResponse returns whether the llm response proves the model read the embedded image,
// together with the text recognized by the model
func evaluateConnectionCheckResponse(content string) (bool, string) {
	parsedResponse := &connectionCheckResponse{}
	err := json.Unmarshal([]byte(strings.TrimSpace(content)), parsedResponse)

	if err != nil || parsedResponse.Text == "" {
		return false, ""
	}

	normalizedText := strings.ToLower(strings.Join(strings.Fields(parsedResponse.Text), ""))

	return strings.Contains(normalizedText, connectionCheckExpectedToken), parsedResponse.Text
}

// ProfileTestHandler tests the llm connection described by the request with the embedded vision test image,
// an empty api key with id > 0 means testing with the stored api key of that profile
func (a *CustomLlmProfilesApi) ProfileTestHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	var profileTestReq models.CustomLlmProfileTestRequest
	err := c.ShouldBindJSON(&profileTestReq)

	if err != nil {
		log.Warnf(c, "[custom_llm_connection_check.ProfileTestHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	decryptedAPIKey := profileTestReq.APIKey

	if decryptedAPIKey == "" && profileTestReq.Id > 0 {
		storedProfile, err := a.llmProfiles.GetProfileById(c, profileTestReq.Id)

		if err != nil {
			log.Errorf(c, "[custom_llm_connection_check.ProfileTestHandler] failed to get profile \"id:%d\", because %s", profileTestReq.Id, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		if storedProfile.APIKey != "" {
			decryptedAPIKey, err = utils.DecryptSecret(storedProfile.APIKey, a.CurrentConfig().SecretKey)

			if err != nil {
				log.Errorf(c, "[custom_llm_connection_check.ProfileTestHandler] failed to decrypt api key of profile \"id:%d\", because %s", profileTestReq.Id, err.Error())
				return &models.CustomLlmProfileTestResponse{
					Success: false,
					Result:  models.CustomLlmProfileTestResultDecryptFailed,
				}, nil
			}
		}
	}

	if errResp := validateCustomLlmProfileFields(profileTestReq.Provider, profileTestReq.BaseURL, decryptedAPIKey != "", profileTestReq.ModelID); errResp != nil {
		return &models.CustomLlmProfileTestResponse{
			Success: false,
			Result:  models.CustomLlmProfileTestResultInvalidConfig,
		}, nil
	}

	testProfile := &models.CustomLlmProfile{
		Provider:   profileTestReq.Provider,
		BaseURL:    profileTestReq.BaseURL,
		APIVersion: profileTestReq.APIVersion,
		ModelID:    profileTestReq.ModelID,
		MaxTokens:  profileTestReq.MaxTokens,
	}

	llmConfig := buildLLMConfigFromCustomProfile(testProfile, decryptedAPIKey, a.CurrentConfig().ReceiptImageRecognitionLLMConfig)

	if llmConfig.LargeLanguageModelAPIRequestTimeout == 0 || llmConfig.LargeLanguageModelAPIRequestTimeout > maxConnectionCheckRequestTimeout {
		llmConfig.LargeLanguageModelAPIRequestTimeout = maxConnectionCheckRequestTimeout
	}

	llmProvider, err := llm.NewCustomLargeLanguageModelProvider(llmConfig, a.CurrentConfig().EnableDebugLog)

	if err != nil {
		log.Warnf(c, "[custom_llm_connection_check.ProfileTestHandler] failed to build llm provider, because %s", err.Error())
		return &models.CustomLlmProfileTestResponse{
			Success: false,
			Result:  models.CustomLlmProfileTestResultInvalidConfig,
		}, nil
	}

	llmRequest := &data.LargeLanguageModelRequest{
		Stream:                false,
		SystemPrompt:          connectionCheckSystemPrompt,
		UserPrompt:            connectionCheckImage,
		UserPromptType:        data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL,
		UserPromptContentType: "image/png",
	}

	llmResponse, err := llmProvider.GetJsonResponse(c, c.GetCurrentUid(), llmConfig, llmRequest)

	if err != nil {
		// the upstream common http provider aggregates auth, network and non-2xx errors into one error type,
		// the detailed reason is available in the server log written by the upstream provider
		log.Warnf(c, "[custom_llm_connection_check.ProfileTestHandler] llm request failed (provider \"%s\", model \"%s\"), because %s", profileTestReq.Provider, profileTestReq.ModelID, err.Error())
		return &models.CustomLlmProfileTestResponse{
			Success: false,
			Result:  models.CustomLlmProfileTestResultRequestFailed,
		}, nil
	}

	if llmResponse == nil || llmResponse.Content == "" {
		return &models.CustomLlmProfileTestResponse{
			Success: false,
			Result:  models.CustomLlmProfileTestResultVisionCheckFailed,
		}, nil
	}

	passed, recognizedText := evaluateConnectionCheckResponse(llmResponse.Content)

	if !passed {
		log.Warnf(c, "[custom_llm_connection_check.ProfileTestHandler] vision check failed (provider \"%s\", model \"%s\"), response content is \"%s\"", profileTestReq.Provider, profileTestReq.ModelID, llmResponse.Content)
		return &models.CustomLlmProfileTestResponse{
			Success:        false,
			Result:         models.CustomLlmProfileTestResultVisionCheckFailed,
			RecognizedText: recognizedText,
		}, nil
	}

	return &models.CustomLlmProfileTestResponse{
		Success:        true,
		Result:         models.CustomLlmProfileTestResultOk,
		RecognizedText: recognizedText,
	}, nil
}
