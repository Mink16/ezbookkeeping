package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/llm"
	"github.com/mayswind/ezbookkeeping/pkg/llm/data"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Default values of the common llm config fields used when the environment llm config is absent.
// NOTE: keep in sync with the upstream defaults in pkg/settings/setting.go
// (defaultLargeLanguageModelAPIRequestTimeout and loadLLMConfiguration)
const (
	defaultCustomLlmProfileRequestTimeout uint32 = 60000
	defaultCustomLlmProfileProxy          string = "system"
)

// anthropic max tokens default, same as the upstream ini default
const defaultCustomLlmProfileMaxTokens uint32 = 1024

// CustomLlmProfilesApi represents the server-wide llm connection profile api (fork-local feature)
type CustomLlmProfilesApi struct {
	ApiUsingConfig
	llmProfiles *services.CustomLlmProfileService
}

// Initialize a custom llm profile api singleton instance
var (
	CustomLlmProfiles = &CustomLlmProfilesApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		llmProfiles: services.CustomLlmProfiles,
	}
)

// ProfileListHandler returns the llm profile list with the virtual default entry built from the environment configuration
func (a *CustomLlmProfilesApi) ProfileListHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	profiles, err := a.llmProfiles.GetAllProfiles(c)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.ProfileListHandler] failed to get profiles, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	anyProfileActive := false
	profileResps := make([]*models.CustomLlmProfileInfoResponse, 0, len(profiles)+1)

	for i := 0; i < len(profiles); i++ {
		if profiles[i].IsActive {
			anyProfileActive = true
		}
	}

	profileResps = append(profileResps, buildDefaultLlmProfileInfoResponse(a.CurrentConfig().ReceiptImageRecognitionLLMConfig, !anyProfileActive))

	for i := 0; i < len(profiles); i++ {
		profileResps = append(profileResps, profiles[i].ToCustomLlmProfileInfoResponse())
	}

	return &models.CustomLlmProfileListResponse{
		Profiles: profileResps,
	}, nil
}

// ProfileGetHandler returns one specific llm profile
func (a *CustomLlmProfilesApi) ProfileGetHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	var profileGetReq models.CustomLlmProfileGetRequest
	err := c.ShouldBindQuery(&profileGetReq)

	if err != nil {
		log.Warnf(c, "[custom_llm_profiles.ProfileGetHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	profile, err := a.llmProfiles.GetProfileById(c, profileGetReq.Id)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.ProfileGetHandler] failed to get profile \"id:%d\", because %s", profileGetReq.Id, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return profile.ToCustomLlmProfileInfoResponse(), nil
}

// ProfileCreateHandler saves a new llm profile
func (a *CustomLlmProfilesApi) ProfileCreateHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	var profileCreateReq models.CustomLlmProfileCreateRequest
	err := c.ShouldBindJSON(&profileCreateReq)

	if err != nil {
		log.Warnf(c, "[custom_llm_profiles.ProfileCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	if errResp := validateCustomLlmProfileFields(profileCreateReq.Provider, profileCreateReq.BaseURL, profileCreateReq.APIKey != "", profileCreateReq.ModelID); errResp != nil {
		return nil, errResp
	}

	encryptedAPIKey := ""

	if profileCreateReq.APIKey != "" {
		encryptedAPIKey, err = utils.EncryptSecret(profileCreateReq.APIKey, a.CurrentConfig().SecretKey)

		if err != nil {
			log.Errorf(c, "[custom_llm_profiles.ProfileCreateHandler] failed to encrypt api key, because %s", err.Error())
			return nil, errs.ErrOperationFailed
		}
	}

	profile := &models.CustomLlmProfile{
		Name:       profileCreateReq.Name,
		Provider:   profileCreateReq.Provider,
		BaseURL:    profileCreateReq.BaseURL,
		APIKey:     encryptedAPIKey,
		APIVersion: profileCreateReq.APIVersion,
		ModelID:    profileCreateReq.ModelID,
		MaxTokens:  profileCreateReq.MaxTokens,
	}

	err = a.llmProfiles.CreateProfile(c, profile)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.ProfileCreateHandler] failed to create profile, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return profile.ToCustomLlmProfileInfoResponse(), nil
}

