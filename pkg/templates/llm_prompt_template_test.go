package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUserPromptTemplate_ValidContent(t *testing.T) {
	tmpl, err := ParseUserPromptTemplate("## Role\nYou are an assistant.\n{{.CurrentDateTime}}")

	assert.Nil(t, err)
	assert.NotNil(t, tmpl)
}

func TestParseUserPromptTemplate_InvalidSyntax(t *testing.T) {
	_, err := ParseUserPromptTemplate("unclosed {{.CurrentDateTime")
	assert.NotNil(t, err)

	_, err = ParseUserPromptTemplate("unexpected {{end}}")
	assert.NotNil(t, err)
}

func TestRenderUserPromptTemplate_AllReceiptParams(t *testing.T) {
	content := "Time: {{.CurrentDateTime}}\nExpense: {{.AllExpenseCategoryNames}}\nIncome: {{.AllIncomeCategoryNames}}\nTransfer: {{.AllTransferCategoryNames}}\nAccounts: {{.AllAccountNames}}\nTags: {{.AllTagNames}}"
	params := map[string]any{
		"CurrentDateTime":          "2026-06-12 10:00:00",
		"AllExpenseCategoryNames":  "Food\nGroceries",
		"AllIncomeCategoryNames":   "Salary",
		"AllTransferCategoryNames": "General Transfer",
		"AllAccountNames":          "Cash\nBank Card",
		"AllTagNames":              "",
	}

	actualContent, err := RenderUserPromptTemplate(content, params)

	assert.Nil(t, err)
	assert.Equal(t, "Time: 2026-06-12 10:00:00\nExpense: Food\nGroceries\nIncome: Salary\nTransfer: General Transfer\nAccounts: Cash\nBank Card\nTags: ", actualContent)
}

func TestRenderUserPromptTemplate_MissingKeyReturnsError(t *testing.T) {
	_, err := RenderUserPromptTemplate("{{.NotExistedParam}}", map[string]any{
		"CurrentDateTime": "2026-06-12 10:00:00",
	})

	assert.NotNil(t, err)
}

func TestRenderUserPromptTemplate_NoHtmlEscaping(t *testing.T) {
	params := map[string]any{
		"AllAccountNames": "Cash & Card <main> \"primary\"",
	}

	actualContent, err := RenderUserPromptTemplate("Accounts: {{.AllAccountNames}}", params)

	assert.Nil(t, err)
	assert.Equal(t, "Accounts: Cash & Card <main> \"primary\"", actualContent)
}

func TestRenderUserPromptTemplate_NormalizesCRLF(t *testing.T) {
	actualContent, err := RenderUserPromptTemplate("line1\r\nline2", map[string]any{})

	assert.Nil(t, err)
	assert.Equal(t, "line1\nline2", actualContent)
}

func TestGetTemplateRawContent(t *testing.T) {
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, templateBasePath, "prompt")
	err := os.MkdirAll(templateDir, 0700)
	assert.Nil(t, err)

	expectedContent := "## Role\n{{.CurrentDateTime}}\n"
	err = os.WriteFile(filepath.Join(templateDir, "test_raw_content.tmpl"), []byte(expectedContent), 0600)
	assert.Nil(t, err)

	t.Chdir(tempDir)

	actualContent, err := GetTemplateRawContent(KnownTemplate("prompt/test_raw_content"))

	assert.Nil(t, err)
	assert.Equal(t, expectedContent, actualContent)
}

func TestGetTemplateRawContent_NotExistedTemplate(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := GetTemplateRawContent(KnownTemplate("prompt/not_existed_template"))

	assert.NotNil(t, err)
}
