package errs

import "net/http"

// NormalSubcategoryCustomAdmin is a fork-local error subcategory for the custom server admin features
// (upstream uses subcategories 0-19, this fork reserves 90-99 to avoid collision; 90 is used by custom llm prompts)
const NormalSubcategoryCustomAdmin = 91

// Error codes related to custom server admins
var (
	ErrCustomAdminPermissionRequired   = NewNormalError(NormalSubcategoryCustomAdmin, 0, http.StatusForbidden, "server admin permission required")
	ErrCustomAdminUserNotFound         = NewNormalError(NormalSubcategoryCustomAdmin, 1, http.StatusBadRequest, "target user not found")
	ErrCustomAdminAlreadyGranted       = NewNormalError(NormalSubcategoryCustomAdmin, 2, http.StatusBadRequest, "user is already an administrator")
	ErrCustomAdminNotGranted           = NewNormalError(NormalSubcategoryCustomAdmin, 3, http.StatusBadRequest, "user is not an administrator")
	ErrCustomAdminCannotRevokeEnvAdmin = NewNormalError(NormalSubcategoryCustomAdmin, 4, http.StatusBadRequest, "cannot revoke administrator configured in server settings")
	ErrCustomAdminCannotRevokeSelf     = NewNormalError(NormalSubcategoryCustomAdmin, 5, http.StatusBadRequest, "cannot revoke your own administrator permission")
)
