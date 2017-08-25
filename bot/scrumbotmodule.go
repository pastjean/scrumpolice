package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
	"github.com/pastjean/scrumpolice/scrum"
	log "github.com/sirupsen/logrus"
)

// HandleMessage handle a received message for scrums and returns if the bot shall continue to process the message or stop
// continue = true
// stop = false
func (b *Bot) HandleScrumMessage(event *slack.MessageEvent) bool {
	// "start scrum [team] [date]"
	// [team == first and only team]
	// [date == yesterday]
	// starting scrum for team [team] date [date]. if you want to abort say stop scrum

	// this module only takes case in private messages
	if event.Channel[0] != 'D' {
		return true
	}

	if context, ok := b.userContexts[event.User]; ok {
		return context.HandleMessage(event)
	}

	if strings.HasPrefix(strings.ToLower(event.Text), "start scrum") {
		return b.startScrum(event)
	}

	if strings.ToLower(event.Text) == "restart scrum" {
		return b.restartScrum(event)
	}

	return true
}

func (b *Bot) restartScrum(event *slack.MessageEvent) bool {
	user, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		return false
	}

	if !b.scrum.DeleteLastReport(user.Name) {
		b.slackBotAPI.PostMessage(event.Channel, "Nothing to restart, let's get out", slack.PostMessageParameters{AsUser: true})
		return false
	}

	b.slackBotAPI.PostMessage(event.Channel, "Your last report was deleted, you can `start scrum` a new one again", slack.PostMessageParameters{AsUser: true})
	return false
}

func (b *Bot) startScrum(event *slack.MessageEvent) bool {
	// can we infer team (aka does the user only has one team)
	// b.scrum.GetTeamForUser(event.User)
	user, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		b.slackBotAPI.PostMessage(event.Channel, "There was an error starting your scrum, please try again", slack.PostMessageParameters{AsUser: true})
		return false
	}
	username := user.Name

	teams := b.scrum.GetTeamsForUser(username)
	if len(teams) == 0 {
		b.slackBotAPI.PostMessage(event.Channel, "You're not part of a team, no point in doing a scrum report", slack.PostMessageParameters{AsUser: true})
	}

	if len(teams) == 1 {
		return b.choosenTeam(event, username, teams[0])
	}

	return b.chooseTeam(event, username, teams)
}

func (b *Bot) chooseTeam(event *slack.MessageEvent, username string, teams []string) bool {
	choices := make([]string, len(teams))
	sort.Strings(teams)
	for i, team := range teams {
		choices[i] = fmt.Sprintf("%d - %s", i, team)
	}

	msg := fmt.Sprintf("Choose your team :\n%s", strings.Join(choices, "\n"))
	b.slackBotAPI.PostMessage(event.Channel, msg, slack.PostMessageParameters{AsUser: true})

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		i, err := strconv.Atoi(event.Text)

		if i < 0 || i >= len(teams) || err != nil {
			b.slackBotAPI.PostMessage(event.Channel, "Wrong choices, please try again :p or type `quit`", slack.PostMessageParameters{AsUser: true})
			b.chooseTeam(event, username, teams)
			return false
		}

		return b.choosenTeam(event, username, teams[i])
	}))

	return false
}

func (b *Bot) choosenTeam(event *slack.MessageEvent, username string, team string) bool {
	qs := b.scrum.GetQuestionSetsForTeam(team)

	if len(qs) == 0 {
		b.slackBotAPI.PostMessage(event.Channel, "Your team has no questions defined", slack.PostMessageParameters{AsUser: true})
		return false
	}

	if len(qs) == 1 {
		return b.choosenTeamAndContext(event, username, team, qs[0])
	}

	return b.chooseContext(event, username, team, qs)
	// get the questionset (if more than one)
}

func (b *Bot) chooseContext(event *slack.MessageEvent, username string, team string, questionSets []*scrum.QuestionSet) bool {
	choices := make([]string, len(questionSets))
	for i, questionSet := range questionSets {
		choices[i] = fmt.Sprintf("%d - %s", i, strings.Join(questionSet.Questions, " & "))
	}

	msg := fmt.Sprintf("Choose your set of Questions to answer :\n%s", strings.Join(choices, "\n"))
	b.slackBotAPI.PostMessage(event.Channel, msg, slack.PostMessageParameters{AsUser: true})

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		i, err := strconv.Atoi(event.Text)

		if i < 0 || i >= len(questionSets) || err != nil {
			b.slackBotAPI.PostMessage(event.Channel, "Wrong choices, please try again :p or type `quit`", slack.PostMessageParameters{AsUser: true})
			b.chooseContext(event, username, team, questionSets)
			return false
		}

		return b.choosenTeamAndContext(event, username, team, questionSets[i])
	}))

	return false
}

func (b *Bot) choosenTeamAndContext(event *slack.MessageEvent, username string, team string, questionSet *scrum.QuestionSet) bool {
	msg := fmt.Sprintf("Scrum report started %s for team %s, type `quit` anytime to stop", username, team)
	b.slackBotAPI.PostMessage(event.Channel, msg, slack.PostMessageParameters{AsUser: true})

	return b.answerQuestions(event, questionSet, &scrum.Report{
		User:    username,
		Team:    team,
		Answers: map[string]string{},
	})
}

func (b *Bot) answerQuestions(event *slack.MessageEvent, questionSet *scrum.QuestionSet, report *scrum.Report) bool {
	ans := len(report.Answers)
	// We're finished, are we ?
	if ans == len(questionSet.Questions) {
		b.scrum.SaveReport(report, questionSet)
		b.slackBotAPI.PostMessage(event.Channel, "Thanks for your scrum report my :deer:! :bear: with us for the digest. :owl: see you later!\n If you want to start again just say `restart scrum`", slack.PostMessageParameters{AsUser: true})
		b.unsetUserContext(event.User)
		b.logger.WithFields(log.Fields{
			"user": report.User,
			"team": report.Team,
		}).Info("All questions anwsered, entry saved.")
		return false
	}

	return b.questionsOut(event, questionSet, report)
}

func (b *Bot) questionsOut(event *slack.MessageEvent, questionSet *scrum.QuestionSet, report *scrum.Report) bool {
	question := questionSet.Questions[len(report.Answers)]
	b.slackBotAPI.PostMessage(event.Channel, question, slack.PostMessageParameters{AsUser: true})

	ctx := b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		report.Answers[question] = event.Text
		return b.answerQuestions(event, questionSet, report)
	})

	b.setUserContext(event.User, ctx)

	return false
}
