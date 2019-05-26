package messages

import (
	"testing"

	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/stretchr/testify/assert"
)

func TestBuildSlackMessageWithSingleRepo(t *testing.T) {
	t.Parallel()

	repositories := []*hosts.Repository{
		{Host: &mockHost{}, Name: "mock-repo", Link: "mock-repo.com",
			ReadyToMergePullRequests: []*hosts.PullRequest{
				{
					Title: "pr1",
					Link:  "link1.com",
				},
			},
			ReadyToReviewPullRequests: []*hosts.PullRequest{
				{
					Title: "pr2",
					Link:  "link2.com",
				},
				{
					Title: "pr3",
					Link:  "link3.com",
				},
			},
		},
	}

	sections := buildSlackMessage(repositories)
	assert.Len(t, sections, 8)
	// 1. Main Title
	// 2. Divider
	// 3. Repository Title
	// 4. PRs waiting for review Title
	// 5. PR 2
	// 6. PR 3
	// 7. Ready to Merge Title
	// 8. PR1

}

type mockHost struct{}

func (host *mockHost) GetName() string {
	return "mock"
}

func (host *mockHost) GetUsers() []string {
	return nil
}

func (host *mockHost) GetRepositories() []*hosts.Repository {
	return nil
}
