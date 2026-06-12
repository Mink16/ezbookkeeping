package models

// LlmPrompt represents user-defined llm system prompt data stored in database
type LlmPrompt struct {
	PromptId        int64  `xorm:"PK"`
	Uid             int64  `xorm:"INDEX(IDX_llm_prompt_uid_deleted) NOT NULL"`
	Deleted         bool   `xorm:"INDEX(IDX_llm_prompt_uid_deleted) NOT NULL"`
	Name            string `xorm:"VARCHAR(64) NOT NULL"`
	Content         string `xorm:"TEXT NOT NULL"`
	IsActive        bool   `xorm:"NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// LlmPromptGetRequest represents all parameters of llm prompt getting request
type LlmPromptGetRequest struct {
	Id int64 `form:"id,string" binding:"required,min=1"`
}

// LlmPromptCreateRequest represents all parameters of llm prompt creation request
type LlmPromptCreateRequest struct {
	Name    string `json:"name" binding:"required,notBlank,max=64"`
	Content string `json:"content" binding:"required,notBlank"`
}

// LlmPromptModifyRequest represents all parameters of llm prompt modification request
type LlmPromptModifyRequest struct {
	Id      int64  `json:"id,string" binding:"required,min=1"`
	Name    string `json:"name" binding:"required,notBlank,max=64"`
	Content string `json:"content" binding:"required,notBlank"`
}

// LlmPromptDeleteRequest represents all parameters of llm prompt deleting request
type LlmPromptDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// LlmPromptSetActiveRequest represents all parameters of llm prompt activation request,
// id "0" means deactivating all prompts (use the default prompt)
type LlmPromptSetActiveRequest struct {
	Id int64 `json:"id,string"`
}

// LlmPromptPreviewRequest represents all parameters of llm prompt previewing request
type LlmPromptPreviewRequest struct {
	Content string `json:"content" binding:"required,notBlank"`
}

// LlmPromptInfoResponse represents a view-object of llm prompt
type LlmPromptInfoResponse struct {
	Id          int64  `json:"id,string"`
	Name        string `json:"name"`
	Content     string `json:"content,omitempty"`
	IsActive    bool   `json:"isActive"`
	CreatedTime int64  `json:"createdTime"`
	UpdatedTime int64  `json:"updatedTime"`
}

// LlmPromptListResponse represents a view-object of llm prompt list with the default prompt content
type LlmPromptListResponse struct {
	DefaultContent string                   `json:"defaultContent"`
	Prompts        []*LlmPromptInfoResponse `json:"prompts"`
}

// LlmPromptPreviewResponse represents a view-object of llm prompt previewing result
type LlmPromptPreviewResponse struct {
	RenderedContent string `json:"renderedContent"`
}

// ToLlmPromptInfoResponse returns a view-object according to database model
func (p *LlmPrompt) ToLlmPromptInfoResponse(includeContent bool) *LlmPromptInfoResponse {
	content := ""

	if includeContent {
		content = p.Content
	}

	return &LlmPromptInfoResponse{
		Id:          p.PromptId,
		Name:        p.Name,
		Content:     content,
		IsActive:    p.IsActive,
		CreatedTime: p.CreatedUnixTime,
		UpdatedTime: p.UpdatedUnixTime,
	}
}
