package hosts

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/julienduchesne/pull-request-reminder/config"

	"github.com/stretchr/testify/assert"
)

var testListPullRequestsResponse = map[string]interface{}{
	"values": []map[string]interface{}{
		{"id": 1},
	},
}

var testGetPullRequestResponse = map[string]interface{}{
	"title":       "My Pull Request",
	"description": "My Description",
	"author": map[string]interface{}{
		"uuid": "{jdoe2}",
	},
	"links": map[string]interface{}{
		"html": map[string]interface{}{
			"href": "pr.com",
			"name": "html",
		},
	},
	"participants": []map[string]interface{}{
		{
			"approved": false,
			"role":     "REVIEWER",
			"user": map[string]interface{}{
				"uuid": "{jdoe3}",
			},
		},
	},
	"created_on": "2019-08-08T17:21:52.698243+00:00",
	"updated_on": "2019-08-08T21:12:11.405493+00:00",
}

func TestGetBitbucketRepositories(t *testing.T) {
	t.Parallel()

	utc, _ := time.LoadLocation("UTC")

	host := &bitbucketCloud{
		client:          &mockBitbucketClient{},
		repositoryNames: []string{"jdoe/test"},
		config: &config.TeamConfig{
			Users: []config.User{
				{Name: "John Doe", BitbucketUUID: "{jdoe}"},
				{Name: "John Doe2", BitbucketUUID: "{jdoe2}"},
				{Name: "John Doe3", BitbucketUUID: "{jdoe3}"},
			},
		},
	}
	repositories, err := host.GetRepositories()

	assert.Nil(t, err)
	assert.Len(t, repositories, 1)
	repository := repositories[0].(*RepositoryImpl)
	assert.Equal(t, "jdoe/test", repository.Name)
	assert.Equal(t, "https://bitbucket.org/jdoe/test", repository.Link)
	assert.Equal(t, host, repository.Host)

	assert.Len(t, repository.OpenPullRequests, 1)
	pullRequest := repository.OpenPullRequests[0]
	assert.Equal(t, "John Doe2", pullRequest.Author.Name)
	assert.Equal(t, "My Description", pullRequest.Description)
	assert.Equal(t, "pr.com", pullRequest.Link)
	assert.Equal(t, "My Pull Request", pullRequest.Title)
	assert.Equal(t, time.Date(2019, time.August, 8, 17, 21, 52, 698243000, utc).UTC(), pullRequest.CreateTime.UTC())
	assert.Equal(t, time.Date(2019, time.August, 8, 21, 12, 11, 405493000, utc).UTC(), pullRequest.UpdateTime.UTC())

	assert.Len(t, pullRequest.Reviewers, 1)
	reviewer := pullRequest.Reviewers[0]
	assert.False(t, reviewer.Approved)
	assert.False(t, reviewer.RequestedChanges)
	assert.Equal(t, "John Doe3", reviewer.User.Name)
}

func TestGetBitbucketRepositoriesErrors(t *testing.T) {
	t.Parallel()

	os.Setenv("BITBUCKET_RETRY_DELAY", "0")

	cases := []struct {
		name          string
		client        bitbucketClient
		emptyTeamName bool
		expectError   string
	}{
		{
			name:        "get error",
			client:      &mockBitbucketClient{errorOnGetPullRequest: true},
			expectError: "Caught an error while describing pull requests: Error fetching the pull request with ID 1 from jdoe/test in Bitbucket",
		},
		{
			name:        "list error",
			client:      &mockBitbucketClient{errorOnListPullRequests: true},
			expectError: "Caught an error while describing pull requests: Error fetching pull requests from jdoe/test in Bitbucket",
		},
		{
			name:        "get team members error",
			client:      &mockBitbucketClient{errorOnGettingTeamMembers: true},
			expectError: "Error fetching users from Bitbucket: Error fetching members from team my-team: Get team members error",
		},
		{
			name:          "get team members error",
			client:        &mockBitbucketClient{},
			emptyTeamName: true,
			expectError:   "Error fetching users from Bitbucket: Bitbucket is set to find users in the team but the team name is not set",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			host := &bitbucketCloud{
				client:          tt.client,
				repositoryNames: []string{"jdoe/test"},
				config: &config.TeamConfig{
					Users: []config.User{
						{Name: "John Doe", BitbucketUUID: "{jdoe}"},
					},
				},
			}
			if !tt.emptyTeamName {
				host.config.Hosts.Bitbucket.Team = "my-team"
			}
			host.config.Hosts.Bitbucket.FindUsersInTeam = true
			_, err := host.GetRepositories()
			assert.EqualError(t, err, tt.expectError)
		})
	}
}

