package errs

import "net/http"

// NormalSubcategoryCustomLlmProfile is a fork-local error subcategory for custom llm connection profile features
// (upstream uses subcategories 0-19, this fork reserves 90-99 to avoid collision; 90/91 are already used)
const NormalSubcategoryCustomLlmProfile = 92

// Error codes related to custom llm connection profiles
var (
	ErrLlmProfileIdInvalid            = NewNormalError(NormalSubcategoryCustomLlmProfile, 0, http.StatusBadRequest, "llm profile id is invalid")
	ErrLlmProfileNotFound             = NewNormalError(NormalSubcategoryCustomLlmProfile, 1, http.StatusBadRequest, "llm profile not found")
	ErrLlmProfileNameAlreadyExists    = NewNormalError(NormalSubcategoryCustomLlmProfile, 2, http.StatusBadRequest, "llm profile name already exists")
	ErrLlmProfileCountExceedsLimit    = NewNormalError(NormalSubcategoryCustomLlmProfile, 3, http.StatusBadRequest, "llm profile count exceeds limitation")
	ErrLlmProfileProviderInvalid      = NewNormalError(NormalSubcategoryCustomLlmProfile, 4, http.StatusBadRequest, "llm profile provider is invalid")
	ErrLlmProfileMissingRequiredField = NewNormalError(NormalSubcategoryCustomLlmProfile, 5, http.StatusBadRequest, "llm profile required field is missing")
	ErrLlmProfileDefaultEntryReadOnly = NewNormalError(NormalSubcategoryCustomLlmProfile, 6, http.StatusBadRequest, "default llm profile cannot be modified or deleted")
	ErrLlmProfileApiKeyDecryptFailed  = NewNormalError(NormalSubcategoryCustomLlmProfile, 7, http.StatusInternalServerError, "failed to decrypt llm profile api key")
	ErrLlmProfileNoAvailableProvider  = NewNormalError(NormalSubcategoryCustomLlmProfile, 8, http.StatusBadRequest, "no available llm provider")
)
