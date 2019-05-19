package messages

import (
	"fmt"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/nlopes/slack"
)

type slackMessageHandler struct {
	channel string
	client  *slack.Client
}

func (handler *slackMessageHandler) Notify(repositoriesNeedingAction []*hosts.Repository) {
	headerText := slack.NewTextBlockObject("plain_text", "Hello, here are the pull requests requiring your attention today:", false, false)

	sections := []slack.Block{slack.NewSectionBlock(headerText, nil, nil)}
	for _, repository := range repositoriesNeedingAction {

		titleBlock := slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("[%v] *<%v|%v>*", repository.Host.GetName(), repository.Link, repository.Name), false, false),
			nil, nil,
		)
		sections = append(sections,
			slack.NewDividerBlock(),
			titleBlock,
		)

		var addPullRequestSections = func(title string, pullRequests []*hosts.PullRequest) {
			if len(pullRequests) == 0 {
				return
			}
			pullRequestTitle := slack.NewSectionBlock(
				slack.NewTextBlockObject("plain_text", title, true, false),
				nil, nil,
			)
			sections = append(sections, pullRequestTitle)
			for _, pr := range pullRequests {
				pullRequestBlock := slack.NewSectionBlock(
					slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("<%v|%v>", pr.Link, pr.Title), false, false),
					nil, nil,
				)
				sections = append(sections, pullRequestBlock)
			}
		}

		addPullRequestSections(":heavy_check_mark: Pull requests awaiting merge", repository.ReadyToMergePullRequests)
		addPullRequestSections(":no_entry: Pull requests still in need of approvers", repository.ReadyToReviewPullRequests)

	}

	if _, _, err := handler.client.PostMessage(handler.channel, slack.MsgOptionAsUser(true), slack.MsgOptionBlocks(sections...)); err != nil {
		panic(err)
	}
}

func newSlackMessageHandler(config *config.TeamConfig) *slackMessageHandler {
	return &slackMessageHandler{
		channel: config.Slack.Channel,
		client:  slack.New(config.Slack.Token),
	}
}