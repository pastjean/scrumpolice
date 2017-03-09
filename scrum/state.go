package scrum

import (
	"time"

	"github.com/robfig/cron"
)

type (
	Team struct {
		Name          string
		Channel       string
		Members       []string
		QuestionsSets []*QuestionSet
		Timezone      *time.Location
	}

	QuestionSet struct {
		Questions                 []string
		ReportSchedule            cron.Schedule
		FirstReminderBeforeReport time.Duration
		LastReminderBeforeReport  time.Duration
	}
)
