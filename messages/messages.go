package messages

import "github.com/julienduchesne/pull-request-reminder/hosts"

type MessageHandler interface {
	Notify([]*hosts.Repository)
}

func GetHandlers() []MessageHandler {
	handlers := []MessageHandler{}
	if slackHandler := newSlackMessageHandler(); slackHandler != nil {
		handlers = append(handlers, slackHandler)
	}
	return handlers
}
