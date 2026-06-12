package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func TestBuildVisibleAccountNames(t *testing.T) {
	accounts := []*models.Account{
		{Name: "Cash", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
		{Name: "Hidden Account", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Hidden: true},
		{Name: "Parent Account", Type: models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS},
		{Name: "Bank Card", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}

	actualNames := buildVisibleAccountNames(accounts)

	assert.Equal(t, []string{"Cash", "Bank Card"}, actualNames)
}

func TestBuildVisibleAccountNames_EmptyList(t *testing.T) {
	actualNames := buildVisibleAccountNames(nil)

	assert.NotNil(t, actualNames)
	assert.Equal(t, 0, len(actualNames))
}

func TestBuildCategoryNamesByType(t *testing.T) {
	categories := []*models.TransactionCategory{
		{Name: "Food", Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: models.LevelOneTransactionCategoryParentId},
		{Name: "Groceries", Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 1001},
		{Name: "Hidden Category", Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 1001, Hidden: true},
		{Name: "Salary", Type: models.CATEGORY_TYPE_INCOME, ParentCategoryId: 1002},
		{Name: "General Transfer", Type: models.CATEGORY_TYPE_TRANSFER, ParentCategoryId: 1003},
	}

	expenseNames, incomeNames, transferNames := buildCategoryNamesByType(categories)

	assert.Equal(t, []string{"Groceries"}, expenseNames)
	assert.Equal(t, []string{"Salary"}, incomeNames)
	assert.Equal(t, []string{"General Transfer"}, transferNames)
}

func TestBuildCategoryNamesByType_EmptyList(t *testing.T) {
	expenseNames, incomeNames, transferNames := buildCategoryNamesByType(nil)

	assert.Equal(t, 0, len(expenseNames))
	assert.Equal(t, 0, len(incomeNames))
	assert.Equal(t, 0, len(transferNames))
}

func TestBuildVisibleTagNames(t *testing.T) {
	tags := []*models.TransactionTag{
		{Name: "family"},
		{Name: "hidden tag", Hidden: true},
		{Name: "work"},
	}

	actualNames := buildVisibleTagNames(tags)

	assert.Equal(t, []string{"family", "work"}, actualNames)
}
