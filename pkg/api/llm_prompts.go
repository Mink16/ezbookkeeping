package api

import (
	"bytes"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/templates"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const maxLlmPromptContentSize = 16384

// LlmPromptsApi represents user-defined llm prompt api
type LlmPromptsApi struct {
	llmPrompts            *services.LlmPromptService
	transactionCategories *services.TransactionCategoryService
	transactionTags       *services.TransactionTagService
	accounts              *services.AccountService
}

// Initialize a llm prompt api singleton instance
var (
	LlmPrompts = &LlmPromptsApi{
		llmPrompts:            services.LlmPrompts,
		transactionCategories: services.TransactionCategories,
		transactionTags:       services.TransactionTags,
		accounts:              services.Accounts,
	}
)

// LlmPromptListHandler returns llm prompt list of current user with the default prompt content
func (a *LlmPromptsApi) LlmPromptListHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	prompts, err := a.llmPrompts.GetAllPromptsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[llm_prompts.LlmPromptListHandler] failed to get prompts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	defaultContent, err := templates.GetTemplateRawContent(templates.SYSTEM_PROMPT_RECEIPT_IMAGE_RECOGNITION)

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptListHandler] failed to get default prompt template content for user \"uid:%d\", because %s", uid, err.Error())
		defaultContent = ""
	}

	promptResps := make([]*models.LlmPromptInfoResponse, len(prompts))

	for i := 0; i < len(prompts); i++ {
		promptResps[i] = prompts[i].ToLlmPromptInfoResponse(false)
	}

	return &models.LlmPromptListResponse{
		DefaultContent: defaultContent,
		Prompts:        promptResps,
	}, nil
}

