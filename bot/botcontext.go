package bot

import "github.com/slack-go/slack"

type ContextHandler interface {
	HandleMessage(event *slack.MessageEvent) bool
}

type ContextHandlerFunc func(event *slack.MessageEvent) bool

func (f ContextHandlerFunc) HandleMessage(event *slack.MessageEvent) bool {
	return f(event)
}
