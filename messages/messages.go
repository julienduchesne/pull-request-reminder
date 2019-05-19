package messages

import (
	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/julienduchesne/pull-request-reminder/hosts"
)

type MessageHandler interface {
	Notify([]*hosts.Repository)
}

func GetHandlers(config *config.TeamConfig) []MessageHandler {
	handlers := []MessageHandler{}
	if slackHandler := newSlackMessageHandler(config); slackHandler != nil {
		handlers = append(handlers, slackHandler)
	}
	return handlers
}