func TestGetUsers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		configuredUsers []config.User
		apiResponse     []map[string]interface{}
		expectedUsers   map[string]config.User
		expectError     bool
	}{
		{
			name: "Fully configured",
			configuredUsers: []config.User{
				{Name: "John Doe", BitbucketUUID: "{jdoe}"},
				{Name: "John Doe2", BitbucketUUID: "{jdoe2}"},
				{Name: "John Doe3", BitbucketUUID: ""},
			},
			expectedUsers: map[string]config.User{
				"{jdoe}":  {Name: "John Doe", BitbucketUUID: "{jdoe}"},
				"{jdoe2}": {Name: "John Doe2", BitbucketUUID: "{jdoe2}"},
			},
		},
		{
			name: "Find exact name match",
			configuredUsers: []config.User{
				{Name: "John Doe", BitbucketUUID: ""},
			},
			apiResponse: []map[string]interface{}{
				{
					"display_name": "John Doe",
					"uuid":         "{jdoe}",
				},
			},
			expectedUsers: map[string]config.User{
				"{jdoe}": {Name: "John Doe", BitbucketUUID: "{jdoe}"},
			},
		},
		{
			name: "Find name that almost matches with dashes",
			configuredUsers: []config.User{
				{Name: "John Master-Doe", BitbucketUUID: ""},
			},
			apiResponse: []map[string]interface{}{
				{
					"display_name": "John master Doe",
					"uuid":         "{jdoe}",
				},
			},
			expectedUsers: map[string]config.User{
				"{jdoe}": {Name: "John Master-Doe", BitbucketUUID: "{jdoe}"},
			},
		},
		{
			name: "Find name with diacritics",
			configuredUsers: []config.User{
				{Name: "Çämûèl Lévesque", BitbucketUUID: ""},
			},
			apiResponse: []map[string]interface{}{
				{
					"display_name": "Camuèl Levesque",
					"uuid":         "{clevesque}",
				},
			},
			expectedUsers: map[string]config.User{
				"{clevesque}": {Name: "Çämûèl Lévesque", BitbucketUUID: "{clevesque}"},
			},
		},
		{
			name: "Multiple matches",
			configuredUsers: []config.User{
				{Name: "John Doe", BitbucketUUID: ""},
			},
			apiResponse: []map[string]interface{}{
				{
					"display_name": "John Doe",
					"uuid":         "{jdoe}",
				},
				{
					"display_name": "John Doe",
					"uuid":         "{jdoe2}",
				},
			},
			expectError: true,
		},
		{
			name: "Multiple users with same UUID",
			configuredUsers: []config.User{
				{Name: "John Doe", BitbucketUUID: ""},
				{Name: "Jane Doe", BitbucketUUID: "{jdoe}"},
			},
			apiResponse: []map[string]interface{}{
				{
					"display_name": "John Doe",
					"uuid":         "{jdoe}",
				},
			},
			expectError: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			host := &bitbucketCloud{
				client: &mockBitbucketClient{
					getTeamResponse: tt.apiResponse,
				},
				repositoryNames: []string{"jdoe/test"},
				config: &config.TeamConfig{
					Users: tt.configuredUsers,
				},
			}
			host.config.Hosts.Bitbucket.Team = "Anything"
			host.config.Hosts.Bitbucket.FindUsersInTeam = tt.apiResponse != nil
			users, err := host.GetUsers()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expectedUsers, users)
		})
	}

}

type mockBitbucketClient struct {
	errorOnGetPullRequest     bool
	errorOnListPullRequests   bool
	errorOnGettingTeamMembers bool
	getTeamResponse           []map[string]interface{}
}

func (mock *mockBitbucketClient) GetPullRequests(owner, slug, id string) (interface{}, error) {
	if id != "" {
		if mock.errorOnGetPullRequest {
			return nil, fmt.Errorf("get error")
		}
		return testGetPullRequestResponse, nil
	}
	if mock.errorOnListPullRequests {
		return nil, fmt.Errorf("list error")
	}
	return testListPullRequestsResponse, nil
}

func (mock *mockBitbucketClient) GetTeamMembers(team string) (interface{}, error) {
	if mock.errorOnGettingTeamMembers {
		return nil, fmt.Errorf("Get team members error")
	}
	return map[string]interface{}{"values": mock.getTeamResponse}, nil
}
