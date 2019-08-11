package messages

import (
	"fmt"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/nlopes/slack"
)

const headerText = "Hello, here are the pull requests requiring your attention today:"

type slackClient interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

type slackMessageHandler struct {
	channel string
	client  slackClient
}

func (handler *slackMessageHandler) Notify(repositoriesNeedingAction []hosts.Repository) error {
	if handler.channel != "" {
		sections := buildChannelSlackMessage(repositoriesNeedingAction)
		if _, _, err := handler.client.PostMessage(handler.channel, slack.MsgOptionAsUser(true), slack.MsgOptionBlocks(sections...)); err != nil {
			return err
		}
	}

	for user, sections := range buildUserSlackMessages(repositoriesNeedingAction) {
		if _, _, err := handler.client.PostMessage(user, slack.MsgOptionAsUser(true), slack.MsgOptionBlocks(sections...)); err != nil {
			return err
		}
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

func buildChannelSlackMessage(repositoriesNeedingAction []hosts.Repository) []slack.Block {
	headerTextBlock := slack.NewTextBlockObject("plain_text", headerText, false, false)

	sections := []slack.Block{slack.NewSectionBlock(headerTextBlock, nil, nil)}
	for _, repository := range repositoriesNeedingAction {

		titleBlock := slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("[%v] *<%v|%v>*", repository.GetHost().GetName(), repository.GetLink(), repository.GetName()), false, false),
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
				if linkAuthor && pr.Author.SlackUsername != "" {
					text = fmt.Sprintf("@%s: %s", pr.Author.SlackUsername, text)
				}
				pullRequestBlock := slack.NewSectionBlock(
					slack.NewTextBlockObject("mrkdwn", text, false, false),
					nil, nil,
				)
				sections = append(sections, pullRequestBlock)
			}
		}

		readyToMerge, readyToReview := repository.GetPullRequestsToDisplay()
		addPullRequestSections(":heavy_check_mark: Pull requests awaiting merge", true, readyToMerge)
		addPullRequestSections(":no_entry: Pull requests still in need of approvers", false, readyToReview)

	}

	return sections
}

func buildUserSlackMessages(repositoriesNeedingAction []hosts.Repository) map[string][]slack.Block {
	messagePerUser := map[string][]slack.Block{}
	return messagePerUser
}
