package errs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLlmPromptErrorCodes(t *testing.T) {
	// the fork-local subcategory 90 must keep error codes in the 290xxx range, far from upstream codes (up to 219xxx)
	assert.Equal(t, int32(290000), ErrLlmPromptIdInvalid.Code())
	assert.Equal(t, int32(290001), ErrLlmPromptNotFound.Code())
	assert.Equal(t, int32(290002), ErrLlmPromptNameAlreadyExists.Code())
	assert.Equal(t, int32(290003), ErrLlmPromptContentTooLarge.Code())
	assert.Equal(t, int32(290004), ErrLlmPromptCountExceedsLimit.Code())
	assert.Equal(t, int32(290005), ErrLlmPromptContentInvalid.Code())
}
