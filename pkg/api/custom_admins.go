package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// CustomAdminsApi represents the server admin management api (fork-local feature)
type CustomAdminsApi struct {
	customAdmins *services.CustomAdminService
	users        *services.UserService
}

// Initialize a custom admin api singleton instance
var (
	CustomAdmins = &CustomAdminsApi{
		customAdmins: services.CustomAdmins,
		users:        services.Users,
	}
)

// checkCurrentUserIsCustomAdmin returns nil when the current user is a server admin,
// otherwise returns the permission error. It is shared by all fork-local admin-only handlers
func checkCurrentUserIsCustomAdmin(c *core.WebContext) *errs.Error {
	uid := c.GetCurrentUid()
	user, err := services.Users.GetUserById(c, uid)

	if err != nil {
		log.Warnf(c, "[custom_admins.checkCurrentUserIsCustomAdmin] failed to get user for user \"uid:%d\", because %s", uid, err.Error())
		return errs.ErrCustomAdminPermissionRequired
	}

	isAdmin, err := services.CustomAdmins.IsAdmin(c, uid, user.Username)

	if err != nil {
		log.Warnf(c, "[custom_admins.checkCurrentUserIsCustomAdmin] failed to get admin status for user \"uid:%d\", because %s", uid, err.Error())
		return errs.ErrCustomAdminPermissionRequired
	}

	if !isAdmin {
		return errs.ErrCustomAdminPermissionRequired
	}

	return nil
}

// AdminStatusHandler returns whether the current user is a server admin
func (a *CustomAdminsApi) AdminStatusHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		log.Errorf(c, "[custom_admins.AdminStatusHandler] failed to get user for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	isAdmin, err := a.customAdmins.IsAdmin(c, uid, user.Username)

	if err != nil {
		log.Errorf(c, "[custom_admins.AdminStatusHandler] failed to get admin status for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return &models.CustomAdminStatusResponse{
		IsAdmin: isAdmin,
	}, nil
}

// AdminUserListHandler returns all users with their admin status
func (a *CustomAdminsApi) AdminUserListHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	users, err := a.customAdmins.GetAllUsers(c)

	if err != nil {
		log.Errorf(c, "[custom_admins.AdminUserListHandler] failed to get all users, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	adminUids, err := a.customAdmins.GetAllAdminUids(c)

	if err != nil {
		log.Errorf(c, "[custom_admins.AdminUserListHandler] failed to get all admin uids, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	envAdminUsernames := a.customAdmins.GetEnvAdminUsernames()
	userResps := make([]*models.CustomAdminUserInfoResponse, len(users))

	for i := 0; i < len(users); i++ {
		isEnvAdmin := services.IsEnvAdminUsername(envAdminUsernames, users[i].Username)

		userResps[i] = &models.CustomAdminUserInfoResponse{
			Uid:        users[i].Uid,
			Username:   users[i].Username,
			Nickname:   users[i].Nickname,
			IsAdmin:    isEnvAdmin || adminUids[users[i].Uid],
			IsEnvAdmin: isEnvAdmin,
		}
	}

	return userResps, nil
}

// AdminGrantHandler grants the admin permission to the specified user
func (a *CustomAdminsApi) AdminGrantHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	var grantReq models.CustomAdminGrantRequest
	err := c.ShouldBindJSON(&grantReq)

	if err != nil {
		log.Warnf(c, "[custom_admins.AdminGrantHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	targetUser, err := a.users.GetUserById(c, grantReq.Uid)

	if err != nil || targetUser == nil {
		return nil, errs.ErrCustomAdminUserNotFound
	}

	if services.IsEnvAdminUsername(a.customAdmins.GetEnvAdminUsernames(), targetUser.Username) {
		return nil, errs.ErrCustomAdminAlreadyGranted
	}

	err = a.customAdmins.GrantAdmin(c, grantReq.Uid, c.GetCurrentUid())

	if err != nil {
		log.Errorf(c, "[custom_admins.AdminGrantHandler] failed to grant admin to user \"uid:%d\", because %s", grantReq.Uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return true, nil
}

// AdminRevokeHandler revokes the admin permission from the specified user
func (a *CustomAdminsApi) AdminRevokeHandler(c *core.WebContext) (any, *errs.Error) {
	if errResp := checkCurrentUserIsCustomAdmin(c); errResp != nil {
		return nil, errResp
	}

	var revokeReq models.CustomAdminRevokeRequest
	err := c.ShouldBindJSON(&revokeReq)

	if err != nil {
		log.Warnf(c, "[custom_admins.AdminRevokeHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	if revokeReq.Uid == c.GetCurrentUid() {
		return nil, errs.ErrCustomAdminCannotRevokeSelf
	}

	targetUser, err := a.users.GetUserById(c, revokeReq.Uid)

	if err != nil || targetUser == nil {
		return nil, errs.ErrCustomAdminUserNotFound
	}

	if services.IsEnvAdminUsername(a.customAdmins.GetEnvAdminUsernames(), targetUser.Username) {
		return nil, errs.ErrCustomAdminCannotRevokeEnvAdmin
	}

	err = a.customAdmins.RevokeAdmin(c, revokeReq.Uid)

	if err != nil {
		log.Errorf(c, "[custom_admins.AdminRevokeHandler] failed to revoke admin from user \"uid:%d\", because %s", revokeReq.Uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return true, nil
}
