package scrum

import (
	"github.com/robfig/cron"
	"time"
)

type (
	Team struct {
		Name          string
		Channel       string
		Members       []string
		QuestionsSets []*QuestionSet
		Timezone      *time.Location
		OutOfOffice   []string
		SplitReport   bool
	}

	QuestionSet struct {
		Questions                 []string
		ReportSchedule            cron.Schedule
		ReportScheduleCron        string
		FirstReminderBeforeReport time.Duration
		LastReminderBeforeReport  time.Duration
	}
)

func (team *Team) toTeamConfig() *TeamConfig {
	var questionSetConfigs []QuestionSetConfig
	for _, questionSet := range team.QuestionsSets {
		questionSetConfig := questionSet.toQuestionSetConfig()
		questionSetConfigs = append(questionSetConfigs, *questionSetConfig)
	}

	return &TeamConfig{
		Name:         team.Name,
		Channel:      team.Channel,
		Members:      team.Members,
		QuestionSets: questionSetConfigs,
		Timezone:     team.Timezone.String(),
		OutOfOffice:  team.OutOfOffice,
		SplitReport:  team.SplitReport,
	}
}

func (questionSet *QuestionSet) toQuestionSetConfig() *QuestionSetConfig {
	return &QuestionSetConfig{
		Questions:                 questionSet.Questions,
		ReportScheduleCron:        questionSet.ReportScheduleCron,
		FirstReminderBeforeReport: questionSet.FirstReminderBeforeReport.String(),
		LastReminderBeforeReport:  questionSet.LastReminderBeforeReport.String(),
	}
}
