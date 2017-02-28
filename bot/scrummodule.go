package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

// HandleMessage handle a received message for scrums and returns if the bot shall continue to process the message or stop
// continue = true
// stop = false
func (b *Bot) HandleScrumMessage(event *slack.MessageEvent) bool {
	// "start scrum [team] [date]"
	// [team == first and only team]
	// [date == yesterday]
	// starting scrum for team [team] date [date]. if you want to abort say stop scrum

	// this module only takes case in private messages
	if event.Channel[0] != 'D' {
		return true
	}

	if context, ok := b.userContexts[event.User]; ok {
		return context.HandleMessage(event)
	}

	if event.Text == "create team" {
		b.createTeam(event)
		return false
	}

	if strings.HasPrefix(strings.ToLower(event.Text), "start scrum") {
		b.startScrum(event)
		return false
	}

	if event.Text == "help" {
		b.scrumHelp(event)
		return true
	}
	return true
}

func (b *Bot) RunScrumModule() {}

func (b *Bot) scrumHelp(event *slack.MessageEvent) {
	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "Here's a list of supported commands for scrums", slack.PostMessageParameters{
		AsUser: true,
		Attachments: []slack.Attachment{slack.Attachment{
			Text: `- "create team" -> start the creation of a new team `,
		}},
	})
	if err != nil {
		b.logger.Printf("%s\n", err)
		return
	}
}

func (b *Bot) startScrum(event *slack.MessageEvent) {
	b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		// can we infer team (aka does the user only has one team)
		// can we infer date (aka before the next scrum)

		team := ""
		date := time.Now()
		message := fmt.Sprintf("Started scrum report for %s on %s", team, date.Format(time.RFC3339))

		_, _, err := b.slackBotAPI.PostMessage(event.Channel, message, slack.PostMessageParameters{AsUser: true})
		if err != nil {
			b.logger.Printf("%s\n", err)
			return false
		}
		return false
	})
}
