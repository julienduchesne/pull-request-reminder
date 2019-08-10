package hosts

import (
	"reflect"
	"testing"
	"time"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/stretchr/testify/assert"
)

func TestCategorizePullRequests(t *testing.T) {
	t.Parallel()

	validAge, _ := time.ParseDuration("36h")
	invalidAge, _ := time.ParseDuration("12h")
	maxAge, _ := time.ParseDuration("24h")

	approvedByOtherUserPR := &PullRequest{Title: "Approved by otheruser", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
		{Approved: true, User: config.User{Name: "otheruser"}},
		{Approved: false, User: config.User{Name: "user2"}},
	}}
	notApprovedPR := &PullRequest{Title: "Not approved", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
		{Approved: false, User: config.User{Name: "user1"}},
		{Approved: false, User: config.User{Name: "user2"}},
	},
		CreateTime: time.Now().Add(-validAge),
	}
	notApprovedPRButTooYoung := &PullRequest{Title: "Not approved but too young", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
		{Approved: false, User: config.User{Name: "user1"}},
		{Approved: false, User: config.User{Name: "user2"}},
	},
		CreateTime: time.Now().Add(-invalidAge),
	}
	approvedPR := &PullRequest{Title: "Approved", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
		{Approved: true, User: config.User{Name: "user1"}},
		{Approved: false, User: config.User{Name: "user2"}},
	}}

	openPullRequests := []*PullRequest{
		{Title: "User not from team", Author: config.User{Name: "otheruser"}, Reviewers: []*Reviewer{
			{Approved: false, User: config.User{Name: "user1"}},
			{Approved: false, User: config.User{Name: "user2"}},
		}},
		{Title: "[WIP] My Title", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
			{Approved: false, User: config.User{Name: "user1"}},
			{Approved: false, User: config.User{Name: "user2"}},
		}},
		{Title: "No Reviewers", Author: config.User{Name: "user1"}},
		notApprovedPRButTooYoung,
		approvedByOtherUserPR,
		notApprovedPR,
		approvedPR,
	}

	repository := NewRepository(&bitbucketCloud{
		config: &config.TeamConfig{
			AgeBeforeNotifying: maxAge,
			Users: []config.User{
				{Name: "user1", BitbucketUUID: "user1"},
				{Name: "user2", BitbucketUUID: "user2"},
			},
		},
	},
		"repo-name", "http://example.com",
		openPullRequests)

	assert.True(t, repository.HasPullRequestsToDisplay())
	readyToMerge, readyToReview := repository.GetPullRequestsToDisplay()
	assert.Len(t, readyToMerge, 1)
	assert.Contains(t, readyToMerge, approvedPR)

	assert.Len(t, readyToReview, 2)
	assert.Contains(t, readyToReview, notApprovedPR)
	assert.Contains(t, readyToReview, approvedByOtherUserPR)
}

func TestGetHosts(t *testing.T) {
	t.Parallel()

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
		teamConfig.Hosts.Bitbucket = config.BitbucketConfig{
			Username:     "user",
			Password:     "pass",
			Repositories: []string{"repo"},
		}
		teamConfig.Users = append(teamConfig.Users, config.User{
			Name:          "John Doe2",
			BitbucketUUID: "{jdoe2}",
		})
	}
	if withGithub {
		teamConfig.Hosts.Github = config.GithubConfig{
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
