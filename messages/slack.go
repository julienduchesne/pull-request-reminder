package messages

import (
	"fmt"
	"os"

	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
)

// [
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "Hello, here are the pull requests requiring your attention today:"
// 		}
// 	},
// 	{
// 		"type": "divider"
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "[Bitbucket] *<https://google.com|Repository name>*"
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "plain_text",
// 			"text": ":heavy_check_mark: Pull requests awaiting merge",
// 			"emoji": true
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "<https://google.com|Pull request title>\nPull request description"
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "<https://google.com|Pull request title>\nPull request description"
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "plain_text",
// 			"text": ":no_entry: Pull requests still in need of approvers",
// 			"emoji": true
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "<https://google.com|Pull request title>\nPull request description"
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "<https://google.com|Pull request title>\nPull request description"
// 		}
// 	},
// 	{
// 		"type": "divider"
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "[Bitbucket] *<https://google.com|Repository name>*"
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "plain_text",
// 			"text": ":heavy_check_mark: Pull requests awaiting merge",
// 			"emoji": true
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "<https://google.com|Pull request title>\nPull request description"
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "<https://google.com|Pull request title>\nPull request description"
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "plain_text",
// 			"text": ":no_entry: Pull requests still in need of approvers",
// 			"emoji": true
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "<https://google.com|Pull request title>\nPull request description"
// 		}
// 	},
// 	{
// 		"type": "section",
// 		"text": {
// 			"type": "mrkdwn",
// 			"text": "<https://google.com|Pull request title>\nPull request description"
// 		}
// 	}
// ]

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

func newSlackMessageHandler() *slackMessageHandler {
	slackToken := os.Getenv("SLACK_TOKEN")
	slackChannel := os.Getenv("SLACK_CHANNEL")

	if slackToken == "" || slackChannel == "" {
		log.Infoln("SLACK_TOKEN and SLACK_CHANNEL must be defined to handle slack")
		return nil
	}
	return &slackMessageHandler{
		channel: slackChannel,
		client:  slack.New(slackToken),
	}
}
