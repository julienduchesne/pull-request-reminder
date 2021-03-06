package messages

import (
	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/julienduchesne/pull-request-reminder/hosts"
)

// MessageHandler is the interface that wraps the Notify method.
// This method sends a message concerning the pull requests to a messaging provider
type MessageHandler interface {
	Notify([]hosts.Repository) error
}

// GetHandlers returns all available and configured MessageHandler instances
func GetHandlers(config *config.TeamConfig) []MessageHandler {
	handlers := []MessageHandler{newSlackMessageHandler(config)}
	return handlers
}
