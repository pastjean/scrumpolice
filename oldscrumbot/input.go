package scrumpolice

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/nlopes/slack"
)

type Input struct {
	slackClient     *slack.Client
	logger          log.Logger
	scrumMaster     *Master
	usernameOfRobot string
}

func NewInput(scrumMaster *Master) *Input {
	this := &Input{}

	config, _ := scrumMaster.ConfigStore.Read()

	this.slackClient = slack.New(config.SlackToken)
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)

	if b, err := strconv.ParseBool(os.Getenv("DEBUG")); err == nil && b == true {
		this.slackClient.SetDebug(true)
	}

	this.usernameOfRobot = "scrumpolice"

	this.scrumMaster = scrumMaster

	return this
}

func (this *Input) Start() {

	rtm := this.slackClient.NewRTM()
	go rtm.ManageConnection()

	// see https://github.com/nlopes/slack/blob/master/examples/websocket/websocket.go
	for {
		select {
		case evt := <-rtm.IncomingEvents:
			this.receiveSlackEvent(evt)
		}
	}
}

func (this *Input) receiveSlackEvent(evt slack.RTMEvent) {
	switch ev := evt.Data.(type) {
	case *slack.MessageEvent:
		fmt.Printf("Message: %v\n", ev)
		this.sendMessageToScrumMasterIfNecessary(ev)
	}
}

func (t *Input) getUsernameFromUserId(userID string) (string, error) {
	user, err := t.slackClient.GetUserInfo(userID)
	if err != nil {
		return user.Name, err
	}
	return user.Name, nil
}

func (this *Input) sendMessageToScrumMasterIfNecessary(message *slack.MessageEvent) {
	channelIsDirectMessage, err := this.channelIsDirectMessage(message.Channel)

	if err != nil {
		fmt.Println(err)
		return
	}
	if channelIsDirectMessage {
		username, err := this.userNameFromUserId(message.User)
		if err != nil {
			fmt.Println(err)
			return
		}
		if username != this.usernameOfRobot {
			fmt.Println(username)
			this.scrumMaster.PublishScrumReport("@"+username, message.Text)
		}
	}
}

func (this *Input) channelIsDirectMessage(channelId string) (bool, error) {
	imChannels, err := this.slackClient.GetIMChannels()
	if err != nil {
		fmt.Println(err)
		return false, err
	}

	for _, channel := range imChannels {
		if channel.ID == channelId {
			return true, nil
		}
	}
	return false, nil
}

func (this *Input) userNameFromUserId(userId string) (string, error) {
	user, err := this.slackClient.GetUserInfo(userId)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return user.Name, nil
}
