package scrumpolice

import "time"

type SlackScrumIO interface {
	SendScrumQuestion(user string, question string) error
	SendScrumDigest(channel string, reports map[string]*Report) error
	SendScrumGracePeriodReminder(channel string, users []string, message string, maxDateTime time.Time) error
}
