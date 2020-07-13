package bot

import "github.com/slack-go/slack"

type BotContextHandler interface {
	HandleMessage(event *slack.MessageEvent) bool
}

type BotContextHandlerFunc func(event *slack.MessageEvent) bool

func (f BotContextHandlerFunc) HandleMessage(event *slack.MessageEvent) bool {
	return f(event)
}
