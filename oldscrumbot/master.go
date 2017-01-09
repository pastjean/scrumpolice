package scrumpolice

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/robfig/cron"
)

type Master struct {
	SlackScrumIO SlackScrumIO
	ConfigStore  ConfigStore
	Cron         *cron.Cron
	TeamStates   map[string]TeamState
}

func NewMaster(configStore ConfigStore, slackScrumIO SlackScrumIO) *Master {
	teamStates := map[string]TeamState{}

	config, _ := configStore.Read()

	for _, team := range config.Teams {
		teamStates[team.Name] = *NewTeamState(team.Name, team.Members, team.QuestionSet.Questions)
	}

	return &Master{
		ConfigStore:  configStore,
		Cron:         cron.New(),
		TeamStates:   teamStates,
		SlackScrumIO: slackScrumIO,
	}
}

func (master *Master) Start() {
	config, _ := master.ConfigStore.Read()

	for _, team := range config.Teams {
		graceDuration, _ := time.ParseDuration(team.QuestionSet.GraceDuration)
		limitTime, _ := time.ParseDuration(team.QuestionSet.LimitTime)

		master.Cron.AddFunc(team.TimeSheetReminder.CronString, func() {
			go func() {
				for _, member := range team.Members {
					master.SlackScrumIO.SendScrumQuestion(member, team.TimeSheetReminder.Message)
				}
			}()
		})
		master.Cron.AddFunc(team.QuestionSet.CronString, func() {
			go func() {
				for _, member := range team.Members {
					master.TeamStates[team.Name] = *NewTeamState(team.Name, team.Members, team.QuestionSet.Questions)
					master.SlackScrumIO.SendScrumQuestion(member, team.QuestionSet.Questions[0])
				}
			}()
			go func() {
				time.Sleep(graceDuration)
				membersToShame := []string{}
				for _, member := range team.Members {
					report := master.TeamStates[team.Name].scrumReports[member]
					if !report.Complete {
						membersToShame = append(membersToShame, member)
					}
				}
				if len(membersToShame) != 0 {
					master.SlackScrumIO.SendScrumGracePeriodReminder(team.Channel, membersToShame, team.QuestionSet.Reminder, time.Now().Add(limitTime-graceDuration))
				}
			}()
			go func() {
				time.Sleep(limitTime)
				master.SlackScrumIO.SendScrumDigest(team.Channel, master.TeamStates[team.Name].scrumReports)
			}()
		})
	}
	master.Cron.Start()
}

func (master *Master) PublishScrumReport(user string, answer string) {
	if strings.HasPrefix(answer, "$config") {
		var buffer bytes.Buffer
		config, _ := master.ConfigStore.Read()
		toml.NewEncoder(&buffer).Encode(&config)
		master.ConfigStore.Write(config)
		master.SlackScrumIO.SendScrumQuestion(user, "```"+buffer.String()+"```")
	} else if (user == "@fboutin2" || user == "@lbourdages" || user == "@pastjean") && strings.HasPrefix(answer, "{") {
		master.ConfigureTeam(user, answer)
	} else {
		for _, teamState := range master.TeamStates {
			for _, teamUser := range teamState.users {
				if teamUser == user {
					report := teamState.scrumReports[user]
					if report.CurrentQuestion() != "" {
						report.QuestionsAndAnswers[report.CurrentQuestion()] = answer
					}
					question := report.NextQuestion()
					if question != "" {
						master.SlackScrumIO.SendScrumQuestion(user, question)
					} else {
						if answer == "yes" {
							master.SlackScrumIO.SendScrumQuestion(user, report.ResetQuestions())
						} else {
							master.SlackScrumIO.SendScrumQuestion(user, "Thanks for your scrum report my :deer:! :bear: with us for the digest. :owl: see you tomorrow!\nWant to restart your scrum report? (yes)")
						}
					}
					return
				}
			}
		}
	}
}

func (master *Master) ConfigureTeam(user string, jsonConfig string) {
	var config Config
	json.Unmarshal([]byte(jsonConfig), &config)
	if err := master.ConfigStore.Write(config); err != nil {
		// TODO: handle errors more gracefully, though the in memory doesn't throw anything
		return
	}

	master.Cron.Stop()

	master.TeamStates = map[string]TeamState{}

	for _, team := range config.Teams {
		master.TeamStates[team.Name] = *NewTeamState(team.Name, team.Members, team.QuestionSet.Questions)
	}
	master.Cron = cron.New()
	master.SlackScrumIO.SendScrumQuestion("#scrumbottest", "User "+user+" changed scrum configuration for : \n"+jsonConfig)
	master.Start()
}
