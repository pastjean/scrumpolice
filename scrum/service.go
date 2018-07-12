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

var SlackParams = slack.PostMessageParameters{AsUser: true, LinkNames: 1}

type Service interface {
	DeleteLastReport(username string) bool
	GetTeamByName(teamName string) (*TeamState, error)
	GetTeamsForUser(username string) []string
	GetQuestionSetsForTeam(team string) []*QuestionSet
	SaveReport(report *Report, qs *QuestionSet)
	AddToOutOfOffice(team string, username string)
	RemoveFromOutOfOffice(team string, username string)
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

func (ts *TeamState) postMessageToSlack(channel string, message string, params slack.PostMessageParameters) {
	_, _, err := ts.service.slackBotAPI.PostMessage(channel, message, params)
	if err != nil {
		log.WithFields(log.Fields{
			"team":    ts.Team.Name,
			"channel": channel,
			"error":   err,
		}).Warn("Error while posting message to slack")
	}
}

func (ts *TeamState) sendReportForTeam(qs *QuestionSet) {
	qsstate := ts.questionSetStates[qs]
	if qsstate.sent == true {
		return
	}
	qsstate.sent = true

	if len(qsstate.enteredReports) == 0 {
		ts.postMessageToSlack(ts.Channel, "I'd like to take time to :shame: everyone for not reporting", SlackParams)
		return
	}

	attachments := []slack.Attachment{}
	didNotDoReport := []string{}
	for _, member := range ts.Members {
		report, ok := qsstate.enteredReports[member]
		if !ok {
			if isMemberOutOfOffice(ts, member) {
				attachment := slack.Attachment{
					Color:      colorful.FastHappyColor().Hex(),
					MarkdownIn: []string{"text", "pretext"},
					Pretext:    "@" + member,
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
				Pretext:    "@" + member,
				Text:       "Has nothing to declare (most probably :bee:cause he did nothing :troll:)",
			}
			attachments = append(attachments, attachment)
		} else {
			message := ""
			for idx, q := range qsstate.QuestionSet.Questions {
				message += q + "\n" + report.Answers[q]

				if idx < len(qsstate.QuestionSet.Questions)-1 {
					message += "\n\n"
				}
			}

			attachment := slack.Attachment{
				Color:      colorful.FastHappyColor().Hex(),
				MarkdownIn: []string{"text", "pretext"},
				Pretext:    "@" + member,
				Text:       message,
			}
			attachments = append(attachments, attachment)
		}

	}

	if ts.SplitReport {
		ts.postMessageToSlack(ts.Channel, ":parrotcop: Alrighty! Here's the scrum report for today!", slack.PostMessageParameters{AsUser: true})
		for i := 0; i < len(attachments); i++ {
			params := slack.PostMessageParameters{
				AsUser:      true,
				LinkNames:   1,
				Attachments: []slack.Attachment{attachments[i]},
			}
			ts.postMessageToSlack(ts.Channel, "*Scrum by:*", params)
		}
	} else {
		params := slack.PostMessageParameters{
			AsUser:      true,
			LinkNames:   1,
			Attachments: attachments,
		}
		ts.postMessageToSlack(ts.Channel, ":parrotcop: Alrighty! Here's the scrum report for today!", params)
	}

	if len(didNotDoReport) > 0 {
		ts.postMessageToSlack(ts.Channel, fmt.Sprintln("And lastly we should take a little time to shame", didNotDoReport), SlackParams)
	}

	log.WithFields(log.Fields{
		"team":    ts.Team.Name,
		"channel": ts.Channel,
	}).Info("Sent scrum report.")
}

func (ts *TeamState) sendFirstReminder(qs *QuestionSet) {
	qsstate := ts.questionSetStates[qs]

	log.WithFields(log.Fields{
		"team":    ts.Team.Name,
		"channel": ts.Channel,
	}).Info("Sending first reminder.")

	for _, member := range ts.Members {
		if !isMemberOutOfOffice(ts, member) {
			_, ok := qsstate.enteredReports[member]
			if !ok {
				_, _, err := ts.service.slackBotAPI.PostMessage("@"+member, "Hey! Don't forget to fill your report! `start` to do it or `skip` if you have nothing to say", SlackParams)
				if err != nil {
					log.WithFields(log.Fields{
						"team":    ts.Team.Name,
						"member":  member,
						"channel": ts.Channel,
						"error":   err,
					}).Warn("Could not send first reminder.")
				}
			}
		} else {
			log.WithFields(log.Fields{
				"team":    ts.Team.Name,
				"member":  member,
				"channel": ts.Channel,
			}).Info("Member out of office, not sending reminder.")
		}
	}
}

func (ts *TeamState) sendLastReminder(qs *QuestionSet) {
	qsstate := ts.questionSetStates[qs]
	didNotDoReport := []string{}

	log.WithFields(log.Fields{
		"team":    ts.Team.Name,
		"channel": ts.Channel,
	}).Info("Sending last reminder.")

	for _, member := range ts.Members {
		if !isMemberOutOfOffice(ts, member) {
			_, ok := qsstate.enteredReports[member]
			if !ok {
				didNotDoReport = append(didNotDoReport, "@"+member)
			}
		}
	}

	if len(didNotDoReport) == 0 {
		return
	}

	memberThatDidNotDoReport := strings.Join(didNotDoReport, ", ")
	ts.postMessageToSlack(ts.Channel, fmt.Sprintf("Last chance to fill report! :shame: to: %s", memberThatDidNotDoReport), SlackParams)
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

	log.Info("Refreshing teams.")

	globalLocation := time.Local
	if config.Timezone != "" {
		l, err := time.LoadLocation(config.Timezone)
		if err == nil {
			globalLocation = l
		} else {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Error loading global location, using default.")
		}
	}

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
}

func (m *service) RemoveFromOutOfOffice(team string, username string) {
	var ooof []string
	for _, outOfOfficeMember := range m.teamStates[team].OutOfOffice {
		if outOfOfficeMember != username {
			ooof = append(ooof, outOfOfficeMember)
		}
	}
	m.teamStates[team].OutOfOffice = ooof
}
