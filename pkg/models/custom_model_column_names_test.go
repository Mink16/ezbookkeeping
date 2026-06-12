package models

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"xorm.io/xorm/names"
)

// fork-local database models, every new fork model must be added here
var customModelsForColumnNameCheck = []any{
	new(LlmPrompt),
	new(CustomAdmin),
	new(CustomLlmProfile),
}

// matches snake names containing split acronyms like "base_u_r_l" or "model_i_d"
var acronymSplitPattern = regexp.MustCompile(`(^|_)[a-z](_[a-z])+(_|$)`)

// The default xorm snake mapper splits acronyms (BaseURL -> "base_u_r_l", ModelID -> "model_i_d"),
// which silently breaks every Cols("base_url", ...) usage in the service layer because the string
// no longer matches the real column name (regression test for the bug fixed in commit 86a617cc).
// Fields containing acronyms must declare an explicit column name tag like xorm:"'base_url' ...".
func TestCustomModelColumnNamesHaveNoSplitAcronyms(t *testing.T) {
	mapper := names.SnakeMapper{}

	for _, model := range customModelsForColumnNameCheck {
		modelType := reflect.TypeOf(model).Elem()

		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			tag := field.Tag.Get("xorm")

			if tag == "-" || strings.Contains(tag, "'") {
				// not mapped, or an explicit column name is declared
				continue
			}

			columnName := mapper.Obj2Table(field.Name)

			if acronymSplitPattern.MatchString(columnName) {
				t.Errorf("%s.%s maps to column \"%s\" (split acronym), declare an explicit column name tag like xorm:\"'%s' ...\"",
					modelType.Name(), field.Name, columnName, strings.ToLower(field.Name))
			}
		}
	}
}
