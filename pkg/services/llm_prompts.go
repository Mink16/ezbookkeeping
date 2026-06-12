package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// llmPromptUuidType is a fork-local uuid type (upstream allocates sequentially from 0, currently up to 10;
// this fork reserves 15, the maximum of the 4-bit uuid type space, to minimize collision)
const llmPromptUuidType = uuid.UuidType(15)

const maxLlmPromptCountPerUser = 20

// LlmPromptService represents user-defined llm prompt service
type LlmPromptService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a llm prompt service singleton instance
var (
	LlmPrompts = &LlmPromptService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// GetAllPromptsByUid returns all llm prompt models of user
func (s *LlmPromptService) GetAllPromptsByUid(c core.Context, uid int64) ([]*models.LlmPrompt, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var prompts []*models.LlmPrompt
	err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).OrderBy("created_unix_time asc").Find(&prompts)

	return prompts, err
}

// GetPromptByPromptId returns a llm prompt model according to llm prompt id
func (s *LlmPromptService) GetPromptByPromptId(c core.Context, uid int64, promptId int64) (*models.LlmPrompt, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if promptId <= 0 {
		return nil, errs.ErrLlmPromptIdInvalid
	}

	prompt := &models.LlmPrompt{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(promptId).Where("uid=? AND deleted=?", uid, false).Get(prompt)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrLlmPromptNotFound
	}

	return prompt, nil
}

// GetActivePromptByUid returns the active llm prompt model of user, or nil if no prompt is active
func (s *LlmPromptService) GetActivePromptByUid(c core.Context, uid int64) (*models.LlmPrompt, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	prompt := &models.LlmPrompt{}
	has, err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=? AND is_active=?", uid, false, true).Limit(1).Get(prompt)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return prompt, nil
}

// ExistsPromptName returns whether the given prompt name exists for user (excluding the specified prompt id)
func (s *LlmPromptService) ExistsPromptName(c core.Context, uid int64, name string, excludePromptId int64) (bool, error) {
	if name == "" {
		return false, errs.ErrLlmPromptNotFound
	}

	return s.UserDataDB(uid).NewSession(c).Cols("name").Where("uid=? AND deleted=? AND name=? AND prompt_id<>?", uid, false, name, excludePromptId).Exist(&models.LlmPrompt{})
}

// CreatePrompt saves a new llm prompt model to database
func (s *LlmPromptService) CreatePrompt(c core.Context, prompt *models.LlmPrompt) error {
	if prompt.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	count, err := s.UserDataDB(prompt.Uid).NewSession(c).Where("uid=? AND deleted=?", prompt.Uid, false).Count(&models.LlmPrompt{})

	if err != nil {
		return err
	} else if count >= maxLlmPromptCountPerUser {
		return errs.ErrLlmPromptCountExceedsLimit
	}

	exists, err := s.ExistsPromptName(c, prompt.Uid, prompt.Name, 0)

	if err != nil {
		return err
	} else if exists {
		return errs.ErrLlmPromptNameAlreadyExists
	}

	prompt.PromptId = s.GenerateUuid(llmPromptUuidType)

	if prompt.PromptId < 1 {
		return errs.ErrSystemIsBusy
	}

	prompt.Deleted = false
	prompt.IsActive = false
	prompt.CreatedUnixTime = time.Now().Unix()
	prompt.UpdatedUnixTime = time.Now().Unix()

	return s.UserDataDB(prompt.Uid).DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Insert(prompt)
		return err
	})
}

// ModifyPrompt updates an existed llm prompt model in database
func (s *LlmPromptService) ModifyPrompt(c core.Context, prompt *models.LlmPrompt) error {
	if prompt.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	exists, err := s.ExistsPromptName(c, prompt.Uid, prompt.Name, prompt.PromptId)

	if err != nil {
		return err
	} else if exists {
		return errs.ErrLlmPromptNameAlreadyExists
	}

	prompt.UpdatedUnixTime = time.Now().Unix()

	return s.UserDataDB(prompt.Uid).DoTransaction(c, func(sess *xorm.Session) error {
		updatedRows, err := sess.ID(prompt.PromptId).Cols("name", "content", "updated_unix_time").Where("uid=? AND deleted=?", prompt.Uid, false).Update(prompt)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrLlmPromptNotFound
		}

		return err
	})
}

// DeletePrompt deletes an existed llm prompt from database, deactivating it at the same time
// so that deleting the active prompt naturally falls back to the default prompt
func (s *LlmPromptService) DeletePrompt(c core.Context, uid int64, promptId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if promptId <= 0 {
		return errs.ErrLlmPromptIdInvalid
	}

	now := time.Now().Unix()

	updateModel := &models.LlmPrompt{
		Deleted:         true,
		IsActive:        false,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.ID(promptId).Cols("deleted", "is_active", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrLlmPromptNotFound
		}

		return err
	})
}

// SetActivePrompt activates the specified llm prompt and deactivates all the others of user,
// promptId 0 deactivates all prompts (the default prompt will be used)
func (s *LlmPromptService) SetActivePrompt(c core.Context, uid int64, promptId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if promptId < 0 {
		return errs.ErrLlmPromptIdInvalid
	}

	now := time.Now().Unix()

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if promptId > 0 {
			exists, err := sess.Cols("prompt_id").Where("uid=? AND deleted=? AND prompt_id=?", uid, false, promptId).Limit(1).Exist(&models.LlmPrompt{})

			if err != nil {
				return err
			} else if !exists {
				return errs.ErrLlmPromptNotFound
			}
		}

		_, err := sess.Cols("is_active", "updated_unix_time").Where("uid=? AND deleted=? AND is_active=?", uid, false, true).Update(&models.LlmPrompt{
			IsActive:        false,
			UpdatedUnixTime: now,
		})

		if err != nil {
			return err
		}

		if promptId > 0 {
			updatedRows, err := sess.ID(promptId).Cols("is_active", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(&models.LlmPrompt{
				IsActive:        true,
				UpdatedUnixTime: now,
			})

			if err != nil {
				return err
			} else if updatedRows < 1 {
				return errs.ErrLlmPromptNotFound
			}
		}

		return nil
	})
}
