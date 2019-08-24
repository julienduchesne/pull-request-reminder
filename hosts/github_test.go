package hosts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-github/v25/github"
	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/stretchr/testify/assert"
)

func TestGetGithubRepositories(t *testing.T) {
	host := &githubHost{
		client:          &mockGithubClient{},
		repositoryNames: []string{"jdoe/test"},
		config: &config.TeamConfig{
			Users: []config.User{
				{Name: "John Doe", GithubUsername: "jdoe1"},
				{Name: "John Doe2", GithubUsername: "jdoe2"},
				{Name: "John Doe3", GithubUsername: "jdoe3"},
			},
		},
	}

	repositories := host.GetRepositories()

	assert.Len(t, repositories, 1)
	repository := repositories[0]
	assert.Equal(t, "https://github.com/jdoe/test", repository.GetLink())
	assert.Equal(t, host, repository.GetHost())

	assert.True(t, repository.HasPullRequestsToDisplay())
	pullRequestsToMerge, pullRequestsToReview := repository.GetPullRequestsToDisplay()
	assert.Len(t, pullRequestsToMerge, 1)
	assert.Len(t, pullRequestsToReview, 0)

	pullRequest := pullRequestsToMerge[0]
	assert.True(t, pullRequest.IsApproved(host.config.GetGithubUsers(), 1))
	assert.False(t, pullRequest.IsApproved(host.config.GetGithubUsers(), 2)) // only one approval
	assert.False(t, pullRequest.IsWIP())
	assert.True(t, pullRequest.IsFromOneOfUsers(host.config.GetGithubUsers()))
	assert.Len(t, pullRequest.TeamReviewers(host.config.GetGithubUsers()), 2) // jdoe2 and jdoe3
	assert.Equal(t, "jdoe1", pullRequest.Author.GithubUsername)
	assert.Equal(t, "Auto update", pullRequest.Title)
	assert.Equal(t, "https://github.com/coveooss/tgf/pull/79", pullRequest.Link) // directly from the response
}

type mockGithubClient struct{}

func (client *mockGithubClient) ListPullRequests(owner string, repo string, opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	jsonFile, _ := os.Open("responses_test/listpullrequests.json")
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var response []*github.PullRequest
	json.Unmarshal(byteValue, &response)
	return response, nil, nil
}

func (client *mockGithubClient) ListReviews(owner, repo string, number int, opt *github.ListOptions) ([]*github.PullRequestReview, *github.Response, error) {
	var response []*github.PullRequestReview
	if opt.Page == 1 {
		jsonFile, _ := os.Open("responses_test/reviews1.json")
		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &response)
		return response, &github.Response{LastPage: 2}, nil
	} else if opt.Page == 2 {
		jsonFile, _ := os.Open("responses_test/reviews2.json")
		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &response)
		return response, &github.Response{LastPage: 2}, nil
	}
	return nil, nil, fmt.Errorf("Too many pages")
}
