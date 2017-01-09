package scrumpolice

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/nlopes/slack"
)

var _ SlackScrumIO = &Output{}

type Output struct {
	slackClient *slack.Client
	slack.PostMessageParameters
}

func NewOutput(configStore ConfigStore) *Output {
	s := &Output{}
	config, _ := configStore.Read()

	s.slackClient = slack.New(config.SlackToken)
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)

	if b, err := strconv.ParseBool(os.Getenv("DEBUG")); err == nil && b == true {
		s.slackClient.SetDebug(true)
	}
	s.PostMessageParameters = defaultPostMessageParameters()

	return s
}

func (this *Output) SendScrumQuestion(user string, question string) error {
	_, _, err := this.slackClient.PostMessage(user, question, this.PostMessageParameters)
	return err
}
func (this *Output) SendScrumDigest(channel string, reports map[string]*Report) error {
	usersToShame := make([]string, 0, len(reports))
	attachments := make([]slack.Attachment, 0, len(reports))

	for user, report := range reports {
		if report.Complete {
			attachments = append(attachments, formatReportAsAttachment(user, report))
		} else {
			usersToShame = append(usersToShame, user)
		}
	}

	messageParameters := defaultPostMessageParameters()
	messageParameters.Attachments = attachments

	_, _, err := this.slackClient.PostMessage(channel, ":parrotcop: Alrighty! Here's the scrum report for today!", messageParameters)

	if len(usersToShame) > 0 {
		shameMessage := shameUsersIfNeedTo(usersToShame)
		_, _, err := this.slackClient.PostMessage(channel, shameMessage, defaultPostMessageParameters())
		return err
	}

	return err
}

func formatReportAsAttachment(user string, report *Report) slack.Attachment {
	attachment := slack.Attachment{
		Color:      colorful.FastHappyColor().Hex(),
		MarkdownIn: []string{"text", "pretext"},
		Pretext:    user,
	}
	message := ""

	for _, question := range report.Questions {
		message += "`" + question + "`\n" + report.QuestionsAndAnswers[question] + "\n"
	}

	attachment.Text = message
	return attachment
}

func shameUsersIfNeedTo(users []string) string {
	message := ""
	if len(users) > 0 {
		message = "A special round of applause to " + strings.Join(users, ", ") + " who did not complete their scrum reports on time. Shame on you!"
	}

	return message
}

func (this *Output) SendScrumGracePeriodReminder(channel string, users []string, message string, maxDateTime time.Time) error {
	messageForChannel := message + " You have until " + maxDateTime.Format(time.Kitchen) + ". I'm talking about " + strings.Join(users, ", ") + " in particular."
	_, _, err := this.slackClient.PostMessage(channel, messageForChannel, this.PostMessageParameters)

	for _, user := range users {
		_, _, err := this.slackClient.PostMessage(user, message+" You have until "+maxDateTime.Format(time.Kitchen), this.PostMessageParameters)
		if err != nil {
			return err
		}
	}
	return err
}

func defaultPostMessageParameters() slack.PostMessageParameters {
	return slack.PostMessageParameters{AsUser: true, LinkNames: 1, IconURL: "http://i.imgur.com/dzZvzXm.jpg", Username: "The Scrum Police"}
}
