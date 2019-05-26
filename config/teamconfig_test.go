package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyTeamConfig(t *testing.T) {
	t.Parallel()

	config := &TeamConfig{Users: []User{
		{BitbucketUUID: "", GithubUsername: ""},
	}}
	assert.False(t, config.IsBitbucketConfigured())
	assert.Empty(t, config.GetBitbucketUsers())
	assert.False(t, config.IsGithubConfigured())
	assert.Empty(t, config.GetGithubUsers())
}

func TestBitbucketTeamConfig(t *testing.T) {
	t.Parallel()

	config := &TeamConfig{Users: []User{
		{BitbucketUUID: "{test}", GithubUsername: ""},
	}}
	assert.Equal(t, map[string]User{"{test}": {BitbucketUUID: "{test}", GithubUsername: ""}}, config.GetBitbucketUsers())
	assert.False(t, config.IsBitbucketConfigured())

	config.Hosts.Bitbucket = BitbucketConfig{
		Username:     "test",
		Password:     "test",
		Repositories: []string{"test"},
	}
	assert.True(t, config.IsBitbucketConfigured())
}

func TestGithubTeamConfig(t *testing.T) {
	t.Parallel()

	config := &TeamConfig{Users: []User{
		{BitbucketUUID: "", GithubUsername: "test"},
	}}
	assert.Equal(t, map[string]User{"test": {BitbucketUUID: "", GithubUsername: "test"}}, config.GetGithubUsers())
	assert.False(t, config.IsGithubConfigured())

	config.Hosts.Github = GithubConfig{
		Token:        "test",
		Repositories: []string{"test"},
	}
	assert.True(t, config.IsGithubConfigured())
}
