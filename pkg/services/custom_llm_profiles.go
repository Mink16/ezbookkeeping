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

// customLlmProfileUuidType is a fork-local uuid type (upstream allocates sequentially from 0, currently up to 10;
// this fork reserves the values downwards from the 4-bit maximum: 15 is used by llm prompts, 14 by llm profiles)
const customLlmProfileUuidType = uuid.UuidType(14)

const maxCustomLlmProfileCount = 20

// CustomLlmProfileService represents the server-wide llm connection profile service (fork-local feature)
type CustomLlmProfileService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a custom llm profile service singleton instance
var (
	CustomLlmProfiles = &CustomLlmProfileService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// GetAllProfiles returns all llm profile models
func (s *CustomLlmProfileService) GetAllProfiles(c core.Context) ([]*models.CustomLlmProfile, error) {
	var profiles []*models.CustomLlmProfile
	err := s.UserDB().NewSession(c).Where("deleted=?", false).OrderBy("created_unix_time asc").Find(&profiles)

	return profiles, err
}

// GetProfileById returns a llm profile model according to profile id
func (s *CustomLlmProfileService) GetProfileById(c core.Context, profileId int64) (*models.CustomLlmProfile, error) {
	if profileId <= 0 {
		return nil, errs.ErrLlmProfileIdInvalid
	}

	profile := &models.CustomLlmProfile{}
	has, err := s.UserDB().NewSession(c).ID(profileId).Where("deleted=?", false).Get(profile)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrLlmProfileNotFound
	}

	return profile, nil
}

// GetActiveProfile returns the active llm profile model, or nil if no profile is active
func (s *CustomLlmProfileService) GetActiveProfile(c core.Context) (*models.CustomLlmProfile, error) {
	profile := &models.CustomLlmProfile{}
	has, err := s.UserDB().NewSession(c).Where("deleted=? AND is_active=?", false, true).Limit(1).Get(profile)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return profile, nil
}

// ExistsProfileName returns whether the given profile name exists (excluding the specified profile id)
func (s *CustomLlmProfileService) ExistsProfileName(c core.Context, name string, excludeProfileId int64) (bool, error) {
	if name == "" {
		return false, errs.ErrLlmProfileNotFound
	}

	return s.UserDB().NewSession(c).Cols("name").Where("deleted=? AND name=? AND profile_id<>?", false, name, excludeProfileId).Exist(&models.CustomLlmProfile{})
}

// CreateProfile saves a new llm profile model to database
func (s *CustomLlmProfileService) CreateProfile(c core.Context, profile *models.CustomLlmProfile) error {
	count, err := s.UserDB().NewSession(c).Where("deleted=?", false).Count(&models.CustomLlmProfile{})

	if err != nil {
		return err
	} else if count >= maxCustomLlmProfileCount {
		return errs.ErrLlmProfileCountExceedsLimit
	}

	exists, err := s.ExistsProfileName(c, profile.Name, 0)

	if err != nil {
		return err
	} else if exists {
		return errs.ErrLlmProfileNameAlreadyExists
	}

	profile.ProfileId = s.GenerateUuid(customLlmProfileUuidType)

	if profile.ProfileId < 1 {
		return errs.ErrSystemIsBusy
	}

	profile.Deleted = false
	profile.IsActive = false
	profile.CreatedUnixTime = time.Now().Unix()
	profile.UpdatedUnixTime = time.Now().Unix()

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Insert(profile)
		return err
	})
}

// ModifyProfile updates an existed llm profile model in database,
// the api_key column is only updated when updateAPIKey is true (an empty submitted key keeps the stored one)
func (s *CustomLlmProfileService) ModifyProfile(c core.Context, profile *models.CustomLlmProfile, updateAPIKey bool) error {
	exists, err := s.ExistsProfileName(c, profile.Name, profile.ProfileId)

	if err != nil {
		return err
	} else if exists {
		return errs.ErrLlmProfileNameAlreadyExists
	}

	profile.UpdatedUnixTime = time.Now().Unix()

	updateCols := []string{"name", "provider", "base_url", "api_version", "model_id", "max_tokens", "updated_unix_time"}

	if updateAPIKey {
		updateCols = append(updateCols, "api_key")
	}

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		updatedRows, err := sess.ID(profile.ProfileId).Cols(updateCols...).Where("deleted=?", false).Update(profile)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrLlmProfileNotFound
		}

		return err
	})
}

// DeleteProfile deletes an existed llm profile from database, deactivating it at the same time
// so that deleting the active profile naturally falls back to the default environment configuration
func (s *CustomLlmProfileService) DeleteProfile(c core.Context, profileId int64) error {
	if profileId <= 0 {
		return errs.ErrLlmProfileIdInvalid
	}

	now := time.Now().Unix()

	updateModel := &models.CustomLlmProfile{
		Deleted:         true,
		IsActive:        false,
		DeletedUnixTime: now,
	}

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.ID(profileId).Cols("deleted", "is_active", "deleted_unix_time").Where("deleted=?", false).Update(updateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrLlmProfileNotFound
		}

		return err
	})
}

// SetActiveProfile activates the specified llm profile and deactivates all the others,
// profileId 0 deactivates all profiles (the default environment configuration will be used)
func (s *CustomLlmProfileService) SetActiveProfile(c core.Context, profileId int64) error {
	if profileId < 0 {
		return errs.ErrLlmProfileIdInvalid
	}

	now := time.Now().Unix()

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		if profileId > 0 {
			exists, err := sess.Cols("profile_id").Where("deleted=? AND profile_id=?", false, profileId).Limit(1).Exist(&models.CustomLlmProfile{})

			if err != nil {
				return err
			} else if !exists {
				return errs.ErrLlmProfileNotFound
			}
		}

		_, err := sess.Cols("is_active", "updated_unix_time").Where("deleted=? AND is_active=?", false, true).Update(&models.CustomLlmProfile{
			IsActive:        false,
			UpdatedUnixTime: now,
		})

		if err != nil {
			return err
		}

		if profileId > 0 {
			updatedRows, err := sess.ID(profileId).Cols("is_active", "updated_unix_time").Where("deleted=?", false).Update(&models.CustomLlmProfile{
				IsActive:        true,
				UpdatedUnixTime: now,
			})

			if err != nil {
				return err
			} else if updatedRows < 1 {
				return errs.ErrLlmProfileNotFound
			}
		}

		return nil
	})
}