// ProfileModifyHandler updates an existed llm profile, an empty submitted api key keeps the stored one
func (a *CustomLlmProfilesApi) ProfileModifyHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	var profileModifyReq models.CustomLlmProfileModifyRequest
	err := c.ShouldBindJSON(&profileModifyReq)

	if err != nil {
		log.Warnf(c, "[custom_llm_profiles.ProfileModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	existedProfile, err := a.llmProfiles.GetProfileById(c, profileModifyReq.Id)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.ProfileModifyHandler] failed to get profile \"id:%d\", because %s", profileModifyReq.Id, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	hasAPIKey := profileModifyReq.APIKey != "" || existedProfile.APIKey != ""

	if errResp := validateCustomLlmProfileFields(profileModifyReq.Provider, profileModifyReq.BaseURL, hasAPIKey, profileModifyReq.ModelID); errResp != nil {
		return nil, errResp
	}

	updateAPIKey := profileModifyReq.APIKey != ""
	encryptedAPIKey := existedProfile.APIKey

	if updateAPIKey {
		encryptedAPIKey, err = utils.EncryptSecret(profileModifyReq.APIKey, a.CurrentConfig().SecretKey)

		if err != nil {
			log.Errorf(c, "[custom_llm_profiles.ProfileModifyHandler] failed to encrypt api key, because %s", err.Error())
			return nil, errs.ErrOperationFailed
		}
	}

	profile := &models.CustomLlmProfile{
		ProfileId:  profileModifyReq.Id,
		Name:       profileModifyReq.Name,
		Provider:   profileModifyReq.Provider,
		BaseURL:    profileModifyReq.BaseURL,
		APIKey:     encryptedAPIKey,
		APIVersion: profileModifyReq.APIVersion,
		ModelID:    profileModifyReq.ModelID,
		MaxTokens:  profileModifyReq.MaxTokens,
		IsActive:   existedProfile.IsActive,
	}

	err = a.llmProfiles.ModifyProfile(c, profile, updateAPIKey)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.ProfileModifyHandler] failed to modify profile \"id:%d\", because %s", profileModifyReq.Id, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	profile.CreatedUnixTime = existedProfile.CreatedUnixTime

	return profile.ToCustomLlmProfileInfoResponse(), nil
}

