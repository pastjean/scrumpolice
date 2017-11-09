package bot

import (
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nlopes/slack"
	"github.com/pastjean/scrumpolice/scrum"
)

var (
	OutOfOfficeRegex, _ = regexp.Compile("(\\w+) is out of office")
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

	if eventText == "tutorial" {
		go b.tutorial(event)
		return
	}

	if eventText == "out of office" {
		b.outOfOffice(event, event.User, true)
		return
	}

	if OutOfOfficeRegex.MatchString(eventText) {
		b.outOfOffice(event, strings.Split(strings.Trim(eventText, " "), " ")[0], false)
		return
	}

	if eventText == "i'm back" || eventText == "i am back" {
		b.backInOffice(event)
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
	b.logger.WithFields(log.Fields{
		"event": event,
	}).Fatalln("Invalid authentication credentials")
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
		b.logSlackRelatedError(event, err, "Fail to add reaction to slack.")
		return
	}
}

func (b *Bot) sourceCode(event *slack.MessageEvent) {
	params := slack.PostMessageParameters{AsUser: true}
	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "My source code is here <https://github.com/pastjean/scrumpolice>", params)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to post message to slack.")
		return
	}
}

func (b *Bot) help(event *slack.MessageEvent) {
	message := slack.Attachment{
		MarkdownIn: []string{"text"},
		Text: "- `source code`: location of my source code\n" +
			"- `help`: well, this command\n" +
			"- `tutorial`: explains how the scrum police works. Try it!\n" +
			"- `start scrum`: starts a scrum for a team and a specific set of questions, defaults to your only team if you got only one, and only questions set if there's only one on the team you chose\n" +
			"- `restart scrum`: restart your last done scrum, if it wasn't posted\n" +
			"- `out of office`: mark current user as out of office (until `i'm back` is used)\n" +
			"- `[user] is out of office`: mark the specified user as out of office (until he or she uses `i'm back`)\n" +
			"- `i am back` or `i'm back`: mark current user as in office. MacOS smart quote can screw up with the `i'm back` command.",
	}

	params := slack.PostMessageParameters{AsUser: true}
	params.Attachments = []slack.Attachment{message}

	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "Here's a list of supported commands", params)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to post message to slack.")
		return
	}
}

// This method sleeps to give a better feeling to the user. It should be use in a sub-routine.
func (b *Bot) tutorial(event *slack.MessageEvent) {
	params := slack.PostMessageParameters{AsUser: true}

	b.slackBotAPI.PostMessage(event.Channel, "*Hi there* :wave: You're new, aren't you? You want to know how I do thing? Here :golang:es!", params)
	time.Sleep(3500 * time.Millisecond)
	b.slackBotAPI.PostMessage(event.Channel, "When you want to start a scrum report, just tell me `start scrum` in a direct message :flag-dm:. _If you are part of more than one team, specify the team (I will ask you if you don't)_", params)
	time.Sleep(6500 * time.Millisecond)
	b.slackBotAPI.PostMessage(event.Channel, "Then, I will ask you a couple of questions, and wait for your answers. Once you anwsered all the questions, you're done :white_check_mark:.", params)
	b.slackBotAPI.PostMessage(event.Channel, "I take care of the rest! :cop:", params)
	time.Sleep(4500 * time.Millisecond)
	b.slackBotAPI.PostMessage(event.Channel, "When it's time :clock10:, I will post the scrum report for you and your friends in your team's channel :raised_hands:\n", params)
	time.Sleep(4500 * time.Millisecond)
	b.slackBotAPI.PostMessage(event.Channel, "All you have to do now is read the report :book: (when you have the time, I don't want to rush you :scream:)", params)
	time.Sleep(3000 * time.Millisecond)
	b.slackBotAPI.PostMessage(event.Channel, "That's all. Enjoy :beers:.", params)
}

func (b *Bot) outOfOffice(event *slack.MessageEvent, userId string, resolveUser bool) {
	params := slack.PostMessageParameters{AsUser: true}
	username := userId
	if resolveUser {
		user, err := b.slackBotAPI.GetUserInfo(userId)
		if err != nil {
			b.logSlackRelatedError(event, err, "Fail to get user information.")
			b.slackBotAPI.PostMessage(event.Channel, "Hmmmm, I couldn't find you. Try again!", params)
			return
		}
		username = user.Name
	}

	teams := b.scrum.GetTeamsForUser(username)

	for _, team := range teams {
		b.scrum.AddToOutOfOffice(team, username)
	}
	if event.User == userId {
		b.slackBotAPI.PostMessage(event.Channel, "I've marked you out of office in all your teams", params)
		log.WithFields(log.Fields{
			"user":   username,
			"doneBy": username,
		}).Info("User was marked out of office.")
	} else {
		b.slackBotAPI.PostMessage(event.Channel, "I've marked @"+userId+" out of office in all of his teams", params)

		user, err := b.slackBotAPI.GetUserInfo(event.User)
		if err != nil {
			b.logSlackRelatedError(event, err, "Fail to get user information.")
			return
		}
		b.slackBotAPI.PostMessage("@"+userId, "You've been marked out of office by @"+user.Name+".", params)
		log.WithFields(log.Fields{
			"user":   userId,
			"doneBy": user.Name,
		}).Info("User was marked out of office.")
	}
}

func (b *Bot) backInOffice(event *slack.MessageEvent) {
	params := slack.PostMessageParameters{AsUser: true}
	user, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		b.slackBotAPI.PostMessage(event.Channel, "Hmmmm, I couldn't find you. Try again!", params)
		return
	}
	username := user.Name

	teams := b.scrum.GetTeamsForUser(username)

	for _, team := range teams {
		b.scrum.RemoveFromOutOfOffice(team, username)
	}
	b.slackBotAPI.PostMessage(event.Channel, "I've marked you in office in all your teams. Welcome back!", params)
	log.WithFields(log.Fields{
		"user": username,
	}).Info("User was marked in office.")
}

func (b *Bot) unrecognizedMessage(event *slack.MessageEvent) {
	log.WithFields(log.Fields{
		"text": event.Text,
		"user": event.Username,
	}).Info("Received unrecognized message.")
	params := slack.PostMessageParameters{AsUser: true}

	_, _, err := b.slackBotAPI.PostMessage(event.Channel, "I don't understand what you're trying to tell me, try `help`", params)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to post message to slack.")
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

func (b *Bot) logSlackRelatedError(event *slack.MessageEvent, err error, logMessage string) {
	b.logger.WithFields(log.Fields{
		"text":  event.Text,
		"user":  event.Username,
		"error": err,
	}).Error(logMessage)
}
