package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	texttemplate "text/template"
)

var rawTemplateContentCache = make(map[KnownTemplate]string)
var rawTemplateContentCacheMutex sync.RWMutex

// ParseUserPromptTemplate parses user-defined prompt content as a text/template
// (text/template is used intentionally instead of html/template to avoid html escaping in llm prompts)
func ParseUserPromptTemplate(content string) (*texttemplate.Template, error) {
	return texttemplate.New("user_prompt").Option("missingkey=error").Parse(content)
}

// RenderUserPromptTemplate parses and executes user-defined prompt content with the specified parameters
func RenderUserPromptTemplate(content string, params map[string]any) (string, error) {
	tmpl, err := ParseUserPromptTemplate(content)

	if err != nil {
		return "", err
	}

	var bodyBuffer bytes.Buffer
	err = tmpl.Execute(&bodyBuffer, params)

	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(bodyBuffer.String(), "\r\n", "\n"), nil
}

// GetTemplateRawContent returns the raw (unparsed) content of the specified template file
func GetTemplateRawContent(templateName KnownTemplate) (string, error) {
	rawTemplateContentCacheMutex.RLock()
	cachedContent, exists := rawTemplateContentCache[templateName]
	rawTemplateContentCacheMutex.RUnlock()

	if exists {
		return cachedContent, nil
	}

	fullPath := filepath.Join(templateBasePath, fmt.Sprintf("%s.%s", templateName, templateFileExtension))
	content, err := os.ReadFile(fullPath)

	if err != nil {
		return "", err
	}

	rawContent := string(content)

	rawTemplateContentCacheMutex.Lock()
	rawTemplateContentCache[templateName] = rawContent
	rawTemplateContentCacheMutex.Unlock()

	return rawContent, nil
}