// LlmPromptGetHandler returns one specific llm prompt of current user
func (a *LlmPromptsApi) LlmPromptGetHandler(c *core.WebContext) (any, *errs.Error) {
	var promptGetReq models.LlmPromptGetRequest
	err := c.ShouldBindQuery(&promptGetReq)

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptGetHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	prompt, err := a.llmPrompts.GetPromptByPromptId(c, uid, promptGetReq.Id)

	if err != nil {
		log.Errorf(c, "[llm_prompts.LlmPromptGetHandler] failed to get prompt \"id:%d\" for user \"uid:%d\", because %s", promptGetReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return prompt.ToLlmPromptInfoResponse(true), nil
}

// LlmPromptCreateHandler saves a new llm prompt by request parameters for current user
func (a *LlmPromptsApi) LlmPromptCreateHandler(c *core.WebContext) (any, *errs.Error) {
	var promptCreateReq models.LlmPromptCreateRequest
	err := c.ShouldBindJSON(&promptCreateReq)

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if errResp := validateLlmPromptContent(c, uid, promptCreateReq.Content); errResp != nil {
		return nil, errResp
	}

	prompt := &models.LlmPrompt{
		Uid:     uid,
		Name:    promptCreateReq.Name,
		Content: promptCreateReq.Content,
	}

	err = a.llmPrompts.CreatePrompt(c, prompt)

	if err != nil {
		log.Errorf(c, "[llm_prompts.LlmPromptCreateHandler] failed to create prompt for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[llm_prompts.LlmPromptCreateHandler] user \"uid:%d\" has created a new prompt \"id:%d\" successfully", uid, prompt.PromptId)

	return prompt.ToLlmPromptInfoResponse(true), nil
}

// LlmPromptModifyHandler saves an existed llm prompt by request parameters for current user
func (a *LlmPromptsApi) LlmPromptModifyHandler(c *core.WebContext) (any, *errs.Error) {
	var promptModifyReq models.LlmPromptModifyRequest
	err := c.ShouldBindJSON(&promptModifyReq)

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if errResp := validateLlmPromptContent(c, uid, promptModifyReq.Content); errResp != nil {
		return nil, errResp
	}

	prompt, err := a.llmPrompts.GetPromptByPromptId(c, uid, promptModifyReq.Id)

	if err != nil {
		log.Errorf(c, "[llm_prompts.LlmPromptModifyHandler] failed to get prompt \"id:%d\" for user \"uid:%d\", because %s", promptModifyReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if prompt.Name == promptModifyReq.Name && prompt.Content == promptModifyReq.Content {
		return nil, errs.ErrNothingWillBeUpdated
	}

	newPrompt := &models.LlmPrompt{
		PromptId: prompt.PromptId,
		Uid:      uid,
		Name:     promptModifyReq.Name,
		Content:  promptModifyReq.Content,
	}

	err = a.llmPrompts.ModifyPrompt(c, newPrompt)

	if err != nil {
		log.Errorf(c, "[llm_prompts.LlmPromptModifyHandler] failed to update prompt \"id:%d\" for user \"uid:%d\", because %s", promptModifyReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[llm_prompts.LlmPromptModifyHandler] user \"uid:%d\" has updated prompt \"id:%d\" successfully", uid, promptModifyReq.Id)

	prompt.Name = newPrompt.Name
	prompt.Content = newPrompt.Content
	prompt.UpdatedUnixTime = newPrompt.UpdatedUnixTime

	return prompt.ToLlmPromptInfoResponse(true), nil
}

// LlmPromptDeleteHandler deletes an existed llm prompt by request parameters for current user
func (a *LlmPromptsApi) LlmPromptDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var promptDeleteReq models.LlmPromptDeleteRequest
	err := c.ShouldBindJSON(&promptDeleteReq)

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.llmPrompts.DeletePrompt(c, uid, promptDeleteReq.Id)

	if err != nil {
		log.Errorf(c, "[llm_prompts.LlmPromptDeleteHandler] failed to delete prompt \"id:%d\" for user \"uid:%d\", because %s", promptDeleteReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[llm_prompts.LlmPromptDeleteHandler] user \"uid:%d\" has deleted prompt \"id:%d\"", uid, promptDeleteReq.Id)

	return true, nil
}

// LlmPromptSetActiveHandler activates the specified llm prompt for current user, id "0" means using the default prompt
func (a *LlmPromptsApi) LlmPromptSetActiveHandler(c *core.WebContext) (any, *errs.Error) {
	var promptSetActiveReq models.LlmPromptSetActiveRequest
	err := c.ShouldBindJSON(&promptSetActiveReq)

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptSetActiveHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if promptSetActiveReq.Id < 0 {
		return nil, errs.ErrLlmPromptIdInvalid
	}

	err = a.llmPrompts.SetActivePrompt(c, uid, promptSetActiveReq.Id)

	if err != nil {
		log.Errorf(c, "[llm_prompts.LlmPromptSetActiveHandler] failed to set active prompt \"id:%d\" for user \"uid:%d\", because %s", promptSetActiveReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[llm_prompts.LlmPromptSetActiveHandler] user \"uid:%d\" has set active prompt \"id:%d\"", uid, promptSetActiveReq.Id)

	return true, nil
}

// LlmPromptPreviewHandler returns the rendered prompt content with the real data of current user
func (a *LlmPromptsApi) LlmPromptPreviewHandler(c *core.WebContext) (any, *errs.Error) {
	var promptPreviewReq models.LlmPromptPreviewRequest
	err := c.ShouldBindJSON(&promptPreviewReq)

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptPreviewHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if errResp := validateLlmPromptContent(c, uid, promptPreviewReq.Content); errResp != nil {
		return nil, errResp
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptPreviewHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	systemPromptParams, errResp := a.buildReceiptSystemPromptParams(c, uid, clientTimezone)

	if errResp != nil {
		return nil, errResp
	}

	renderedContent, err := templates.RenderUserPromptTemplate(promptPreviewReq.Content, systemPromptParams)

	if err != nil {
		log.Warnf(c, "[llm_prompts.LlmPromptPreviewHandler] failed to render prompt content for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrLlmPromptContentInvalid
	}

	return &models.LlmPromptPreviewResponse{
		RenderedContent: renderedContent,
	}, nil
}

// buildReceiptSystemPromptParams returns the template parameters for the receipt image recognition system prompt
// NOTE: keep in sync with RecognizeReceiptImageHandler systemPromptParams (large_language_models.go)
func (a *LlmPromptsApi) buildReceiptSystemPromptParams(c *core.WebContext, uid int64, clientTimezone *time.Location) (map[string]any, *errs.Error) {
	accounts, err := a.accounts.GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[llm_prompts.buildReceiptSystemPromptParams] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	categories, err := a.transactionCategories.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[llm_prompts.buildReceiptSystemPromptParams] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	tags, err := a.transactionTags.GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[llm_prompts.buildReceiptSystemPromptParams] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	expenseCategoryNames, incomeCategoryNames, transferCategoryNames := buildCategoryNamesByType(categories)

	return map[string]any{
		"CurrentDateTime":          utils.FormatUnixTimeToLongDateTime(time.Now().Unix(), clientTimezone),
		"AllExpenseCategoryNames":  strings.Join(expenseCategoryNames, "\n"),
		"AllIncomeCategoryNames":   strings.Join(incomeCategoryNames, "\n"),
		"AllTransferCategoryNames": strings.Join(transferCategoryNames, "\n"),
		"AllAccountNames":          strings.Join(buildVisibleAccountNames(accounts), "\n"),
		"AllTagNames":              strings.Join(buildVisibleTagNames(tags), "\n"),
	}, nil
}

// applyCustomReceiptSystemPrompt overwrites the rendered system prompt in bodyBuffer with the active
// user-defined prompt of the user if it exists, any failure keeps the default prompt untouched
func applyCustomReceiptSystemPrompt(c *core.WebContext, uid int64, systemPromptParams map[string]any, bodyBuffer *bytes.Buffer) {
	activePrompt, err := services.LlmPrompts.GetActivePromptByUid(c, uid)

	if err != nil {
		log.Warnf(c, "[llm_prompts.applyCustomReceiptSystemPrompt] failed to get active prompt for user \"uid:%d\", fallback to default prompt, because %s", uid, err.Error())
		return
	}

	if activePrompt == nil {
		return
	}

	renderedContent, err := templates.RenderUserPromptTemplate(activePrompt.Content, systemPromptParams)

	if err != nil {
		log.Warnf(c, "[llm_prompts.applyCustomReceiptSystemPrompt] failed to render active prompt \"id:%d\" for user \"uid:%d\", fallback to default prompt, because %s", activePrompt.PromptId, uid, err.Error())
		return
	}

	bodyBuffer.Reset()
	bodyBuffer.WriteString(renderedContent)
}

// validateLlmPromptContent returns an error when the prompt content exceeds the maximum size or is not a valid template
func validateLlmPromptContent(c *core.WebContext, uid int64, content string) *errs.Error {
	if len(content) > maxLlmPromptContentSize {
		log.Warnf(c, "[llm_prompts.validateLlmPromptContent] the prompt content size \"%d\" exceeds the maximum size \"%d\" for user \"uid:%d\"", len(content), maxLlmPromptContentSize, uid)
		return errs.ErrLlmPromptContentTooLarge
	}

	if _, err := templates.ParseUserPromptTemplate(content); err != nil {
		log.Warnf(c, "[llm_prompts.validateLlmPromptContent] the prompt content is not a valid template for user \"uid:%d\", because %s", uid, err.Error())
		return errs.ErrLlmPromptContentInvalid
	}

	return nil
}

// buildVisibleAccountNames returns the names of visible accounts
// NOTE: keep in sync with RecognizeReceiptImageHandler (large_language_models.go)
func buildVisibleAccountNames(accounts []*models.Account) []string {
	accountNames := make([]string, 0, len(accounts))

	for i := 0; i < len(accounts); i++ {
		if accounts[i].Hidden || accounts[i].Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			continue
		}

		accountNames = append(accountNames, accounts[i].Name)
	}

	return accountNames
}

// buildCategoryNamesByType returns the names of visible secondary categories grouped by category type
// NOTE: keep in sync with RecognizeReceiptImageHandler (large_language_models.go)
func buildCategoryNamesByType(categories []*models.TransactionCategory) (expenseCategoryNames []string, incomeCategoryNames []string, transferCategoryNames []string) {
	expenseCategoryNames = make([]string, 0)
	incomeCategoryNames = make([]string, 0)
	transferCategoryNames = make([]string, 0)

	for i := 0; i < len(categories); i++ {
		category := categories[i]

		if category.Hidden || category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
			continue
		}

		if category.Type == models.CATEGORY_TYPE_INCOME {
			incomeCategoryNames = append(incomeCategoryNames, category.Name)
		} else if category.Type == models.CATEGORY_TYPE_EXPENSE {
			expenseCategoryNames = append(expenseCategoryNames, category.Name)
		} else if category.Type == models.CATEGORY_TYPE_TRANSFER {
			transferCategoryNames = append(transferCategoryNames, category.Name)
		}
	}

	return expenseCategoryNames, incomeCategoryNames, transferCategoryNames
}

// buildVisibleTagNames returns the names of visible tags
// NOTE: keep in sync with RecognizeReceiptImageHandler (large_language_models.go)
func buildVisibleTagNames(tags []*models.TransactionTag) []string {
	tagNames := make([]string, 0, len(tags))

	for i := 0; i < len(tags); i++ {
		if tags[i].Hidden {
			continue
		}

		tagNames = append(tagNames, tags[i].Name)
	}

	return tagNames
}
