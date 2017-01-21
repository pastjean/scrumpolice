package slack

import (
	"fmt"

	"github.com/nlopes/slack"
	"github.com/scrumpolice/scrumpolice/chat"
)

type Adapter struct {
	client *slack.Client
	rtm    *slack.RTM

	receivedMessagesChan chan<- chat.Message

	username string
	iconURL  string
	token    string
	// caches
	userIDsUsernamesMap map[string]string
}

var _ chat.Adapter = &Adapter{}

func NewChatAdapter(slackToken string, receivedMessagesChan chan<- chat.Message) *Adapter {
	return &Adapter{
		client:               slack.New(slackToken),
		rtm:                  nil,
		receivedMessagesChan: receivedMessagesChan,
		username:             "Scrumpolice",
		iconURL:              "http://i.imgur.com/dzZvzXm.jpg",
		userIDsUsernamesMap:  map[string]string{},
	}
}

func (s *Adapter) Run() {
	// Prevent double Run()
	if s.rtm != nil {
		s.rtm.Disconnect()
	}

	rtm := s.client.NewRTM()
	s.rtm = rtm
	go rtm.ManageConnection()

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				s.onConnectedEvent(ev)
			case *slack.MessageEvent:
				s.onMessageEvent(ev)
			case *slack.InvalidAuthEvent:
				fmt.Printf("[ERROR] Invalid credentials")
				return
			default:
				// Ignore other events..
			}
		}
	}
}

func (s *Adapter) Disconnect() {
	if s.rtm != nil {
		s.rtm.Disconnect()
	}
	s.rtm = nil
}

func (s *Adapter) onConnectedEvent(evt *slack.ConnectedEvent) {
	// TODO: populate caches (channels, userids)
}

func (s *Adapter) onMessageEvent(evt *slack.MessageEvent) {
	s.receivedMessagesChan <- *s.slackMessageEventToMessage(evt)
}

func (s *Adapter) SendMessage(msg *chat.Message) error {
	channel, text, postMessageParameters := s.messageToPostMessageParameters(msg)
	_, _, err := s.rtm.PostMessage(channel, text, postMessageParameters)
	return err
}

func (s *Adapter) slackMessageEventToMessage(evt *slack.MessageEvent) *chat.Message {
	m := &chat.Message{
		Text:    evt.Text,
		Channel: evt.Channel, // todo convert channel id
		User:    s.getUsernameFromID(evt.User),
	}

	switch evt.Channel[0] {
	case 'D':
		m.IsIM = true
		m.Channel = m.User
	case 'G':
		m.IsGroup = true
		if group, err := s.client.GetGroupInfo(evt.Channel); err == nil {
			m.Channel = group.Name
		}
	default:
		if channel, err := s.client.GetChannelInfo(evt.Channel); err == nil {
			m.Channel = channel.Name
		}
	}

	return m
}

func (s *Adapter) messageToPostMessageParameters(msg *chat.Message) (string, string, slack.PostMessageParameters) {
	params := s.defaultPostMessageParameters()

	if msg.Attachements != nil {
		params.Attachments = make([]slack.Attachment, 0, len(msg.Attachements))
		for _, attachment := range msg.Attachements {
			params.Attachments = append(params.Attachments, *messageAttachmentToSlackAttachment(&attachment))
		}
	}

	return msg.Channel, msg.Text, slack.NewPostMessageParameters()
}

func (s *Adapter) getUsernameFromID(userID string) string {
	if username, ok := s.userIDsUsernamesMap[userID]; ok {
		return username
	}

	user, err := s.client.GetUserInfo(userID)
	if err != nil {
		return ""
	}

	s.userIDsUsernamesMap[userID] = user.Name
	return user.Name
}

func messageAttachmentToSlackAttachment(attachment *chat.MessageAttachement) *slack.Attachment {
	return &slack.Attachment{
		Color:      attachment.Color,
		MarkdownIn: []string{"text", "pretext"},
		Text:       attachment.Text,
		Pretext:    attachment.Header,
	}
}

func slackAttachementToMessageAttachment(attachment *slack.Attachment) *chat.MessageAttachement {
	return &chat.MessageAttachement{
		Color:  attachment.Color,
		Text:   attachment.Text,
		Header: attachment.Pretext,
	}
}

func (s *Adapter) defaultPostMessageParameters() slack.PostMessageParameters {
	params := slack.NewPostMessageParameters()
	params.AsUser = true
	params.Markdown = true
	params.LinkNames = 1

	params.IconURL = s.iconURL
	params.Username = s.username
	return params
}
