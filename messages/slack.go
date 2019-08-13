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
	channel      string
	messageUsers bool
	client       slackClient
}

func (handler *slackMessageHandler) Notify(repositoriesNeedingAction []hosts.Repository) error {
	if handler.channel != "" {
		sections := buildChannelSlackMessage(repositoriesNeedingAction)
		if _, _, err := handler.client.PostMessage(handler.channel, slack.MsgOptionAsUser(true), slack.MsgOptionBlocks(sections...)); err != nil {
			return err
		}
	}

	if handler.messageUsers {
		for user, sections := range buildUserSlackMessages(repositoriesNeedingAction) {
			if _, _, err := handler.client.PostMessage(user, slack.MsgOptionAsUser(true), slack.MsgOptionBlocks(sections...)); err != nil {
				return err
			}
		}
	}

	return nil
}

func newSlackMessageHandler(config *config.TeamConfig) *slackMessageHandler {
	slackConfig := config.Messaging.Slack
	return &slackMessageHandler{
		channel:      slackConfig.Channel,
		messageUsers: slackConfig.MessageUsersIndividually,
		client:       slack.New(slackConfig.Token),
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

		readyToMerge, readyToReview := repository.GetPullRequestsToDisplay()
		sections = append(sections, getPullRequestSections(":heavy_check_mark: Pull requests awaiting merge", true, readyToMerge)...)
		sections = append(sections, getPullRequestSections(":no_entry: Pull requests still in need of approvers", false, readyToReview)...)
	}

	return sections
}

func buildUserSlackMessages(repositoriesNeedingAction []hosts.Repository) map[string][]slack.Block {
	headerTextBlock := slack.NewTextBlockObject("plain_text", headerText, false, false)

	messagePerUser := map[string][]slack.Block{}
	for _, repository := range repositoriesNeedingAction {
		readyToMerge, readyToReview := repository.GetPullRequestsToDisplay()
		readyToMergeByUser, readyToReviewByUser := map[string][]*hosts.PullRequest{}, map[string][]*hosts.PullRequest{}
		for _, pullRequest := range readyToMerge {
			author := pullRequest.Author.SlackUsername
			readyToMergeByUser[author] = append(readyToMergeByUser[author], pullRequest)
		}
		for _, pullRequest := range readyToReview {
			for _, reviewer := range pullRequest.Reviewers {
				username := reviewer.User.SlackUsername
				if !reviewer.Approved {
					readyToReviewByUser[username] = append(readyToReviewByUser[username], pullRequest)
				}
			}
		}

		usersInit := map[string]bool{}
		var initRepositoryMessage = func(user string) {
			if _, ok := messagePerUser[user]; !ok {
				messagePerUser[user] = []slack.Block{slack.NewSectionBlock(headerTextBlock, nil, nil)}
			}
			if !usersInit[user] {
				titleBlock := slack.NewSectionBlock(
					slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("[%v] *<%v|%v>*", repository.GetHost().GetName(), repository.GetLink(), repository.GetName()), false, false),
					nil, nil,
				)
				messagePerUser[user] = append(messagePerUser[user],
					slack.NewDividerBlock(),
					titleBlock,
				)
				usersInit[user] = true
			}

		}

		for user, pullRequests := range readyToMergeByUser {
			initRepositoryMessage(user)
			messagePerUser[user] = append(messagePerUser[user], getPullRequestSections(":heavy_check_mark: Pull requests awaiting merge", false, pullRequests)...)
		}
		for user, pullRequests := range readyToReviewByUser {
			initRepositoryMessage(user)
			messagePerUser[user] = append(messagePerUser[user], getPullRequestSections(":no_entry: Pull requests still in need of approvers", false, pullRequests)...)
		}

	}

	return messagePerUser
}

func getPullRequestSections(title string, linkAuthor bool, pullRequests []*hosts.PullRequest) []slack.Block {
	sections := []slack.Block{}
	if len(pullRequests) == 0 {
		return sections
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
	return sections
}
