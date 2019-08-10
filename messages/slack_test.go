package messages

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

func TestBuildChannelSlackMessageWithSingleRepo(t *testing.T) {
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
	assert.IsType(t, sections[1], &slack.DividerBlock{})
	// 3. Repository Title
	assert.Equal(t, "[mock] *<mock-repo.com|mock-repo>*", sections[2].(*slack.SectionBlock).Text.Text)
	// 4. PRs waiting for review Title
	assert.Equal(t, ":heavy_check_mark: Pull requests awaiting merge", sections[3].(*slack.SectionBlock).Text.Text)
	assert.Equal(t, "<link1.com|pr1>", sections[4].(*slack.SectionBlock).Text.Text)
	// 5. Ready to Merge Title
	assert.Equal(t, ":no_entry: Pull requests still in need of approvers", sections[5].(*slack.SectionBlock).Text.Text)
	assert.Equal(t, "<link2.com|pr2>", sections[6].(*slack.SectionBlock).Text.Text)
	assert.Equal(t, "<link3.com|pr3>", sections[7].(*slack.SectionBlock).Text.Text)

}
