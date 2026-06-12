package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEnvAdminUsernames(t *testing.T) {
	assert.Equal(t, []string{}, ParseEnvAdminUsernames(""))
	assert.Equal(t, []string{"alice"}, ParseEnvAdminUsernames("alice"))
	assert.Equal(t, []string{"alice", "bob"}, ParseEnvAdminUsernames("alice,bob"))
	assert.Equal(t, []string{"alice", "bob"}, ParseEnvAdminUsernames(" alice , bob "))
	assert.Equal(t, []string{"alice", "bob"}, ParseEnvAdminUsernames("alice,,bob,"))
	assert.Equal(t, []string{}, ParseEnvAdminUsernames(" , , "))
}

func TestIsEnvAdminUsername(t *testing.T) {
	usernames := []string{"alice", "bob"}

	assert.True(t, IsEnvAdminUsername(usernames, "alice"))
	assert.True(t, IsEnvAdminUsername(usernames, "bob"))
	assert.False(t, IsEnvAdminUsername(usernames, "carol"))
	assert.False(t, IsEnvAdminUsername(usernames, "Alice")) // exact match, case sensitive
	assert.False(t, IsEnvAdminUsername(usernames, ""))
	assert.False(t, IsEnvAdminUsername([]string{}, "alice"))
	assert.False(t, IsEnvAdminUsername(nil, "alice"))
}
