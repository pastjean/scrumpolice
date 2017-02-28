package bot

import "time"
import "github.com/robfig/cron"

type (
	Team struct {
		Name                      string
		Channel                   string
		Members                   []string
		Questions                 []string
		ReportCronLimit           cron.Schedule
		ReminderLimitBeforeReport time.Duration
		AlertToFillReport         time.Duration
		Timezone                  *time.Location
	}

	TeamConfig struct {
		Name                      string   `json:"name"`
		Channel                   string   `json:"channel"`
		Members                   []string `json:"members"`
		Questions                 []string `json:"questions"`
		ReportCronLimit           string   `json:"report_schedule_cron"`
		ReminderLimitBeforeReport string   `json:"limit_period"`
		AlertToFillReport         string   `json:"grace_reminder_period"`
		Timezone                  string   `json:"timezone"`
	}

	Config struct {
		SlackToken      string       `json:"slack_token"`
		DefaultTimezone string       `json:"default_timezone"`
		Teams           []TeamConfig `json:"teams"`
	}
)

var (
	DefaultConfig = Config{
		DefaultTimezone: "Local",
		Teams:           []TeamConfig{},
	}
)
