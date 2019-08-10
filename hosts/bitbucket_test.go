package hosts

import (
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
	repositories := host.GetRepositories()
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

type mockBitbucketClient struct{}

func (mock *mockBitbucketClient) GetPullRequests(owner, slug, id string) (interface{}, error) {
	if id != "" {
		return testGetPullRequestResponse, nil
	}
	return testListPullRequestsResponse, nil
}
