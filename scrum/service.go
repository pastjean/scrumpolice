package scrum

import (
	"errors"
	"fmt"
	"strings"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/nlopes/slack"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

type Service interface {
	DeleteLastReport(username string) bool
	GetTeams() []string
	GetTeamByName(teamName string) (*TeamState, error)
	GetTeamsForUser(username string) []string
	GetQuestionSetsForTeam(team string) []*QuestionSet
	GetUsersForTeam(team string) []string
	SaveReport(report *Report, qs *QuestionSet)
	AddToOutOfOffice(team string, username string)
	RemoveFromOutOfOffice(team string, username string)
	AddTeam(team *Team)
	DeleteTeam(team string)
	AddToTeam(team string, username string)
	RemoveFromTeam(team string, username string)
	ReplaceScrumScheduleInTeam(team string, schedule cron.Schedule, scheduleAsString string)
	ReplaceFirstReminderInTeam(team string, duration time.Duration)
	ReplaceLastReminderInTeam(team string, duration time.Duration)
	ReplaceScrumQuestionsInTeam(team string, questions []string)
	ChangeTeamChannel(team string, channel string)
}

type service struct {
	configurationStorage ConfigurationStorage
	timezone             string
	teamStates           map[string]*TeamState
	slackBotAPI          *slack.Client
	lastEnteredReport    map[string]*Report
}

type TeamState struct {
	*Team
	*cron.Cron
	*service

	questionSetStates map[*QuestionSet]*questionSetState
}

type questionSetState struct {
	*QuestionSet
	enteredReports map[string]*Report
	sent           bool
}

type Report struct {
	User    string
	Team    string
	Skipped bool
	// questions / answers
	Answers map[string]string
}

func emptyQuestionSetState(qs *QuestionSet) *questionSetState {
	return &questionSetState{qs, map[string]*Report{}, false}
}

func isMemberOutOfOffice(ts *TeamState, member string) bool {
	isOutOfOffice := false
	for _, outOfOfficeMember := range ts.OutOfOffice {
		if outOfOfficeMember == member {
			isOutOfOffice = true
			break
		}
	}
	return isOutOfOffice
}

func (teamState *TeamState) postMessageToSlack(channel string, options ...slack.MsgOption) {
	_, _, err := teamState.service.slackBotAPI.PostMessage(channel, append(options, slack.MsgOptionAsUser(true))...)
	if err != nil {
		log.WithFields(log.Fields{
			"team":    teamState.Team.Name,
			"channel": channel,
			"error":   err,
		}).Warn("Error while posting message to slack")
	}
}

func (teamState *TeamState) sendReportForTeam(qs *QuestionSet) {
	qsstate := teamState.questionSetStates[qs]
	if qsstate.sent == true {
		return
	}
	qsstate.sent = true

	if len(qsstate.enteredReports) == 0 {
		teamState.postMessageToSlack(teamState.Channel, slack.MsgOptionText("I'd like to take time to :shame: everyone for not reporting", false))
		return
	}

	attachments := []slack.Attachment{}
	didNotDoReport := []string{}
	for _, member := range teamState.Members {
		report, ok := qsstate.enteredReports[member]
		if !ok {
			if isMemberOutOfOffice(teamState, member) {
				attachment := slack.Attachment{
					Color:      colorful.FastHappyColor().Hex(),
					MarkdownIn: []string{"text", "pretext"},
					Pretext:    member,
					Text:       "I am currently out of office :sunglasses: :palm_tree:",
				}
				attachments = append(attachments, attachment)
			} else {
				didNotDoReport = append(didNotDoReport, member)
			}
		} else if report.Skipped {
			attachment := slack.Attachment{
				Color:      colorful.FastHappyColor().Hex(),
				MarkdownIn: []string{"text", "pretext"},
				Pretext:    member,
				Text:       "Has nothing to declare (most probably :bee:cause he did nothing :troll:)",
			}
			attachments = append(attachments, attachment)
		} else {
			message := ""
			for _, q := range qsstate.QuestionSet.Questions {
				message += "`" + q + "`\n" + report.Answers[q] + "\n"
			}

			attachment := slack.Attachment{
				Color:      colorful.FastHappyColor().Hex(),
				MarkdownIn: []string{"text", "pretext"},
				Pretext:    member,
				Text:       message,
			}
			attachments = append(attachments, attachment)
		}

	}

	if teamState.SplitReport {
		teamState.postMessageToSlack(teamState.Channel, slack.MsgOptionText(":parrotcop: Alrighty! Here's the scrum report for today!", false))
		for i := 0; i < len(attachments); i++ {
			teamState.postMessageToSlack(teamState.Channel, slack.MsgOptionText("*Scrum by*", false), slack.MsgOptionAttachments([]slack.Attachment{attachments[i]}...))
		}
	} else {
		teamState.postMessageToSlack(teamState.Channel, slack.MsgOptionText(":parrotcop: Alrighty! Here's the scrum report for today!", false), slack.MsgOptionAttachments(attachments...))
	}

	if len(didNotDoReport) > 0 {
		teamState.postMessageToSlack(teamState.Channel, slack.MsgOptionText(fmt.Sprintln("And lastly we should take a little time to shame", didNotDoReport), false))
	}

	log.WithFields(log.Fields{
		"team":    teamState.Team.Name,
		"channel": teamState.Channel,
	}).Info("Sent scrum report.")
}

func (teamState *TeamState) sendFirstReminder(qs *QuestionSet) {
	questionSetState := teamState.questionSetStates[qs]

	log.WithFields(log.Fields{
		"team":    teamState.Team.Name,
		"channel": teamState.Channel,
	}).Info("Sending first reminder.")

	for _, member := range teamState.Members {
		if !isMemberOutOfOffice(teamState, member) {
			_, ok := questionSetState.enteredReports[member]
			if !ok {
				_, _, err := teamState.service.slackBotAPI.PostMessage("@"+member, slack.MsgOptionText("Hey! Don't forget to fill your report! `start scrum` to do it or `skip` if you have nothing to say", false))
				if err != nil {
					log.WithFields(log.Fields{
						"team":    teamState.Team.Name,
						"member":  member,
						"channel": teamState.Channel,
						"error":   err,
					}).Warn("Could not send first reminder.")
				}
			}
		} else {
			log.WithFields(log.Fields{
				"team":    teamState.Team.Name,
				"member":  member,
				"channel": teamState.Channel,
			}).Info("Member out of office, not sending reminder.")
		}
	}
}

func (teamState *TeamState) sendLastReminder(qs *QuestionSet) {
	questionSetState := teamState.questionSetStates[qs]
	didNotDoReport := []string{}

	log.WithFields(log.Fields{
		"team":    teamState.Team.Name,
		"channel": teamState.Channel,
	}).Info("Sending last reminder.")

	for _, member := range teamState.Members {
		if !isMemberOutOfOffice(teamState, member) {
			_, ok := questionSetState.enteredReports[member]
			if !ok {
				didNotDoReport = append(didNotDoReport, member)
			}
		}
	}

	if len(didNotDoReport) == 0 {
		return
	}

	memberThatDidNotDoReport := strings.Join(didNotDoReport, ", ")
	teamState.postMessageToSlack(teamState.Channel, slack.MsgOptionText(fmt.Sprintf("Last chance to fill report! :shame: to: %s", memberThatDidNotDoReport), false))
}

type ScrumReportJob struct {
	*TeamState
	*QuestionSet
}

func (job *ScrumReportJob) Run() {
	job.TeamState.sendReportForTeam(job.QuestionSet)
	// Reset the questionSetState
	job.TeamState.questionSetStates[job.QuestionSet] = emptyQuestionSetState(job.QuestionSet)
}

type iteration uint8

const (
	First iteration = iota
	Last
)

type ScrumReminderJob struct {
	iteration iteration
	*TeamState
	*QuestionSet
}

func (job *ScrumReminderJob) Run() {
	// Post to slack things
	if job.iteration == First {
		job.TeamState.sendFirstReminder(job.QuestionSet)
		return
	}
	if job.iteration == Last {
		job.TeamState.sendLastReminder(job.QuestionSet)
		return
	}
}

func NewService(configurationStorage ConfigurationStorage, slackBotAPI *slack.Client) Service {
	mod := &service{
		configurationStorage: configurationStorage,
		slackBotAPI:          slackBotAPI,
		teamStates:           map[string]*TeamState{},
		lastEnteredReport:    map[string]*Report{},
	}

	// initial *refresh
	mod.refresh(configurationStorage.Load())

	return mod
}

func (mod *service) refresh(config *Config) {
	teams := config.ToTeams()

	log.Info("Refreshing teams.")

	globalLocation := time.Local
	if config.Timezone != "" {
		location, err := time.LoadLocation(config.Timezone)
		if err == nil {
			globalLocation = location
		} else {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Error loading global location, using default.")
		}
	}
	mod.timezone = globalLocation.String()

	for _, team := range teams {
		state, ok := mod.teamStates[team.Name]
		if !ok {
			log.WithFields(log.Fields{
				"team": team.Name,
			}).Info("Initializing team.")
		} else {
			// FIXME: Check if team changed before doing that
			log.WithFields(log.Fields{
				"team": team.Name,
			}).Info("Refreshing team.")
			state.Cron.Stop()
		}
		state = initTeamStateWithLocation(team, globalLocation, mod)
		mod.teamStates[team.Name] = state
	}
}

func (mod *service) saveConfig() {
	mod.configurationStorage.Save(mod.getCurrentConfig())
}

func (mod *service) getCurrentConfig() *Config {
	var teamConfigs []TeamConfig

	for _, teamState := range mod.teamStates {
		teamConfigs = append(teamConfigs, *teamState.Team.toTeamConfig())
	}

	return &Config{
		Timezone: mod.timezone,
		Teams:    teamConfigs,
	}
}

func initTeamStateWithLocation(team *Team, globalLocation *time.Location, mod *service) *TeamState {
	state := &TeamState{
		Team:              team,
		service:           mod,
		questionSetStates: map[*QuestionSet]*questionSetState{},
	}

	state.Cron = cron.NewWithLocation(team.Timezone)

	initTeamState(team, state)

	return state
}

func initTeamState(team *Team, state *TeamState) {
	for _, qs := range team.QuestionsSets {
		state.questionSetStates[qs] = emptyQuestionSetState(qs)
		state.Cron.Schedule(qs.ReportSchedule, &ScrumReportJob{state, qs})
		state.Cron.Schedule(newScheduleDependentSchedule(qs.ReportSchedule, qs.FirstReminderBeforeReport), &ScrumReminderJob{First, state, qs})
		state.Cron.Schedule(newScheduleDependentSchedule(qs.ReportSchedule, qs.LastReminderBeforeReport), &ScrumReminderJob{Last, state, qs})
	}
	state.Cron.Start()
}

// scheduleDependentSchedule is a schedule that depends on another one to trigger.
type scheduleDependentSchedule struct {
	cron.Schedule
	time.Duration

	depNext time.Time
}

func newScheduleDependentSchedule(s cron.Schedule, t time.Duration) *scheduleDependentSchedule {
	return &scheduleDependentSchedule{s, t, time.Time{}}
}

func (s *scheduleDependentSchedule) Next(t time.Time) time.Time {
	// On init, this is not done in the constructor because Next is only called when the Cron is started
	if s.depNext.IsZero() {
		s.depNext = s.Schedule.Next(t)
	}

	if s.depNext.Add(s.Duration).Before(t) {
		s.depNext = s.Schedule.Next(s.depNext)
	}

	return s.depNext.Add(s.Duration)
}

func (m *service) GetTeams() []string {
	teams := []string{}
	for _, ts := range m.teamStates {
		teams = append(teams, ts.Name)
	}

	return teams
}

func (m *service) GetTeamsForUser(username string) []string {
	teams := []string{}
	for _, ts := range m.teamStates {
		for _, member := range ts.Members {
			if username == member {
				teams = append(teams, ts.Name)
			}
		}
	}

	return teams
}

func (m *service) GetTeamByName(teamName string) (*TeamState, error) {
	for _, ts := range m.teamStates {
		if teamName == ts.Team.Name {
			return ts, nil
		}
	}
	return nil, errors.New("Team " + teamName + " does not exist")
}

func (m *service) GetQuestionSetsForTeam(team string) []*QuestionSet {
	return m.teamStates[team].QuestionsSets
}

func (m *service) GetUsersForTeam(team string) []string {
	return m.teamStates[team].Members
}

func (m *service) SaveReport(report *Report, qs *QuestionSet) {
	m.lastEnteredReport[report.User] = report
	m.teamStates[report.Team].questionSetStates[qs].enteredReports[report.User] = report

	// if done launch report answers
	if len(m.teamStates[report.Team].Members) == len(m.teamStates[report.Team].questionSetStates[qs].enteredReports) {
		m.teamStates[report.Team].sendReportForTeam(qs)
	}
}

func (m *service) DeleteLastReport(user string) bool {

	r, ok := m.lastEnteredReport[user]
	if !ok {
		return false
	}
	delete(m.lastEnteredReport, user)

	ts, ok := m.teamStates[r.Team]
	if !ok {
		return false
	}

	for _, qs := range ts.questionSetStates {
		report, ok := qs.enteredReports[r.User]
		if ok && r == report {
			delete(qs.enteredReports, r.User)
			return true
		}
	}

	return false
}

func (m *service) AddToOutOfOffice(team string, username string) {
	m.teamStates[team].OutOfOffice = append(m.teamStates[team].OutOfOffice, username)

	m.saveConfig()
}

func (m *service) RemoveFromOutOfOffice(team string, username string) {
	var ooof []string
	for _, outOfOfficeMember := range m.teamStates[team].OutOfOffice {
		if outOfOfficeMember != username {
			ooof = append(ooof, outOfOfficeMember)
		}
	}
	m.teamStates[team].OutOfOffice = ooof

	m.saveConfig()
}

func (m *service) AddToTeam(team string, username string) {
	m.teamStates[team].Members = append(m.teamStates[team].Members, username)

	m.saveConfig()
}

func (m *service) RemoveFromTeam(team string, username string) {
	var members []string
	for _, member := range m.teamStates[team].Members {
		if member != username {
			members = append(members, member)
		}
	}
	m.teamStates[team].Members = members

	m.saveConfig()
}

func (m *service) AddTeam(team *Team) {
	location, _ := time.LoadLocation(m.getCurrentConfig().Timezone)
	team.Timezone = location
	state := initTeamStateWithLocation(team, location, m)
	m.teamStates[team.Name] = state

	m.saveConfig()
}

func (m *service) DeleteTeam(team string) {
	m.teamStates[team].Cron.Stop()
	delete(m.teamStates, team)

	m.saveConfig()
}

func (m *service) ReplaceScrumScheduleInTeam(team string, schedule cron.Schedule, scheduleAsString string) {
	m.teamStates[team].Team.QuestionsSets[0].ReportSchedule = schedule
	m.teamStates[team].Team.QuestionsSets[0].ReportScheduleCron = scheduleAsString

	m.teamStates[team].Cron.Stop()
	m.teamStates[team].Cron = cron.NewWithLocation(m.teamStates[team].Team.Timezone)
	initTeamState(m.teamStates[team].Team, m.teamStates[team])

	m.saveConfig()
}

func (m *service) ReplaceFirstReminderInTeam(team string, duration time.Duration) {
	m.teamStates[team].Team.QuestionsSets[0].FirstReminderBeforeReport = duration

	m.teamStates[team].Cron.Stop()
	m.teamStates[team].Cron = cron.NewWithLocation(m.teamStates[team].Team.Timezone)
	initTeamState(m.teamStates[team].Team, m.teamStates[team])

	m.saveConfig()
}

func (m *service) ReplaceLastReminderInTeam(team string, duration time.Duration) {
	m.teamStates[team].Team.QuestionsSets[0].LastReminderBeforeReport = duration

	m.teamStates[team].Cron.Stop()
	m.teamStates[team].Cron = cron.NewWithLocation(m.teamStates[team].Team.Timezone)
	initTeamState(m.teamStates[team].Team, m.teamStates[team])

	m.saveConfig()
}

func (m *service) ReplaceScrumQuestionsInTeam(team string, questions []string) {
	m.teamStates[team].Cron.Stop()
	m.teamStates[team].Cron = cron.NewWithLocation(m.teamStates[team].Team.Timezone)

	m.teamStates[team].Team.QuestionsSets[0].Questions = questions
	initTeamState(m.teamStates[team].Team, m.teamStates[team])

	m.saveConfig()
}

func (m *service) ChangeTeamChannel(team string, channel string) {
	m.teamStates[team].Channel = channel

	m.saveConfig()
}
