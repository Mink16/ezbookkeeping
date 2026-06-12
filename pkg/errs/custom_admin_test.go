package errs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomAdminErrorCodes(t *testing.T) {
	// the fork-local subcategory 91 must keep error codes in the 291xxx range, far from upstream codes (up to 219xxx)
	assert.Equal(t, int32(291000), ErrCustomAdminPermissionRequired.Code())
	assert.Equal(t, int32(291001), ErrCustomAdminUserNotFound.Code())
	assert.Equal(t, int32(291002), ErrCustomAdminAlreadyGranted.Code())
	assert.Equal(t, int32(291003), ErrCustomAdminNotGranted.Code())
	assert.Equal(t, int32(291004), ErrCustomAdminCannotRevokeEnvAdmin.Code())
	assert.Equal(t, int32(291005), ErrCustomAdminCannotRevokeSelf.Code())
}
