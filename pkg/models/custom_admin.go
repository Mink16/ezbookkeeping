package models

// CustomAdmin represents a server-wide administrator grant stored in database (fork-local feature)
type CustomAdmin struct {
	Uid             int64 `xorm:"PK"`
	GrantedByUid    int64 `xorm:"NOT NULL"`
	CreatedUnixTime int64
}

// CustomAdminStatusResponse represents a view-object of the current user admin status
type CustomAdminStatusResponse struct {
	IsAdmin bool `json:"isAdmin"`
}

// CustomAdminUserInfoResponse represents a view-object of a user with admin status
type CustomAdminUserInfoResponse struct {
	Uid        int64  `json:"uid,string"`
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	IsAdmin    bool   `json:"isAdmin"`
	IsEnvAdmin bool   `json:"isEnvAdmin"`
}

// CustomAdminGrantRequest represents all parameters of admin granting request
type CustomAdminGrantRequest struct {
	Uid int64 `json:"uid,string" binding:"required,min=1"`
}

// CustomAdminRevokeRequest represents all parameters of admin revoking request
type CustomAdminRevokeRequest struct {
	Uid int64 `json:"uid,string" binding:"required,min=1"`
}
