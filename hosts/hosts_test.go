package hosts

import (
	"reflect"
	"testing"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/stretchr/testify/assert"
)

func TestCategorizePullRequests(t *testing.T) {

}

func TestGetHosts(t *testing.T) {
	cases := []struct {
		name          string
		config        *config.TeamConfig
		expectedHosts []reflect.Type
	}{
		{
			name:          "No config",
			config:        &config.TeamConfig{},
			expectedHosts: []reflect.Type{},
		},
		{
			name:   "With bitbucket config",
			config: getTeamConfig(true, false),
			expectedHosts: []reflect.Type{
				reflect.TypeOf(&bitbucketCloud{}),
			},
		},
		{
			name:   "With github config",
			config: getTeamConfig(false, true),
			expectedHosts: []reflect.Type{
				reflect.TypeOf(&githubHost{}),
			},
		},
		{
			name:   "With github and bitbucket config",
			config: getTeamConfig(true, true),
			expectedHosts: []reflect.Type{
				reflect.TypeOf(&bitbucketCloud{}),
				reflect.TypeOf(&githubHost{}),
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gottenHosts := GetHosts(tt.config)
			for _, hostType := range tt.expectedHosts {
				hasType := false
				for _, host := range gottenHosts {
					if reflect.TypeOf(host) == hostType {
						hasType = true
					}
				}
				assert.True(t, hasType, "There should be a host of type: %v", hostType)
			}
			assert.Equal(t, len(tt.expectedHosts), len(gottenHosts), "There are not the same amount of expected and gotten hosts")
		})
	}
}

func getTeamConfig(withBitbucket bool, withGithub bool) *config.TeamConfig {
	teamConfig := &config.TeamConfig{
		Users: []config.User{},
	}
	if withBitbucket {
		teamConfig.Bitbucket = config.BitbucketConfig{
			Username:     "user",
			Password:     "pass",
			Repositories: []string{"repo"},
		}
		teamConfig.Users = append(teamConfig.Users, config.User{
			Name:              "John Doe2",
			BitbucketUsername: "jdoe2",
		})
	}
	if withGithub {
		teamConfig.Github = config.GithubConfig{
			Token:        "token",
			Repositories: []string{"repo"},
		}
		teamConfig.Users = append(teamConfig.Users, config.User{
			Name:           "John Doe2",
			GithubUsername: "jdoe2",
		})
	}
	return teamConfig
}
