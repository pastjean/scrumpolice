package bot

import (
	"log"
	"runtime"
	"strings"

	"github.com/nlopes/slack"
)

type (
	Bot struct {
		slackBotRTM *slack.RTM
		slackBotAPI *slack.Client

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
		slackBotAPI: slackApiClient,
		slackBotRTM: slackBotRTM,
		logger:      logger,

		iconURL: "http://i.imgur.com/dzZvzXm.jpg",
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

	eventText := event.Text

	// Handle commands that can be handled in public and can be adressed to anyone
	// TODO: what are the commands that can do this ? :/

	if !isIM && !b.adressedToMe(eventText) {
		return
	}

	if b.adressedToMe(eventText) {
		eventText = b.trimBot(eventText)
	}
	// Handle commands that can be handled in public and that must be adressed to me
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

	// Handle commands that can be handled in private :)
	if isIM {

	}
}

func (b *Bot) adressedToMe(msg string) bool {
	return strings.HasPrefix(msg, "<@"+b.id+">") || strings.HasPrefix(msg, b.name)
}

func (b *Bot) trimBot(msg string) string {
	b.logger.Println(msg, b.id, b.name)

	msg = strings.Replace(msg, "<@"+b.id+">", "", 1)
	msg = strings.Replace(msg, b.name, "", 1)
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

func (b *Bot) sourceCode(event *slack.MessageEvent) {
	params := slack.PostMessageParameters{AsUser: true}
	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "My source code is here <https://github.com/scrumpolice/scrumpolice>", params)
	if err != nil {
		b.logger.Println("%s\n", err)
		return
	}
}

func (b *Bot) replyBotLocation(event *slack.MessageEvent) {
	params := slack.PostMessageParameters{AsUser: true}
	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "*My irresponsible owners forgot to fill this placeholder.*", params)
	//	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "I'm currently living in the Clouds, powered by (JUST A REGULAR SOMETHING). You can find my heart at: <https://github.com/scrumpolice/scrumpolice>, *My non responsible owners forgot to fill this text.*", params)
	if err != nil {
		b.logger.Println("%s\n", err)
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
		b.logger.Println("%s\n", err)
		return
	}
}
