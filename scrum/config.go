package scrum

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
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
	ConfigurationProvider interface {
		Config() *Config
		OnChange(handler func(cfg *Config))
	}

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

type configFileWatcher struct {
	config         *Config
	changeHandlers []func(cfg *Config)
}

func NewConfigWatcher(file string) ConfigurationProvider {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	fw := &configFileWatcher{changeHandlers: []func(cfg *Config){}}
	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("Configuration file modified '", event.Name, "', reloading...")
					fw.reloadAndDistributeChange(file)
				}
			case err := <-watcher.Errors:
				log.Println("Error while watching for configuration file:", err)
			}
		}
	}()

	err = watcher.Add(file)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Loading initial configuration")
	fw.reloadAndDistributeChange(file)
	return fw
}

func (fw *configFileWatcher) Config() *Config {
	return fw.config
}

func (fw *configFileWatcher) OnChange(handler func(cfg *Config)) {
	fw.changeHandlers = append(fw.changeHandlers, handler)
}

func (fw *configFileWatcher) reloadAndDistributeChange(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Cannot open file '", filename, "', error:", err)
	}
	err = json.NewDecoder(file).Decode(&fw.config)
	if err != nil {
		log.Println("Cannot parse configuration file ('", filename, "') content:", err)
	}

	for _, handler := range fw.changeHandlers {
		go handler(fw.config)
	}
}

func (c *Config) ToTeams() []*Team {
	teams := []*Team{}
	for _, teamConfig := range c.Teams {
		teams = append(teams, teamConfig.ToTeam())
	}
	return teams
}

func (tc *TeamConfig) ToTeam() *Team {
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

	if tc.Timezone != "" {
		lloc, err := time.LoadLocation(tc.Timezone)
		if err != nil {
			log.Println("Timezone error for team:", tc.Name, "Will use default timezone")
			lloc = nil
		}
		t.Timezone = lloc
	}

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
		FirstReminderBeforeReport: fir,
		LastReminderBeforeReport:  sec,
	}, nil
}
