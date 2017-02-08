package bot

import (
	"log"
	"runtime"
	"strings"
	"sync"

	"github.com/nlopes/slack"
)

type (
	Bot struct {
		slackBotRTM *slack.RTM
		slackBotAPI *slack.Client

		userContextsMutex sync.Mutex
		userContexts      map[string]BotContextHandler

		name    string
		iconURL string
		id      string

		logger *log.Logger
	}
)

func New(slackApiClient *slack.Client, logger *log.Logger) *Bot {
	slackBotRTM := slackApiClient.NewRTM()
	go slackBotRTM.ManageConnection()
	runtime.Gosched()

	return &Bot{
		slackBotAPI:  slackApiClient,
		slackBotRTM:  slackBotRTM,
		logger:       logger,
		userContexts: map[string]BotContextHandler{},
		iconURL:      "http://i.imgur.com/dzZvzXm.jpg",
	}
}

func (b *Bot) Run() {
	go b.RunScrumModule()

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
	if !isIM && b.adressedToMe(eventText) {
		eventText = b.trimBotNameInMessage(eventText)
		return
	}

	if isIM {
		// HANDLE PRIVATE TALK IN HERE
	}

	if b.adressedToMe(eventText) {
		eventText = b.trimBotNameInMessage(eventText)
	}

	// Handle commands adressed to me (can be public or private)
	if eventText == "where do you live?" ||
		eventText == "stack" {
		b.replyBotLocation(event)
		return
	}

	if eventText == "source code" {
		b.sourceCode(event)
		return
	}

	if eventText == "help" {
		b.help(event)
		return
	}
}

func (b *Bot) adressedToMe(msg string) bool {
	return strings.HasPrefix(msg, strings.ToLower("<@"+b.id+">")) ||
		strings.HasPrefix(msg, strings.ToLower(b.name))
}

func (b *Bot) trimBotNameInMessage(msg string) string {
	b.logger.Println(msg, b.id, b.name)

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
	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "My source code is here <https://github.com/scrumpolice/scrumpolice>", params)
	if err != nil {
		b.logger.Printf("%s\n", err)
		return
	}
}

func (b *Bot) replyBotLocation(event *slack.MessageEvent) {
	params := slack.PostMessageParameters{AsUser: true}
	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "*My irresponsible owners forgot to fill this placeholder.*", params)
	//	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "I'm currently living in the Clouds, powered by (JUST A REGULAR SOMETHING). You can find my heart at: <https://github.com/scrumpolice/scrumpolice>, *My non responsible owners forgot to fill this text.*", params)
	if err != nil {
		b.logger.Printf("%s\n", err)
		return
	}
}

func (b *Bot) help(event *slack.MessageEvent) {
	message := slack.Attachment{
		Text: `- "source code" -> location of my source code
- "where do you live?" OR "stack" -> get information about where the tech stack behind @scrumpolice
- "help" -> well, this command`,
	}
	params := slack.PostMessageParameters{AsUser: true}
	params.Attachments = []slack.Attachment{message}

	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "Here's a list of supported commands", params)
	if err != nil {
		b.logger.Printf("%s\n", err)
		return
	}
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
