package bot

import (
	"log"
	"strings"
	"sync"

	"github.com/nlopes/slack"
	"github.com/pastjean/scrumpolice/scrum"
)

type (
	Bot struct {
		slackBotRTM *slack.RTM
		slackBotAPI *slack.Client

		userContextsMutex sync.Mutex
		userContexts      map[string]BotContextHandler

		scrum scrum.Service

		name    string
		iconURL string
		id      string

		logger *log.Logger
	}
)

func New(slackApiClient *slack.Client, logger *log.Logger, scrum scrum.Service) *Bot {
	slackBotRTM := slackApiClient.NewRTM()
	go slackBotRTM.ManageConnection()

	return &Bot{
		slackBotAPI:  slackApiClient,
		slackBotRTM:  slackBotRTM,
		logger:       logger,
		userContexts: map[string]BotContextHandler{},
		iconURL:      "http://i.imgur.com/dzZvzXm.jpg",
		scrum:        scrum,
	}
}

func (b *Bot) Run() {
	go func() {
		for msg := range b.slackBotRTM.IncomingEvents {
			switch evt := msg.Data.(type) {
			case *slack.MessageEvent:
				go b.handleMessage(evt)
			case *slack.InvalidAuthEvent:
				go b.handleInvalidAuth(evt)
			case *slack.ConnectedEvent:
				go b.handleConnected(evt)
			}
		}
	}()

	select {}
}

func (b *Bot) handleMessage(event *slack.MessageEvent) {
	if event.BotID != "" {
		// Ignore the messages commong from other bots
		return
	}

	isIM := false
	switch event.Channel[0] {
	case 'D':
		isIM = true
		// IM
	case 'G':
		// GROUP
	default:
		// ALL OTHER CHANNEL TYPES
	}

	eventText := strings.ToLower(event.Text)

	if !b.HandleScrumMessage(event) {
		return
	}

	// HANDLE GLOBAL PUBLIC COMMANDS HERE
	if strings.Contains(eventText, "wave") {
		b.reactToEvent(event, "wave")
		b.reactToEvent(event, "oncoming_police_car")
		return
	}

	if !isIM && !b.adressedToMe(eventText) {
		return
	}

	// FROM HERE All Commands need to be adressed to me or handled in private conversations
	adressedToMe := b.adressedToMe(eventText)
	if !isIM && adressedToMe {
		eventText = b.trimBotNameInMessage(eventText)
	}

	if isIM {
		// HANDLE PRIVATE TALK IN HERE
	}

	// From here on i only care of messages that were clearly adressed to me so i'll just get out
	if !adressedToMe && !isIM {
		return
	}

	// Handle commands adressed to me (can be public or private)
	if eventText == "source code" {
		b.sourceCode(event)
		return
	}

	if eventText == "help" {
		b.help(event)
		return
	}

	// Unrecogned message so let's help the user
	b.unrecognizedMessage(event)
	return
}

func (b *Bot) adressedToMe(msg string) bool {
	return strings.HasPrefix(msg, strings.ToLower("<@"+b.id+">")) ||
		strings.HasPrefix(msg, strings.ToLower(b.name))
}

func (b *Bot) trimBotNameInMessage(msg string) string {
	msg = strings.Replace(msg, strings.ToLower("<@"+b.id+">"), "", 1)
	msg = strings.Replace(msg, strings.ToLower(b.name), "", 1)
	msg = strings.Trim(msg, " :\n")

	return msg
}

func (b *Bot) handleInvalidAuth(event *slack.InvalidAuthEvent) {
	b.logger.Fatalln("Invalid authentication credentials", event)
}

func (b *Bot) handleConnected(event *slack.ConnectedEvent) {
	b.id = event.Info.User.ID
	b.name = event.Info.User.Name
}

func (b *Bot) reactToEvent(event *slack.MessageEvent, reaction string) {
	item := slack.ItemRef{
		Channel:   event.Channel,
		Timestamp: event.Timestamp,
	}
	err := b.slackBotAPI.AddReaction(reaction, item)
	if err != nil {
		b.logger.Printf("%s\n", err)
		return
	}
}

func (b *Bot) sourceCode(event *slack.MessageEvent) {
	params := slack.PostMessageParameters{AsUser: true}
	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "My source code is here <https://github.com/pastjean/scrumpolice>", params)
	if err != nil {
		b.logger.Printf("%s\n", err)
		return
	}
}

func (b *Bot) help(event *slack.MessageEvent) {
	message := slack.Attachment{
		MarkdownIn: []string{"text"},
		Text: "- `source code`: location of my source code\n" +
			"- `help`: well, this command\n" +
			"- `start scrum`: starts a scrum for a team and a specific set of questions, defaults to your only team if you got only one, and only questions set if there's only one on the team you chose\n" +
			"- `restart scrum`: restart your last done scrum, if it wasn't posted",
	}

	params := slack.PostMessageParameters{AsUser: true}
	params.Attachments = []slack.Attachment{message}

	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "Here's a list of supported commands", params)
	if err != nil {
		b.logger.Printf("%s\n", err)
		return
	}
}

func (b *Bot) unrecognizedMessage(event *slack.MessageEvent) {
	params := slack.PostMessageParameters{AsUser: true}

	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "I don't understand what you're trying to tell me, try `help`", params)
	if err != nil {
		b.logger.Printf("%s\n", err)
		return
	}
}

func (b *Bot) canQuitBotContextHandlerFunc(handler func(event *slack.MessageEvent) bool) BotContextHandler {
	return b.canQuitBotContext(BotContextHandlerFunc(handler))
}

func (b *Bot) canQuitBotContext(handler BotContextHandler) BotContextHandler {
	return BotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		if event.Text == "quit" {
			b.slackBotAPI.PostMessage(event.Channel, "Action is canceled, if you wanna do anything else, just poke me, `help` is always available! :wave:", slack.PostMessageParameters{AsUser: true})
			delete(b.userContexts, event.User)
			return false
		}

		return handler.HandleMessage(event)
	})
}

func (b *Bot) setUserContext(user string, context BotContextHandler) {
	b.userContextsMutex.Lock()
	b.userContexts[user] = context
	b.userContextsMutex.Unlock()
}

func (b *Bot) unsetUserContext(user string) {
	b.userContextsMutex.Lock()
	delete(b.userContexts, user)
	b.userContextsMutex.Unlock()
}
