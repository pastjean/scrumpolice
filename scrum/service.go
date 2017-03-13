package scrum

import (
	"fmt"
	"log"
	"strings"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/nlopes/slack"
	"github.com/robfig/cron"
)

var (
	SlackParams = slack.PostMessageParameters{AsUser: true}
)

type Service interface {
	DeleteLastReport(username string) bool
	GetTeamsForUser(username string) []string
	GetQuestionSetsForTeam(team string) []*QuestionSet
	SaveReport(report *Report, qs *QuestionSet)
}

type service struct {
	configurationProvider ConfigurationProvider
	teamStates            map[string]*TeamState
	slackBotAPI           *slack.Client
	lastEnteredReport     map[string]*Report
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
	User string
	Team string
	// questions / answers
	Answers map[string]string
}

func emptyQuestionSetState(qs *QuestionSet) *questionSetState {
	return &questionSetState{qs, map[string]*Report{}, false}
}

func (ts *TeamState) sendReportForTeam(qs *QuestionSet) {
	qsstate := ts.questionSetStates[qs]
	if qsstate.sent == true {
		return
	}
	qsstate.sent = true

	if len(qsstate.enteredReports) == 0 {
		ts.service.slackBotAPI.PostMessage(ts.Channel, "I'd like to take time to :shame: everyone for not reporting", SlackParams)
		return
	}

	ts.service.slackBotAPI.PostMessage(ts.Channel, ":parrotcop: Alrighty! Here's the scrum report for today!", SlackParams)

	attachments := make([]slack.Attachment, 0, len(qsstate.enteredReports))
	didNotDoReport := []string{}
	for _, member := range ts.Members {
		report, ok := qsstate.enteredReports[member]
		if !ok {
			didNotDoReport = append(didNotDoReport, member)
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

	params := slack.PostMessageParameters{
		AsUser:      true,
		Attachments: attachments,
	}

	if len(didNotDoReport) > 0 {
		ts.service.slackBotAPI.PostMessage(ts.Channel, fmt.Sprintln("And lastly we should take a little time to shame", didNotDoReport), params)
	}
}

func (ts *TeamState) sendFirstReminder(qs *QuestionSet) {
	qsstate := ts.questionSetStates[qs]

	for _, member := range ts.Members {
		_, ok := qsstate.enteredReports[member]
		if !ok {
			ts.service.slackBotAPI.PostMessage("@"+member, "Hey! Don't forget to fill your report! `start scrum` to do it", SlackParams)
		}
	}
}

func (ts *TeamState) sendLastReminder(qs *QuestionSet) {
	qsstate := ts.questionSetStates[qs]
	didNotDoReport := []string{}
	for _, member := range ts.Members {
		_, ok := qsstate.enteredReports[member]
		if !ok {
			didNotDoReport = append(didNotDoReport, member)
		}
	}

	if len(didNotDoReport) == 0 {
		return
	}

	memberThatDidNotDoReport := strings.Join(didNotDoReport, ", ")
	ts.service.slackBotAPI.PostMessage(ts.Channel, fmt.Sprintf("Last chance to fill report! :shame: to: %s", memberThatDidNotDoReport), SlackParams)
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

func NewService(configurationProvider ConfigurationProvider, slackBotAPI *slack.Client) Service {
	mod := &service{
		configurationProvider: configurationProvider,
		slackBotAPI:           slackBotAPI,
		teamStates:            map[string]*TeamState{},
		lastEnteredReport:     map[string]*Report{},
	}

	// initial *refresh
	mod.refresh(configurationProvider.Config())

	configurationProvider.OnChange(func(cfg *Config) {
		log.Println("Configuration File Changed refreshing state")
		mod.refresh(cfg)
	})

	return mod
}

func (mod *service) refresh(config *Config) {
	teams := config.ToTeams()

	globalLocation := time.Local
	if config.Timezone != "" {
		l, err := time.LoadLocation(config.Timezone)
		if err == nil {
			globalLocation = l
		} else {
			log.Println("error loading global location, using default", err)
		}
	}

	for _, team := range teams {
		state, ok := mod.teamStates[team.Name]
		if !ok {
			log.Println("Initializing team", team.Name)
		} else {
			// FIXME: Check if team changed before doing that
			log.Println("Refreshing team", team.Name)
			state.Cron.Stop()
		}
		state = initTeamState(team, globalLocation, mod)
		mod.teamStates[team.Name] = state
	}
}

func initTeamState(team *Team, globalLocation *time.Location, mod *service) *TeamState {
	state := &TeamState{
		Team:              team,
		service:           mod,
		questionSetStates: map[*QuestionSet]*questionSetState{},
	}

	loc := globalLocation
	if team.Timezone != nil {
		loc = team.Timezone
	}
	state.Cron = cron.NewWithLocation(loc)

	for _, qs := range team.QuestionsSets {
		state.questionSetStates[qs] = emptyQuestionSetState(qs)
		state.Cron.Schedule(qs.ReportSchedule, &ScrumReportJob{state, qs})
		state.Cron.Schedule(newScheduleDependentSchedule(qs.ReportSchedule, qs.FirstReminderBeforeReport), &ScrumReminderJob{First, state, qs})
		state.Cron.Schedule(newScheduleDependentSchedule(qs.ReportSchedule, qs.LastReminderBeforeReport), &ScrumReminderJob{Last, state, qs})
	}

	state.Cron.Start()

	return state
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

func (m *service) GetQuestionSetsForTeam(team string) []*QuestionSet {
	return m.teamStates[team].QuestionsSets
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