// ProfileDeleteHandler deletes an existed llm profile
func (a *CustomLlmProfilesApi) ProfileDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	var profileDeleteReq models.CustomLlmProfileDeleteRequest
	err := c.ShouldBindJSON(&profileDeleteReq)

	if err != nil {
		log.Warnf(c, "[custom_llm_profiles.ProfileDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	err = a.llmProfiles.DeleteProfile(c, profileDeleteReq.Id)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.ProfileDeleteHandler] failed to delete profile \"id:%d\", because %s", profileDeleteReq.Id, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return true, nil
}

// ProfileSetActiveHandler activates the specified llm profile, id "0" deactivates all profiles
func (a *CustomLlmProfilesApi) ProfileSetActiveHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	var setActiveReq models.CustomLlmProfileSetActiveRequest
	err := c.ShouldBindJSON(&setActiveReq)

	if err != nil {
		log.Warnf(c, "[custom_llm_profiles.ProfileSetActiveHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	err = a.llmProfiles.SetActiveProfile(c, setActiveReq.Id)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.ProfileSetActiveHandler] failed to set active profile \"id:%d\", because %s", setActiveReq.Id, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return true, nil
}

// buildDefaultLlmProfileInfoResponse returns the virtual default entry (id "0") built from the environment llm config.
// NOTE: keep in sync with the provider-specific fields of settings.LLMConfig (pkg/settings/setting.go)
// and buildLLMConfigFromCustomProfile below
func buildDefaultLlmProfileInfoResponse(envConfig *settings.LLMConfig, isActive bool) *models.CustomLlmProfileInfoResponse {
	resp := &models.CustomLlmProfileInfoResponse{
		Id:        0,
		Name:      "",
		Provider:  "",
		IsActive:  isActive,
		IsDefault: true,
		Editable:  false,
	}

	if envConfig == nil || envConfig.LLMProvider == "" {
		return resp
	}

	resp.Provider = envConfig.LLMProvider

	switch envConfig.LLMProvider {
	case settings.OpenAILLMProvider:
		resp.HasAPIKey = envConfig.OpenAIAPIKey != ""
		resp.ModelID = envConfig.OpenAIModelID
	case settings.OpenAICompatibleLLMProvider:
		resp.BaseURL = envConfig.OpenAICompatibleBaseURL
		resp.HasAPIKey = envConfig.OpenAICompatibleAPIKey != ""
		resp.ModelID = envConfig.OpenAICompatibleModelID
	case settings.AnthropicLLMProvider:
		resp.HasAPIKey = envConfig.AnthropicAPIKey != ""
		resp.ModelID = envConfig.AnthropicModelID
		resp.MaxTokens = envConfig.AnthropicMaxTokens
	case settings.AnthropicCompatibleLLMProvider:
		resp.BaseURL = envConfig.AnthropicCompatibleBaseURL
		resp.APIVersion = envConfig.AnthropicCompatibleAPIVersion
		resp.HasAPIKey = envConfig.AnthropicCompatibleAPIKey != ""
		resp.ModelID = envConfig.AnthropicCompatibleModelID
		resp.MaxTokens = envConfig.AnthropicCompatibleMaxTokens
	case settings.OpenRouterLLMProvider:
		resp.HasAPIKey = envConfig.OpenRouterAPIKey != ""
		resp.ModelID = envConfig.OpenRouterModelID
	case settings.OllamaLLMProvider:
		resp.BaseURL = envConfig.OllamaServerURL
		resp.ModelID = envConfig.OllamaModelID
	case settings.LMStudioLLMProvider:
		resp.BaseURL = envConfig.LMStudioServerURL
		resp.HasAPIKey = envConfig.LMStudioToken != ""
		resp.ModelID = envConfig.LMStudioModelID
	case settings.GoogleAILLMProvider:
		resp.HasAPIKey = envConfig.GoogleAIAPIKey != ""
		resp.ModelID = envConfig.GoogleAIModelID
	}

	return resp
}

// validateCustomLlmProfileFields validates the provider-specific required fields.
// Fields not used by the provider are ignored instead of being rejected.
// NOTE: keep in sync with the provider list in pkg/settings/setting.go
func validateCustomLlmProfileFields(providerType string, baseURL string, hasAPIKey bool, modelID string) *errs.Error {
	if modelID == "" {
		return errs.ErrLlmProfileMissingRequiredField
	}

	switch providerType {
	case settings.OpenAILLMProvider, settings.OpenRouterLLMProvider, settings.GoogleAILLMProvider, settings.AnthropicLLMProvider:
		if !hasAPIKey {
			return errs.ErrLlmProfileMissingRequiredField
		}
	case settings.OpenAICompatibleLLMProvider:
		if baseURL == "" || !hasAPIKey {
			return errs.ErrLlmProfileMissingRequiredField
		}
	case settings.AnthropicCompatibleLLMProvider, settings.OllamaLLMProvider, settings.LMStudioLLMProvider:
		if baseURL == "" {
			return errs.ErrLlmProfileMissingRequiredField
		}
	default:
		return errs.ErrLlmProfileProviderInvalid
	}

	return nil
}

// inheritCommonLLMConfigFields copies the common fields (request timeout, proxy and tls verification)
// from the environment llm config, or applies the upstream defaults when the environment config is absent
func inheritCommonLLMConfigFields(target *settings.LLMConfig, envLLMConfig *settings.LLMConfig) {
	if envLLMConfig != nil {
		target.LargeLanguageModelAPIRequestTimeout = envLLMConfig.LargeLanguageModelAPIRequestTimeout
		target.LargeLanguageModelAPIProxy = envLLMConfig.LargeLanguageModelAPIProxy
		target.LargeLanguageModelAPISkipTLSVerify = envLLMConfig.LargeLanguageModelAPISkipTLSVerify
	} else {
		target.LargeLanguageModelAPIRequestTimeout = defaultCustomLlmProfileRequestTimeout
		target.LargeLanguageModelAPIProxy = defaultCustomLlmProfileProxy
		target.LargeLanguageModelAPISkipTLSVerify = false
	}
}

// buildLLMConfigFromCustomProfile returns a settings.LLMConfig built from the given llm profile.
// NOTE: keep in sync with the provider-specific fields of settings.LLMConfig (pkg/settings/setting.go)
// and buildDefaultLlmProfileInfoResponse above
func buildLLMConfigFromCustomProfile(profile *models.CustomLlmProfile, decryptedAPIKey string, envLLMConfig *settings.LLMConfig) *settings.LLMConfig {
	llmConfig := &settings.LLMConfig{
		LLMProvider: profile.Provider,
	}

	maxTokens := profile.MaxTokens

	if maxTokens == 0 {
		maxTokens = defaultCustomLlmProfileMaxTokens
	}

	switch profile.Provider {
	case settings.OpenAILLMProvider:
		llmConfig.OpenAIAPIKey = decryptedAPIKey
		llmConfig.OpenAIModelID = profile.ModelID
	case settings.OpenAICompatibleLLMProvider:
		llmConfig.OpenAICompatibleBaseURL = profile.BaseURL
		llmConfig.OpenAICompatibleAPIKey = decryptedAPIKey
		llmConfig.OpenAICompatibleModelID = profile.ModelID
	case settings.AnthropicLLMProvider:
		llmConfig.AnthropicAPIKey = decryptedAPIKey
		llmConfig.AnthropicModelID = profile.ModelID
		llmConfig.AnthropicMaxTokens = maxTokens
	case settings.AnthropicCompatibleLLMProvider:
		llmConfig.AnthropicCompatibleBaseURL = profile.BaseURL
		llmConfig.AnthropicCompatibleAPIVersion = profile.APIVersion
		llmConfig.AnthropicCompatibleAPIKey = decryptedAPIKey
		llmConfig.AnthropicCompatibleModelID = profile.ModelID
		llmConfig.AnthropicCompatibleMaxTokens = maxTokens
	case settings.OpenRouterLLMProvider:
		llmConfig.OpenRouterAPIKey = decryptedAPIKey
		llmConfig.OpenRouterModelID = profile.ModelID
	case settings.OllamaLLMProvider:
		llmConfig.OllamaServerURL = profile.BaseURL
		llmConfig.OllamaModelID = profile.ModelID
	case settings.LMStudioLLMProvider:
		llmConfig.LMStudioServerURL = profile.BaseURL
		llmConfig.LMStudioToken = decryptedAPIKey
		llmConfig.LMStudioModelID = profile.ModelID
	case settings.GoogleAILLMProvider:
		llmConfig.GoogleAIAPIKey = decryptedAPIKey
		llmConfig.GoogleAIModelID = profile.ModelID
	}

	inheritCommonLLMConfigFields(llmConfig, envLLMConfig)

	return llmConfig
}

// getCustomReceiptRecognitionJsonResponse returns the llm response for receipt recognition.
// When an active llm profile exists in database it is used to build the provider dynamically,
// otherwise the request is delegated to the startup-initialized provider built from the environment config.
// A failing active profile never falls back to the environment config so that the used llm is always explicit.
// This function is the fork-local hook called from RecognizeReceiptImageHandler (large_language_models.go)
func getCustomReceiptRecognitionJsonResponse(c *core.WebContext, uid int64, config *settings.Config, request *data.LargeLanguageModelRequest) (*data.LargeLanguageModelTextualResponse, error) {
	activeProfile, err := services.CustomLlmProfiles.GetActiveProfile(c)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.getCustomReceiptRecognitionJsonResponse] failed to get active llm profile for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	if activeProfile == nil {
		return llm.Container.GetJsonResponseByReceiptImageRecognitionModel(c, uid, config, request)
	}

	decryptedAPIKey := ""

	if activeProfile.APIKey != "" {
		decryptedAPIKey, err = utils.DecryptSecret(activeProfile.APIKey, config.SecretKey)

		if err != nil {
			log.Errorf(c, "[custom_llm_profiles.getCustomReceiptRecognitionJsonResponse] failed to decrypt api key of llm profile \"id:%d\" for user \"uid:%d\", because %s", activeProfile.ProfileId, uid, err.Error())
			return nil, errs.ErrLlmProfileApiKeyDecryptFailed
		}
	}

	llmConfig := buildLLMConfigFromCustomProfile(activeProfile, decryptedAPIKey, config.ReceiptImageRecognitionLLMConfig)
	llmProvider, err := llm.NewCustomLargeLanguageModelProvider(llmConfig, config.EnableDebugLog)

	if err != nil {
		log.Errorf(c, "[custom_llm_profiles.getCustomReceiptRecognitionJsonResponse] failed to build llm provider from profile \"id:%d\" for user \"uid:%d\", because %s", activeProfile.ProfileId, uid, err.Error())
		return nil, err
	}

	log.Infof(c, "[custom_llm_profiles.getCustomReceiptRecognitionJsonResponse] using llm profile \"id:%d\" (provider \"%s\", model \"%s\") for user \"uid:%d\"", activeProfile.ProfileId, activeProfile.Provider, activeProfile.ModelID, uid)

	return llmProvider.GetJsonResponse(c, uid, llmConfig, request)
}
