package messages

import (
	reflect "reflect"
	"testing"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/stretchr/testify/assert"
)

func TestGetSlackMessageHandler(t *testing.T) {
	t.Parallel()

	teamConfig := &config.TeamConfig{}
	teamConfig.Messaging.Slack = config.SlackConfig{
		Channel:                  "#my-channel",
		MessageUsersIndividually: true,
		Token:                    "xoxb-stuff",
		DebugUser:                "@admin",
	}

	hasType := false
	for _, handler := range GetHandlers(teamConfig) {

		if slackHandler, ok := handler.(*slackMessageHandler); ok {
			hasType = true
			assert.Equal(t, "#my-channel", slackHandler.channel)
			assert.True(t, slackHandler.messageUsers)
			assert.Equal(t, "@admin", slackHandler.debugUser)
			assert.NotNil(t, slackHandler.client)
		}
	}
	assert.True(t, hasType, "There should be a handler of type: %v", reflect.TypeOf(&slackMessageHandler{}))
}
