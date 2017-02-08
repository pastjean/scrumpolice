package bot

import (
	"fmt"

	"github.com/nlopes/slack"
	"github.com/scrumpolice/scrumpolice"
)

func (b *Bot) createTeam(event *slack.MessageEvent) {
	b.slackBotAPI.PostMessage(event.Channel, "Creating a new team, what will be the team name ? (you can cancel anytime by typing `quit`)", slack.PostMessageParameters{AsUser: true})
	b.userContexts[event.User] = b.createTeamChooseTeamName()
}

func (b *Bot) createTeamChooseTeamName() BotContextHandler {
	return b.canQuitBotContext(BotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		teamName := event.Text
		b.slackBotAPI.PostMessage(event.Channel, fmt.Sprintf("the team `%s` will be created, in what channel do they operate?", teamName), slack.PostMessageParameters{AsUser: true})

		b.userContexts[event.User] = b.createTeamChooseChannel(&scrumpolice.Team{Name: teamName})
		return false
	}))
}

func (b *Bot) createTeamChooseChannel(team *scrumpolice.Team) BotContextHandler {
	return b.canQuitBotContext(BotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		b.logger.Println("that thing happened", event)

		chans, err := b.slackBotAPI.GetChannels(true)
		if err != nil {
			b.logger.Println("NOOO", err)
		}

		var channel *slack.Channel
		for _, c := range chans {
			if c.Name == event.Text {
				channel = &c
			}
		}

		if channel == nil {
			b.slackBotAPI.PostMessage(event.Channel, "OHOH i'm not part of that channel, please invite me and try typing the channel name again", slack.PostMessageParameters{AsUser: true})
			b.setUserContext(event.User, b.createTeamChooseChannel(team))
			return false
		}

		if !channel.IsMember {
			b.slackBotAPI.PostMessage(event.Channel, "This is a public channel and i'm not a member, I'm gonna join it!", slack.PostMessageParameters{AsUser: true})
			go func() { b.slackBotAPI.JoinChannel(channel.ID) }()
		}

		b.slackBotAPI.PostMessage(event.Channel, "OK! All good, i'm part of that channel, who is part of your team? (comma separated eg: \"pastjean, lbourdages\")", slack.PostMessageParameters{AsUser: true})
		b.setUserContext(event.User, b.createTeamChooseTeamMembers(team))

		team.Channel = event.Text

		return false
	}))
}

func (b *Bot) createTeamChooseTeamMembers(team *scrumpolice.Team) BotContextHandler {
	return b.canQuitBotContext(BotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		return false
	}))
}
