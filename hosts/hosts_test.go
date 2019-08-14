package hosts

import (
	"reflect"
	"testing"
	"time"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/stretchr/testify/assert"
)

func TestGetPullRequestsToDisplay(t *testing.T) {
	t.Parallel()

	validAge, _ := time.ParseDuration("36h")
	invalidAge, _ := time.ParseDuration("12h")
	maxAge, _ := time.ParseDuration("24h")

	cases := []struct {
		name                    string
		pullRequest             *PullRequest
		readyToMerge            bool
		readyToReview           bool
		numberOfNeededApprovals int
		reviewPRsFromNonMembers bool
	}{
		{
			name: "Not Approved PR",
			pullRequest: &PullRequest{Title: "Not approved", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
				{Approved: false, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			},
				CreateTime: time.Now().Add(-validAge),
			},
			readyToMerge:  false,
			readyToReview: true,
		},
		{
			name: "Approved by other user (not in team)",
			pullRequest: &PullRequest{Title: "Approved by otheruser", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
				{Approved: true, User: config.User{Name: "otheruser"}},
				{Approved: false, User: config.User{Name: "user2"}},
			}},
			readyToMerge:  false,
			readyToReview: true,
		},
		{
			name: "Approved PR",
			pullRequest: &PullRequest{Title: "Approved", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
				{Approved: true, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			}},
			readyToMerge:  true,
			readyToReview: false,
		},
		{
			name: "Author not from team",
			pullRequest: &PullRequest{Title: "User not from team", Author: config.User{Name: "otheruser"}, Reviewers: []*Reviewer{
				{Approved: false, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			}},
			readyToMerge:  false,
			readyToReview: false,
		},
		{
			name: "Author not from team with config to review anyways",
			pullRequest: &PullRequest{Title: "User not from team", Author: config.User{Name: "otheruser"}, Reviewers: []*Reviewer{
				{Approved: false, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			}},
			readyToMerge:            false,
			readyToReview:           true,
			reviewPRsFromNonMembers: true,
		},
		{
			name: "Approved PR not from team",
			pullRequest: &PullRequest{Title: "Approved", Author: config.User{Name: "otheruser"}, Reviewers: []*Reviewer{
				{Approved: true, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			}},
			readyToMerge:  false,
			readyToReview: false,
		},
		{
			name: "Work in progress",
			pullRequest: &PullRequest{Title: "[WIP] My Title", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
				{Approved: false, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			}},
			readyToMerge:  false,
			readyToReview: false,
		},
		{
			name:          "No Reviewers",
			pullRequest:   &PullRequest{Title: "No Reviewers", Author: config.User{Name: "user1"}},
			readyToMerge:  false,
			readyToReview: false,
		},
		{
			name: "Not created long enough ago",
			pullRequest: &PullRequest{Title: "Not approved but too young", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
				{Approved: false, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			},
				CreateTime: time.Now().Add(-invalidAge),
			},
			readyToMerge:  false,
			readyToReview: false,
		},
		{
			name: "Not approved long enough ago (have to wait 24h after update before annoying with merge notification)",
			pullRequest: &PullRequest{Title: "Approved", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
				{Approved: true, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			},
				UpdateTime: time.Now().Add(-invalidAge),
			},
			readyToMerge:  false,
			readyToReview: false,
		},
		{
			name: "Not enough approvals",
			pullRequest: &PullRequest{Title: "Approved", Author: config.User{Name: "user1"}, Reviewers: []*Reviewer{
				{Approved: true, User: config.User{Name: "user1"}},
				{Approved: false, User: config.User{Name: "user2"}},
			}},
			readyToMerge:            false,
			readyToReview:           true,
			numberOfNeededApprovals: 2,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repository := NewRepository(&bitbucketCloud{
				config: &config.TeamConfig{
					AgeBeforeNotifying:      maxAge,
					ReviewPRsFromNonMembers: tt.reviewPRsFromNonMembers,
					NumberOfApprovals:       tt.numberOfNeededApprovals,
					Users: []config.User{
						{Name: "user1", BitbucketUUID: "user1"},
						{Name: "user2", BitbucketUUID: "user2"},
					},
				},
			},
				"repo-name", "http://example.com",
				[]*PullRequest{tt.pullRequest})
			if tt.readyToMerge || tt.readyToReview {
				assert.True(t, repository.HasPullRequestsToDisplay())
			}

			readyToMerge, readyToReview := repository.GetPullRequestsToDisplay()
			assert.Equal(t, tt.readyToMerge, len(readyToMerge) == 1, "The pull request should or should not have been ready to merge")
			assert.Equal(t, tt.readyToReview, len(readyToReview) == 1, "The pull request should or should not have been ready to review")

			assert.Equal(t, "repo-name", repository.GetName())
			assert.Equal(t, "http://example.com", repository.GetLink())
		})
	}
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
