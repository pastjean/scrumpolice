package scrumpolice

import (
	"fmt"

	"github.com/nlopes/slack"
)

type SlackAdapter struct {
	client *slack.Client
	rtm    *slack.RTM

	receivedMessagesChan chan<- Message

	configStore ConfigStore
	// caches
	userIDsUsernamesMap map[string]string
}

var _ ChatAdapter = &SlackAdapter{}

func NewSlackAdapter(configStore ConfigStore, receivedMessagesChan chan<- Message) *SlackAdapter {
	config, _ := configStore.Read()
	slackClient := slack.New(config.SlackToken)

	return &SlackAdapter{slackClient, nil, receivedMessagesChan, configStore, map[string]string{}}
}

func (s *SlackAdapter) Run() {
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

func (s *SlackAdapter) Disconnect() {
	if s.rtm != nil {
		s.rtm.Disconnect()
	}
	s.rtm = nil
}

func (s *SlackAdapter) onConnectedEvent(evt *slack.ConnectedEvent) {
	// TODO: populate caches (channels, userids)
}

func (s *SlackAdapter) onMessageEvent(evt *slack.MessageEvent) {
	s.receivedMessagesChan <- *s.slackMessageEventToMessage(evt)
}

func (s *SlackAdapter) SendMessage(msg *Message) error {
	channel, text, postMessageParameters := s.messageToPostMessageParameters(msg)
	_, _, err := s.rtm.PostMessage(channel, text, postMessageParameters)
	return err
}

func (s *SlackAdapter) slackMessageEventToMessage(evt *slack.MessageEvent) *Message {
	m := &Message{
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

func (s *SlackAdapter) messageToPostMessageParameters(msg *Message) (string, string, slack.PostMessageParameters) {
	params := s.defaultPostMessageParameters()

	if msg.Attachements != nil {
		params.Attachments = make([]slack.Attachment, 0, len(msg.Attachements))
		for _, attachment := range msg.Attachements {
			params.Attachments = append(params.Attachments, *messageAttachmentToSlackAttachment(&attachment))
		}
	}

	return msg.Channel, msg.Text, slack.NewPostMessageParameters()
}

func (s *SlackAdapter) getUsernameFromID(userID string) string {
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

func messageAttachmentToSlackAttachment(attachment *MessageAttachement) *slack.Attachment {
	return &slack.Attachment{
		Color:      attachment.Color,
		MarkdownIn: []string{"text", "pretext"},
		Text:       attachment.Text,
		Pretext:    attachment.Header,
	}
}

func slackAttachementToMessageAttachment(attachment *slack.Attachment) *MessageAttachement {
	return &MessageAttachement{
		Color:  attachment.Color,
		Text:   attachment.Text,
		Header: attachment.Pretext,
	}
}

func (s *SlackAdapter) defaultPostMessageParameters() slack.PostMessageParameters {
	config, _ := s.configStore.Read()

	params := slack.NewPostMessageParameters()
	params.AsUser = true
	params.Markdown = true
	params.LinkNames = 1

	params.IconURL = config.BotIconURL
	params.Username = config.BotName
	return params
}
