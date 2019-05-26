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

func (handler *slackMessageHandler) Notify(repositoriesNeedingAction []*hosts.Repository) error {
	sections := buildSlackMessage(repositoriesNeedingAction)
	if _, _, err := handler.client.PostMessage(handler.channel, slack.MsgOptionAsUser(true), slack.MsgOptionBlocks(sections...)); err != nil {
		return err
	}

	return nil
}

func newSlackMessageHandler(config *config.TeamConfig) *slackMessageHandler {
	slackConfig := config.Messaging.Slack
	return &slackMessageHandler{
		channel: slackConfig.Channel,
		client:  slack.New(slackConfig.Token),
	}
}

func buildSlackMessage(repositoriesNeedingAction []*hosts.Repository) []slack.Block {
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

		var addPullRequestSections = func(title string, linkAuthor bool, pullRequests []*hosts.PullRequest) {
			if len(pullRequests) == 0 {
				return
			}
			pullRequestTitle := slack.NewSectionBlock(
				slack.NewTextBlockObject("plain_text", title, true, false),
				nil, nil,
			)
			sections = append(sections, pullRequestTitle)
			for _, pr := range pullRequests {
				text := fmt.Sprintf("<%v|%v>", pr.Link, pr.Title)
				if linkAuthor {
					text = fmt.Sprintf("@%s: %s", pr.Author.SlackUsername, text)
				}
				pullRequestBlock := slack.NewSectionBlock(
					slack.NewTextBlockObject("mrkdwn", text, false, false),
					nil, nil,
				)
				sections = append(sections, pullRequestBlock)
			}
		}

		addPullRequestSections(":heavy_check_mark: Pull requests awaiting merge", true, repository.ReadyToMergePullRequests)
		addPullRequestSections(":no_entry: Pull requests still in need of approvers", false, repository.ReadyToReviewPullRequests)

	}

	return sections
}
