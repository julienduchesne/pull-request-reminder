package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUsers(t *testing.T) {
	cases := []struct {
		name                   string
		config                 *TeamConfig
		expectedBitbucketUsers []string
		expectedGithubUsers    []string
	}{
		{
			name:                   "No users",
			config:                 &TeamConfig{},
			expectedBitbucketUsers: []string{},
			expectedGithubUsers:    []string{},
		},
		{
			name: "No bitbucket users",
			config: &TeamConfig{Users: []User{
				{BitbucketUsername: "", GithubUsername: "test"},
			}},
			expectedBitbucketUsers: []string{},
			expectedGithubUsers:    []string{"test"},
		},
		{
			name: "No github users",
			config: &TeamConfig{Users: []User{
				{BitbucketUsername: "test", GithubUsername: ""},
			}},
			expectedBitbucketUsers: []string{"test"},
			expectedGithubUsers:    []string{},
		},
		{
			name: "Bitbucket and github",
			config: &TeamConfig{Users: []User{
				{BitbucketUsername: "test1"},
				{BitbucketUsername: "", GithubUsername: "test2"},
				{BitbucketUsername: "test3"},
			}},
			expectedBitbucketUsers: []string{"test1", "test3"},
			expectedGithubUsers:    []string{"test2"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.config.GetBitbucketUsers(), tt.expectedBitbucketUsers)
			assert.Equal(t, tt.config.GetGithubUsers(), tt.expectedGithubUsers)
		})
	}
}
