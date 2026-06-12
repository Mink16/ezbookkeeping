package services

import (
	"os"
	"strings"
	"sync"
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// customAdminUsernamesEnvName is the environment variable bootstrapping server admins (fork-local).
// It is read directly via os.Getenv instead of the upstream settings loader to keep pkg/settings unchanged;
// the name follows the upstream EBK_{SECTION}_{ITEM} convention with the upstream-unused "custom" section,
// but unlike upstream settings it cannot be set in the ini file or via EBK_FILE_* indirection
const customAdminUsernamesEnvName = "EBK_CUSTOM_ADMIN_USERNAMES"

// CustomAdminService represents the server-wide administrator service (fork-local feature)
type CustomAdminService struct {
	ServiceUsingDB
	envAdminUsernamesOnce sync.Once
	envAdminUsernames     []string
}

// Initialize a custom admin service singleton instance
var (
	CustomAdmins = &CustomAdminService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
	}
)

// ParseEnvAdminUsernames returns the comma-separated username list with whitespace trimmed and empty items removed
func ParseEnvAdminUsernames(value string) []string {
	items := strings.Split(value, ",")
	usernames := make([]string, 0, len(items))

	for i := 0; i < len(items); i++ {
		username := strings.TrimSpace(items[i])

		if username != "" {
			usernames = append(usernames, username)
		}
	}

	return usernames
}

// IsEnvAdminUsername returns whether the given username is in the environment-defined admin username list (exact match)
func IsEnvAdminUsername(envUsernames []string, username string) bool {
	if username == "" {
		return false
	}

	for i := 0; i < len(envUsernames); i++ {
		if envUsernames[i] == username {
			return true
		}
	}

	return false
}

// GetEnvAdminUsernames returns the environment-defined admin usernames (read once and cached)
func (s *CustomAdminService) GetEnvAdminUsernames() []string {
	s.envAdminUsernamesOnce.Do(func() {
		s.envAdminUsernames = ParseEnvAdminUsernames(os.Getenv(customAdminUsernamesEnvName))
	})

	return s.envAdminUsernames
}

// IsAdmin returns whether the given user is a server admin (environment-defined or granted in database)
func (s *CustomAdminService) IsAdmin(c core.Context, uid int64, username string) (bool, error) {
	if uid <= 0 {
		return false, errs.ErrUserIdInvalid
	}

	if IsEnvAdminUsername(s.GetEnvAdminUsernames(), username) {
		return true, nil
	}

	return s.UserDB().NewSession(c).ID(uid).Exist(&models.CustomAdmin{})
}

// GetAllAdminUids returns the uid set of all database-granted admins
func (s *CustomAdminService) GetAllAdminUids(c core.Context) (map[int64]bool, error) {
	var admins []*models.CustomAdmin
	err := s.UserDB().NewSession(c).Find(&admins)

	if err != nil {
		return nil, err
	}

	adminUids := make(map[int64]bool, len(admins))

	for i := 0; i < len(admins); i++ {
		adminUids[admins[i].Uid] = true
	}

	return adminUids, nil
}

// GetAllUsers returns all not-deleted users (uid, username and nickname columns only)
func (s *CustomAdminService) GetAllUsers(c core.Context) ([]*models.User, error) {
	var users []*models.User
	err := s.UserDB().NewSession(c).Cols("uid", "username", "nickname").Where("deleted=?", false).OrderBy("uid asc").Find(&users)

	return users, err
}

// GrantAdmin grants the database admin permission to the given user
func (s *CustomAdminService) GrantAdmin(c core.Context, targetUid int64, operatorUid int64) error {
	if targetUid <= 0 {
		return errs.ErrUserIdInvalid
	}

	exists, err := s.UserDB().NewSession(c).ID(targetUid).Exist(&models.CustomAdmin{})

	if err != nil {
		return err
	} else if exists {
		return errs.ErrCustomAdminAlreadyGranted
	}

	admin := &models.CustomAdmin{
		Uid:             targetUid,
		GrantedByUid:    operatorUid,
		CreatedUnixTime: time.Now().Unix(),
	}

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Insert(admin)
		return err
	})
}

// RevokeAdmin revokes the database admin permission from the given user
func (s *CustomAdminService) RevokeAdmin(c core.Context, targetUid int64) error {
	if targetUid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.ID(targetUid).Delete(&models.CustomAdmin{})

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrCustomAdminNotGranted
		}

		return nil
	})
}
