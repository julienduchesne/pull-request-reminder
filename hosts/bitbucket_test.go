package hosts

import (
	"testing"

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
}

func TestGetBitbucketRepositories(t *testing.T) {
	host := &bitbucketCloud{
		client:          &mockBitbucketClient{},
		repositoryNames: []string{"jdoe/test"},
		users: map[string]config.User{
			"{jdoe}":  {Name: "John Doe"},
			"{jdoe2}": {Name: "John Doe2"},
			"{jdoe3}": {Name: "John Doe3"},
		},
	}
	repositories := host.GetRepositories()
	assert.Len(t, repositories, 1)
	repository := repositories[0]
	assert.Equal(t, "jdoe/test", repository.Name)
	assert.Equal(t, "https://bitbucket.org/jdoe/test", repository.Link)
	assert.Equal(t, host, repository.Host)

	assert.Len(t, repository.OpenPullRequests, 1)
	pullRequest := repository.OpenPullRequests[0]
	assert.Equal(t, config.User{Name: "John Doe2"}, pullRequest.Author)
	assert.Equal(t, "My Description", pullRequest.Description)
	assert.Equal(t, "pr.com", pullRequest.Link)
	assert.Equal(t, "My Pull Request", pullRequest.Title)

	assert.Len(t, pullRequest.Reviewers, 1)
	reviewer := pullRequest.Reviewers[0]
	assert.False(t, reviewer.Approved)
	assert.False(t, reviewer.RequestedChanges)
	assert.Equal(t, config.User{Name: "John Doe3"}, reviewer.User)
}

type mockBitbucketClient struct{}

func (mock *mockBitbucketClient) GetPullRequests(owner, slug, id string) (interface{}, error) {
	if id != "" {
		return testGetPullRequestResponse, nil
	}
	return testListPullRequestsResponse, nil
}
