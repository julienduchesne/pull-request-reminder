package messages

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

func TestBuildSlackMessageWithSingleRepo(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHost := hosts.NewMockHost(ctrl)
	mockHost.EXPECT().GetName().Return("mock").AnyTimes()

	mockRepository := hosts.NewMockRepository(ctrl)
	mockRepository.EXPECT().GetHost().Return(mockHost).AnyTimes()
	mockRepository.EXPECT().GetLink().Return("mock-repo.com").AnyTimes()
	mockRepository.EXPECT().GetName().Return("mock-repo").AnyTimes()
	mockRepository.EXPECT().GetPullRequestsToDisplay().Return(
		[]*hosts.PullRequest{
			{
				Title: "pr1",
				Link:  "link1.com",
			},
		},
		[]*hosts.PullRequest{
			{
				Title: "pr2",
				Link:  "link2.com",
			},
			{
				Title: "pr3",
				Link:  "link3.com",
			},
		}).AnyTimes()

	repositories := []hosts.Repository{mockRepository}

	sections := buildChannelSlackMessage(repositories)
	assert.Len(t, sections, 8)
	// 1. Main Title
	assert.Equal(t, "Hello, here are the pull requests requiring your attention today:", sections[0].(*slack.SectionBlock).Text.Text)
	// 2. Divider
	// 3. Repository Title
	// 4. PRs waiting for review Title
	// 5. PR 2
	// 6. PR 3
	// 7. Ready to Merge Title
	// 8. PR1

}
