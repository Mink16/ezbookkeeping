package models

// CustomLlmProfile represents a server-wide llm connection profile stored in database (fork-local feature).
// Provider-specific settings of all 8 upstream llm providers are normalized into the
// {base_url, api_key, api_version, model_id, max_tokens} columns
// (see buildLLMConfigFromCustomProfile in pkg/api/custom_llm_profiles.go for the mapping)
type CustomLlmProfile struct {
	ProfileId       int64  `xorm:"PK"`
	Deleted         bool   `xorm:"INDEX(IDX_custom_llm_profile_deleted_is_active) NOT NULL"`
	Name            string `xorm:"VARCHAR(64) NOT NULL"`
	Provider        string `xorm:"VARCHAR(32) NOT NULL"`
	BaseURL         string `xorm:"VARCHAR(255) NOT NULL DEFAULT ''"`
	APIKey          string `xorm:"VARCHAR(512) NOT NULL DEFAULT ''"` // encrypted by utils.EncryptSecret, never returned to clients
	APIVersion      string `xorm:"VARCHAR(32) NOT NULL DEFAULT ''"`
	ModelID         string `xorm:"VARCHAR(128) NOT NULL"`
	MaxTokens       uint32 `xorm:"NOT NULL DEFAULT 0"` // 0 means using the upstream default (1024)
	IsActive        bool   `xorm:"INDEX(IDX_custom_llm_profile_deleted_is_active) NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// CustomLlmProfileGetRequest represents all parameters of llm profile getting request
type CustomLlmProfileGetRequest struct {
	Id int64 `form:"id,string" binding:"required,min=1"`
}

// CustomLlmProfileCreateRequest represents all parameters of llm profile creation request
type CustomLlmProfileCreateRequest struct {
	Name       string `json:"name" binding:"required,notBlank,max=64"`
	Provider   string `json:"provider" binding:"required,notBlank,max=32"`
	BaseURL    string `json:"baseUrl" binding:"max=255"`
	APIKey     string `json:"apiKey" binding:"max=256"`
	APIVersion string `json:"apiVersion" binding:"max=32"`
	ModelID    string `json:"modelId" binding:"required,notBlank,max=128"`
	MaxTokens  uint32 `json:"maxTokens"`
}

// CustomLlmProfileModifyRequest represents all parameters of llm profile modification request,
// an empty api key means keeping the stored one unchanged
type CustomLlmProfileModifyRequest struct {
	Id         int64  `json:"id,string" binding:"required,min=1"`
	Name       string `json:"name" binding:"required,notBlank,max=64"`
	Provider   string `json:"provider" binding:"required,notBlank,max=32"`
	BaseURL    string `json:"baseUrl" binding:"max=255"`
	APIKey     string `json:"apiKey" binding:"max=256"`
	APIVersion string `json:"apiVersion" binding:"max=32"`
	ModelID    string `json:"modelId" binding:"required,notBlank,max=128"`
	MaxTokens  uint32 `json:"maxTokens"`
}

// CustomLlmProfileDeleteRequest represents all parameters of llm profile deleting request
type CustomLlmProfileDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// CustomLlmProfileSetActiveRequest represents all parameters of llm profile activation request,
// id "0" means deactivating all profiles (the default environment configuration will be used)
type CustomLlmProfileSetActiveRequest struct {
	Id int64 `json:"id,string"`
}

// CustomLlmProfileTestRequest represents all parameters of llm profile connection testing request,
// an empty api key with id > 0 means testing with the stored api key of that profile
type CustomLlmProfileTestRequest struct {
	Id         int64  `json:"id,string"`
	Provider   string `json:"provider" binding:"required,notBlank,max=32"`
	BaseURL    string `json:"baseUrl" binding:"max=255"`
	APIKey     string `json:"apiKey" binding:"max=256"`
	APIVersion string `json:"apiVersion" binding:"max=32"`
	ModelID    string `json:"modelId" binding:"required,notBlank,max=128"`
	MaxTokens  uint32 `json:"maxTokens"`
}

// CustomLlmProfileInfoResponse represents a view-object of llm profile,
// it intentionally has no api key field so that the stored key can never leak to clients
type CustomLlmProfileInfoResponse struct {
	Id          int64  `json:"id,string"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	BaseURL     string `json:"baseUrl,omitempty"`
	HasAPIKey   bool   `json:"hasApiKey"`
	APIVersion  string `json:"apiVersion,omitempty"`
	ModelID     string `json:"modelId"`
	MaxTokens   uint32 `json:"maxTokens,omitempty"`
	IsActive    bool   `json:"isActive"`
	IsDefault   bool   `json:"isDefault"`
	Editable    bool   `json:"editable"`
	CreatedTime int64  `json:"createdTime,omitempty"`
	UpdatedTime int64  `json:"updatedTime,omitempty"`
}

// CustomLlmProfileListResponse represents a view-object of llm profile list,
// the first entry is always the virtual default entry built from the environment configuration
type CustomLlmProfileListResponse struct {
	Profiles []*CustomLlmProfileInfoResponse `json:"profiles"`
}

// Custom llm profile connection test result types
const (
	CustomLlmProfileTestResultOk                = "ok"
	CustomLlmProfileTestResultInvalidConfig     = "invalid_config"
	CustomLlmProfileTestResultDecryptFailed     = "decrypt_failed"
	CustomLlmProfileTestResultRequestFailed     = "request_failed"
	CustomLlmProfileTestResultVisionCheckFailed = "vision_check_failed"
)

// CustomLlmProfileTestResponse represents a view-object of llm profile connection testing result
type CustomLlmProfileTestResponse struct {
	Success        bool   `json:"success"`
	Result         string `json:"result"`
	RecognizedText string `json:"recognizedText,omitempty"`
}

// ToCustomLlmProfileInfoResponse returns a view-object according to database model
func (p *CustomLlmProfile) ToCustomLlmProfileInfoResponse() *CustomLlmProfileInfoResponse {
	return &CustomLlmProfileInfoResponse{
		Id:          p.ProfileId,
		Name:        p.Name,
		Provider:    p.Provider,
		BaseURL:     p.BaseURL,
		HasAPIKey:   p.APIKey != "",
		APIVersion:  p.APIVersion,
		ModelID:     p.ModelID,
		MaxTokens:   p.MaxTokens,
		IsActive:    p.IsActive,
		IsDefault:   false,
		Editable:    true,
		CreatedTime: p.CreatedUnixTime,
		UpdatedTime: p.UpdatedUnixTime,
	}
}
