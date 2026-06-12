package api

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/geocoding"
	"github.com/mayswind/ezbookkeeping/pkg/llm/data"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/templates"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// CustomReceiptRecognitionApi represents the fork-local receipt recognition api with item details and geocoding.
// It coexists with the upstream RecognizeReceiptImageHandler (large_language_models.go) and follows the same
// flow, so upstream changes to that handler (validation, prompt parameters etc.) should be mirrored here
type CustomReceiptRecognitionApi struct {
	ApiUsingConfig
	transactionCategories *services.TransactionCategoryService
	transactionTags       *services.TransactionTagService
	accounts              *services.AccountService
	users                 *services.UserService
}

// Initialize a custom receipt recognition api singleton instance
var (
	CustomReceiptRecognition = &CustomReceiptRecognitionApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		transactionCategories: services.TransactionCategories,
		transactionTags:       services.TransactionTags,
		accounts:              services.Accounts,
		users:                 services.Users,
	}
)

// RecognizeReceiptImageDetailsHandler returns the recognized receipt image result with item details,
// the reconciled item list (sum invariant) and the geocoded merchant location
func (a *CustomReceiptRecognitionApi) RecognizeReceiptImageDetailsHandler(c *core.WebContext) (any, *errs.Error) {
	if !a.CurrentConfig().TransactionFromAIImageRecognition {
		return nil, errs.ErrLargeLanguageModelProviderNotEnabled
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Warnf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get user for user \"uid:%d\", because %s", uid, err.Error())
		}

		return false, errs.ErrUserNotFound
	}

	if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION) {
		return false, errs.ErrNotPermittedToPerformThisAction
	}

	form, err := c.MultipartForm()

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get multi-part form data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrParameterInvalid
	}

	imageFiles := form.File["image"]

	if len(imageFiles) < 1 {
		log.Warnf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] there is no image in request for user \"uid:%d\"", uid)
		return nil, errs.ErrNoAIRecognitionImage
	}

	if imageFiles[0].Size < 1 {
		log.Warnf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] the size of image in request is zero for user \"uid:%d\"", uid)
		return nil, errs.ErrAIRecognitionImageIsEmpty
	}

	if imageFiles[0].Size > int64(a.CurrentConfig().MaxAIRecognitionPictureFileSize) {
		log.Warnf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] the upload file size \"%d\" exceeds the maximum size \"%d\" of image for user \"uid:%d\"", imageFiles[0].Size, a.CurrentConfig().MaxAIRecognitionPictureFileSize, uid)
		return nil, errs.ErrExceedMaxAIRecognitionImageFileSize
	}

	fileExtension := utils.GetFileNameExtension(imageFiles[0].Filename)
	contentType := utils.GetImageContentType(fileExtension)

	if contentType == "" {
		log.Warnf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] the file extension \"%s\" of image in request is not supported for user \"uid:%d\"", fileExtension, uid)
		return nil, errs.ErrImageTypeNotSupported
	}

	imageFile, err := imageFiles[0].Open()

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get image file from request for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	defer imageFile.Close()

	imageData, err := io.ReadAll(imageFile)

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to read image file from request for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	accounts, err := a.accounts.GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	accountMap := a.accounts.GetVisibleAccountNameMapByList(accounts)
	accountNames := make([]string, 0, len(accounts))

	for i := 0; i < len(accounts); i++ {
		if accounts[i].Hidden || accounts[i].Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			continue
		}

		accountNames = append(accountNames, accounts[i].Name)
	}

	categories, err := a.transactionCategories.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	incomeCategoryMap := make(map[string]*models.TransactionCategory)
	incomeCategoryNames := make([]string, 0)

	expenseCategoryMap := make(map[string]*models.TransactionCategory)
	expenseCategoryNames := make([]string, 0)

	transferCategoryMap := make(map[string]*models.TransactionCategory)
	transferCategoryNames := make([]string, 0)

	for i := 0; i < len(categories); i++ {
		category := categories[i]

		if category.Hidden || category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
			continue
		}

		if category.Type == models.CATEGORY_TYPE_INCOME {
			incomeCategoryMap[category.Name] = category
			incomeCategoryNames = append(incomeCategoryNames, category.Name)
		} else if category.Type == models.CATEGORY_TYPE_EXPENSE {
			expenseCategoryMap[category.Name] = category
			expenseCategoryNames = append(expenseCategoryNames, category.Name)
		} else if category.Type == models.CATEGORY_TYPE_TRANSFER {
			transferCategoryMap[category.Name] = category
			transferCategoryNames = append(transferCategoryNames, category.Name)
		}
	}

	tags, err := a.transactionTags.GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	tagMap := a.transactionTags.GetVisibleTagNameMapByList(tags)
	tagNames := make([]string, 0, len(tags))

	for i := 0; i < len(tags); i++ {
		if tags[i].Hidden {
			continue
		}

		tagNames = append(tagNames, tags[i].Name)
	}

	systemPrompt, err := templates.GetTemplate(templates.SYSTEM_PROMPT_RECEIPT_IMAGE_RECOGNITION_DETAILS)

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get system prompt template for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	systemPromptParams := map[string]any{
		"CurrentDateTime":          utils.FormatUnixTimeToLongDateTime(time.Now().Unix(), clientTimezone),
		"AllExpenseCategoryNames":  strings.Join(expenseCategoryNames, "\n"),
		"AllIncomeCategoryNames":   strings.Join(incomeCategoryNames, "\n"),
		"AllTransferCategoryNames": strings.Join(transferCategoryNames, "\n"),
		"AllAccountNames":          strings.Join(accountNames, "\n"),
		"AllTagNames":              strings.Join(tagNames, "\n"),
	}

	var bodyBuffer bytes.Buffer
	err = systemPrompt.Execute(&bodyBuffer, systemPromptParams)

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get final system prompt from template for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	// the user-defined active prompt (applyCustomReceiptSystemPrompt) is intentionally not applied here,
	// user-defined prompts are written for the legacy single-transaction schema and would break the items output

	llmRequest := &data.LargeLanguageModelRequest{
		Stream:                false,
		SystemPrompt:          strings.ReplaceAll(bodyBuffer.String(), "\r\n", "\n"),
		UserPrompt:            imageData,
		UserPromptType:        data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL,
		UserPromptContentType: contentType,
	}

	llmResponse, err := getCustomReceiptRecognitionJsonResponse(c, uid, a.CurrentConfig(), llmRequest)

	if err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to get llm response user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if llmResponse == nil || len(llmResponse.Content) == 0 || strings.HasPrefix(llmResponse.Content, "{}") {
		return nil, errs.ErrNoTransactionInformationInImage
	}

	var result *models.RecognizedReceiptDetailsResult

	if err := json.Unmarshal([]byte(llmResponse.Content), &result); err != nil {
		log.Errorf(c, "[custom_receipt_recognition.RecognizeReceiptImageDetailsHandler] failed to unmarshal recognized receipt details result from llm response \"%s\" for user \"uid:%d\", because %s", llmResponse.Content, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if result == nil {
		return nil, errs.ErrNoTransactionInformationInImage
	}

	baseResponse, errResp := LargeLanguageModels.parseRecognizedReceiptImageResponse(c, uid, clientTimezone, &result.RecognizedReceiptImageResult, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)

	if errResp != nil {
		return nil, errResp
	}

	response := &models.RecognizedReceiptDetailsResponse{
		RecognizedReceiptImageResponse: *baseResponse,
	}

	items := a.parseRecognizedReceiptItems(c, uid, response.Type, result.Items, expenseCategoryMap, incomeCategoryMap, transferCategoryMap)

	if len(items) > 0 {
		totalAmount, reconciledItems := models.ReconcileRecognizedReceiptItems(response.Type, response.SourceAmount, items)
		response.SourceAmount = totalAmount
		response.Items = reconciledItems
	}

	if result.Merchant != nil {
		geocodingResult := services.CustomGeocoding.ResolveReceiptLocation(c, result.Merchant.Name, result.Merchant.Branch, result.Merchant.Address)

		if geocodingResult != nil {
			response.GeoLocation = &models.TransactionGeoLocationResponse{
				Latitude:  geocodingResult.Latitude,
				Longitude: geocodingResult.Longitude,
			}

			confidence := "low"

			if geocodingResult.Confidence == geocoding.GEOCODING_CONFIDENCE_HIGH {
				confidence = "high"
			}

			response.Location = &models.RecognizedReceiptLocationResponse{
				DisplayName: geocodingResult.DisplayName,
				Confidence:  confidence,
				Provider:    geocodingResult.Provider,
			}
		}
	}

	return response, nil
}

// parseRecognizedReceiptItems converts the recognized receipt item results to view-objects.
// It returns nil when there is no item or any item amount cannot be parsed, so that the caller
// falls back to the legacy single-transaction flow instead of failing the whole recognition
func (a *CustomReceiptRecognitionApi) parseRecognizedReceiptItems(c *core.WebContext, uid int64, transactionType models.TransactionType, recognizedItems []*models.RecognizedReceiptItemResult, expenseCategoryMap map[string]*models.TransactionCategory, incomeCategoryMap map[string]*models.TransactionCategory, transferCategoryMap map[string]*models.TransactionCategory) []*models.RecognizedReceiptItemResponse {
	if len(recognizedItems) < 1 {
		return nil
	}

	var categoryMap map[string]*models.TransactionCategory

	if transactionType == models.TRANSACTION_TYPE_EXPENSE {
		categoryMap = expenseCategoryMap
	} else if transactionType == models.TRANSACTION_TYPE_INCOME {
		categoryMap = incomeCategoryMap
	} else if transactionType == models.TRANSACTION_TYPE_TRANSFER {
		categoryMap = transferCategoryMap
	}

	items := make([]*models.RecognizedReceiptItemResponse, 0, len(recognizedItems))

	for i := 0; i < len(recognizedItems); i++ {
		recognizedItem := recognizedItems[i]

		if recognizedItem == nil {
			log.Warnf(c, "[custom_receipt_recognition.parseRecognizedReceiptItems] recognized item #%d is null for user \"uid:%d\", fallback to single transaction", i, uid)
			return nil
		}

		// utils.ParseAmount silently maps an empty string to 0, so a missing amount must be rejected
		// explicitly to trigger the single-transaction fallback (A-6) instead of producing a 0 item
		if len(recognizedItem.Amount) < 1 {
			log.Warnf(c, "[custom_receipt_recognition.parseRecognizedReceiptItems] recognized amount of item #%d is missing for user \"uid:%d\", fallback to single transaction", i, uid)
			return nil
		}

		amount, err := utils.ParseAmount(recognizedItem.Amount)

		if err != nil {
			log.Warnf(c, "[custom_receipt_recognition.parseRecognizedReceiptItems] recognized amount \"%s\" of item #%d is invalid for user \"uid:%d\", fallback to single transaction", recognizedItem.Amount, i, uid)
			return nil
		}

		item := &models.RecognizedReceiptItemResponse{
			Amount:  amount,
			Comment: recognizedItem.Name,
		}

		if len(recognizedItem.Category) > 0 && categoryMap != nil {
			category, exists := categoryMap[recognizedItem.Category]

			if exists {
				item.CategoryId = category.CategoryId
			}
		}

		items = append(items, item)
	}

	return items
}
