package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/coveord/scrumpolice/scrum"
	"github.com/nlopes/slack"
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
		return b.startScrum(event, false)
	}

	if strings.HasPrefix(strings.ToLower(event.Text), "skip") {
		return b.startScrum(event, true)
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
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Nothing to restart, let's get out.", false), AsUser)
		return false
	}

	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Your last report was deleted, you can `start scrum` a new one again.", false), AsUser)
	return false
}

func (b *Bot) startScrum(event *slack.MessageEvent, isSkipped bool) bool {
	// can we infer team (aka does the user only has one team)
	// b.scrum.GetTeamForUser(event.User)
	user, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("There was an error starting your scrum, please try again.", false), AsUser)
		return false
	}
	username := user.Name

	teams := b.scrum.GetTeamsForUser(username)
	if len(teams) == 0 {
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("You're not part of a team, no point in doing a scrum report.", false), AsUser)
	}

	if len(teams) == 1 {
		return b.choosenTeam(event, username, teams[0], isSkipped)
	}

	return b.chooseTeam(event, username, teams, isSkipped)
}

func (b *Bot) chooseTeam(event *slack.MessageEvent, username string, teams []string, isSkipped bool) bool {
	choices := make([]string, len(teams))
	sort.Strings(teams)
	for i, team := range teams {
		// i+1 is included because if the user inputs "0", it becomes "" and throws an error during `start scrum`
		choices[i] = fmt.Sprintf("%d - %s", i+1, team)
	}

	msg := fmt.Sprintf("Choose your team :\n%s", strings.Join(choices, "\n"))
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		i, err := strconv.Atoi(event.Text)
		// len(teams+1) related to above comment
		if i < 1 || i >= len(teams)+1 || err != nil {
			b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Wrong choice, try again or type 'quit'", false), AsUser)
			b.chooseTeam(event, username, teams, isSkipped)
			return false
		}
		// i-1 is related to above comments
		return b.choosenTeam(event, username, teams[i-1], isSkipped)
	}))

	return false
}

func (b *Bot) choosenTeam(event *slack.MessageEvent, username string, team string, isSkipped bool) bool {
	qs := b.scrum.GetQuestionSetsForTeam(team)

	if len(qs) == 0 {
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Your team has no questions defined.", false), AsUser)
		return false
	}

	if len(qs) == 1 {
		return b.choosenTeamAndContext(event, username, team, qs[0], isSkipped)
	}

	return b.chooseContext(event, username, team, qs, isSkipped)
	// get the questionset (if more than one)
}

func (b *Bot) chooseContext(event *slack.MessageEvent, username string, team string, questionSets []*scrum.QuestionSet, isSkipped bool) bool {
	choices := make([]string, len(questionSets))
	for i, questionSet := range questionSets {
		choices[i] = fmt.Sprintf("%d - %s", i, strings.Join(questionSet.Questions, " & "))
	}

	msg := fmt.Sprintf("Choose your set of Questions to answer :\n%s", strings.Join(choices, "\n"))
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		i, err := strconv.Atoi(event.Text)

		if i < 0 || i >= len(questionSets) || err != nil {
			b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Wrong choice, try again or type `quit`", false), AsUser)
			b.chooseContext(event, username, team, questionSets, isSkipped)
			return false
		}

		return b.choosenTeamAndContext(event, username, team, questionSets[i], isSkipped)
	}))

	return false
}

func (b *Bot) choosenTeamAndContext(event *slack.MessageEvent, username string, team string, questionSet *scrum.QuestionSet, isSkipped bool) bool {
	if isSkipped {
		msg := fmt.Sprintf("Scrum report skipped for %s in team %s, type `restart scrum` if it should not be skipped.", username, team)
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

		b.scrum.SaveReport(&scrum.Report{
			User:    username,
			Team:    team,
			Skipped: true,
			Answers: map[string]string{},
		}, questionSet)
		b.unsetUserContext(event.User)
		return false
	}

	msg := fmt.Sprintf("Scrum report started %s for team %s, type `quit` anytime to stop.", username, team)
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

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
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Thanks for your scrum report my :deer:! :bear: with us for the digest. :owl: see you later!\n If you want to start again just say `restart scrum`", false), AsUser)
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
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(question, false), AsUser)

	ctx := b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		report.Answers[question] = event.Text
		return b.answerQuestions(event, questionSet, report)
	})

	b.setUserContext(event.User, ctx)

	return false
}
