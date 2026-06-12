package errs

import "net/http"

// NormalSubcategoryCustomLlmPrompt is a fork-local error subcategory for custom llm prompt features
// (upstream uses subcategories 0-19, this fork reserves 90 to avoid collision)
const NormalSubcategoryCustomLlmPrompt = 90

// Error codes related to custom llm prompts
var (
	ErrLlmPromptIdInvalid         = NewNormalError(NormalSubcategoryCustomLlmPrompt, 0, http.StatusBadRequest, "llm prompt id is invalid")
	ErrLlmPromptNotFound          = NewNormalError(NormalSubcategoryCustomLlmPrompt, 1, http.StatusBadRequest, "llm prompt not found")
	ErrLlmPromptNameAlreadyExists = NewNormalError(NormalSubcategoryCustomLlmPrompt, 2, http.StatusBadRequest, "llm prompt name already exists")
	ErrLlmPromptContentTooLarge   = NewNormalError(NormalSubcategoryCustomLlmPrompt, 3, http.StatusBadRequest, "llm prompt content exceeds maximum size")
	ErrLlmPromptCountExceedsLimit = NewNormalError(NormalSubcategoryCustomLlmPrompt, 4, http.StatusBadRequest, "llm prompt count exceeds limitation")
	ErrLlmPromptContentInvalid    = NewNormalError(NormalSubcategoryCustomLlmPrompt, 5, http.StatusBadRequest, "llm prompt content is not a valid template")
)
