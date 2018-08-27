package scrum

import (
	"log"
	"time"
	"github.com/robfig/cron"
)

// {
//   "timezone": "America/Montreal",
//   "teams": [
//     {
//       "channel": "general",
//       "name": "L337",
//       "members": [
//         "fboutin2",
//         "lbourdages",
//         "pa",
//         "jo"
//       ],
//       "split_report": false,
//       "question_sets": [
//         {
//           "questions": [
//             "What did you do yesterday?",
//             "What will you do today?",
//             "Are you being blocked by someone for a review? who ? why ?",
//             "How will you dominate the world"
//           ],
//           "report_schedule_cron": "@every 30s",
//           "first_reminder_limit": "-8s",
//           "last_reminder_limit": "-3s"
//         }
//       ]
//     }
//   ]
// }
type (

	// Config is the configuration format
	Config struct {
		Timezone string       `json:"timezone"`
		Teams    []TeamConfig `json:"teams"`
	}

	TeamConfig struct {
		Name         string              `json:"name"`
		Channel      string              `json:"channel"`
		Members      []string            `json:"members"`
		QuestionSets []QuestionSetConfig `json:"question_sets"`
		OutOfOffice  []string            `json:"out_of_office"`
		Timezone     string              `json:"timezone"`
		SplitReport  bool                `json:"split_report"`
	}

	QuestionSetConfig struct {
		Questions                 []string `json:"questions"`
		ReportScheduleCron        string   `json:"report_schedule_cron"`
		FirstReminderBeforeReport string   `json:"first_reminder_limit"`
		LastReminderBeforeReport  string   `json:"last_reminder_limit"`
	}
)

func (c *Config) ToTeams() []*Team {
	teams := []*Team{}
	for _, teamConfig := range c.Teams {
		teams = append(teams, teamConfig.ToTeam(c.Timezone))
	}
	return teams
}

func (tc *TeamConfig) ToTeam(timezone string) *Team {
	qsets := []*QuestionSet{}
	for _, questionsetconfig := range tc.QuestionSets {
		qs, err := questionsetconfig.toQuestionSet()
		if err != nil {
			log.Println("error parsing question set for team", tc.Name, err)
		} else {
			qsets = append(qsets, qs)
		}
	}

	t := &Team{
		Name:          tc.Name,
		Channel:       tc.Channel,
		Members:       tc.Members,
		QuestionsSets: qsets,
		SplitReport:   tc.SplitReport,
	}

	tz := timezone
	if tc.Timezone != "" {
		tz = tc.Timezone
	}

	lloc, err := time.LoadLocation(tz)
		if err != nil {
			log.Println("Timezone error for team:", tc.Name, "Will use default timezone")
			lloc = nil
		}
		t.Timezone = lloc

	return t
}

func (qs *QuestionSetConfig) toQuestionSet() (*QuestionSet, error) {
	schedule, err := cron.Parse(qs.ReportScheduleCron)
	if err != nil {
		return nil, err
	}

	fir, err := time.ParseDuration(qs.FirstReminderBeforeReport)
	if err != nil {
		return nil, err
	}

	sec, err := time.ParseDuration(qs.LastReminderBeforeReport)
	if err != nil {
		return nil, err
	}

	return &QuestionSet{
		Questions:                 qs.Questions,
		ReportSchedule:            schedule,
		ReportScheduleCron:        qs.ReportScheduleCron,
		FirstReminderBeforeReport: fir,
		LastReminderBeforeReport:  sec,
	}, nil
}
