package llm

import (
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/llm/provider"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// NewCustomLargeLanguageModelProvider returns a large language model provider built from the given config (fork-local).
// It exposes the private upstream factory so that fork features can build providers dynamically
// (e.g. from database-stored llm connection profiles) instead of the startup-initialized singleton
func NewCustomLargeLanguageModelProvider(llmConfig *settings.LLMConfig, enableResponseLog bool) (provider.LargeLanguageModelProvider, error) {
	if llmConfig == nil || llmConfig.LLMProvider == "" {
		return nil, errs.ErrInvalidLLMProvider
	}

	return initializeLargeLanguageModelProvider(llmConfig, enableResponseLog)
}
