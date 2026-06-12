package errs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomLlmProfileErrorCodes(t *testing.T) {
	// the fork-local subcategory 92 must keep error codes in the 292xxx range, far from upstream codes (up to 219xxx)
	assert.Equal(t, int32(292000), ErrLlmProfileIdInvalid.Code())
	assert.Equal(t, int32(292001), ErrLlmProfileNotFound.Code())
	assert.Equal(t, int32(292002), ErrLlmProfileNameAlreadyExists.Code())
	assert.Equal(t, int32(292003), ErrLlmProfileCountExceedsLimit.Code())
	assert.Equal(t, int32(292004), ErrLlmProfileProviderInvalid.Code())
	assert.Equal(t, int32(292005), ErrLlmProfileMissingRequiredField.Code())
	assert.Equal(t, int32(292006), ErrLlmProfileDefaultEntryReadOnly.Code())
	assert.Equal(t, int32(292007), ErrLlmProfileApiKeyDecryptFailed.Code())
	assert.Equal(t, int32(292008), ErrLlmProfileNoAvailableProvider.Code())
}
