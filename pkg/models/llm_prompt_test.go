package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLlmPromptToLlmPromptInfoResponse_IncludeContent(t *testing.T) {
	prompt := &LlmPrompt{
		PromptId:        1234567890,
		Uid:             1001,
		Name:            "Test Prompt",
		Content:         "## Role\n{{.CurrentDateTime}}",
		IsActive:        true,
		CreatedUnixTime: 1781000000,
		UpdatedUnixTime: 1781100000,
	}

	actualResponse := prompt.ToLlmPromptInfoResponse(true)

	assert.Equal(t, int64(1234567890), actualResponse.Id)
	assert.Equal(t, "Test Prompt", actualResponse.Name)
	assert.Equal(t, "## Role\n{{.CurrentDateTime}}", actualResponse.Content)
	assert.Equal(t, true, actualResponse.IsActive)
	assert.Equal(t, int64(1781000000), actualResponse.CreatedTime)
	assert.Equal(t, int64(1781100000), actualResponse.UpdatedTime)
}

func TestLlmPromptToLlmPromptInfoResponse_ExcludeContent(t *testing.T) {
	prompt := &LlmPrompt{
		PromptId: 1234567890,
		Uid:      1001,
		Name:     "Test Prompt",
		Content:  "## Role\n{{.CurrentDateTime}}",
		IsActive: false,
	}

	actualResponse := prompt.ToLlmPromptInfoResponse(false)

	assert.Equal(t, int64(1234567890), actualResponse.Id)
	assert.Equal(t, "Test Prompt", actualResponse.Name)
	assert.Equal(t, "", actualResponse.Content)
	assert.Equal(t, false, actualResponse.IsActive)
}
