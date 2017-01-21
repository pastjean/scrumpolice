package scrumpolice

import (
	"os"

	"github.com/BurntSushi/toml"
)

// Version is the version of the scrumbot
const Version = "0.0.2-beta"

// Config is the configuration of the running bot
type Config struct {
	SlackToken string       `toml:"slack_token"`
	Teams      []TeamConfig `toml:"team"`
}

// TeamConfig is the Configuration of one team
type TeamConfig struct {
	QuestionSet       QuestionSet `toml:"question_set"`
	Channel           string
	Name              string
	Members           []string
	TimeSheetReminder Reminder `toml:"timesheet_reminder"`
}

// QuestionSet is a set of questions asked to a team on every Cron call
type QuestionSet struct {
	Questions  []string
	CronString string `toml:"cron_schedule"`
	// GraceDuration must be a duration parsable string
	GraceDuration string `toml:"grace_duration"`
	// LimitTime must be a duration parsable string
	LimitTime string `toml:"limit_time"`
	Reminder  string
}

// Reminder is a nice reminder for your team members to receive a message (eg: timesheet)
type Reminder struct {
	// Reminder is the message the bot will send to your team on every Cron call
	Message string
	// CronString is the cron formatted string used to call the reminder
	CronString string `toml:"cron_schedule"`
}

// ConfigStore is resposible of storing and reading the config
type ConfigStore interface {
	Read() (Config, error)
	Write(Config) error
	// TODO: probably will need a OnChange Hook
}

type TOMLConfigStore struct {
	fpath  string
	config Config
}

var _ ConfigStore = &TOMLConfigStore{}

func NewTOMLConfigStore(filepath string) (*TOMLConfigStore, error) {
	var c Config
	_, err := toml.DecodeFile(filepath, &c)
	if err != nil {
		return nil, err
	}

	return &TOMLConfigStore{
		fpath:  filepath,
		config: c,
	}, nil
}

func (t *TOMLConfigStore) Read() (Config, error) {
	return t.config, nil
}

func (t *TOMLConfigStore) Write(config Config) error {
	file, err := os.OpenFile(t.fpath, os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	if err := toml.NewEncoder(file).Encode(t.config); err != nil {
		return err
	}

	// We update local state once everything is done
	t.config = config
	return nil
}
