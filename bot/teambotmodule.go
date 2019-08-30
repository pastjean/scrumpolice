package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"time"

	"github.com/coveord/scrumpolice/scrum"
	"github.com/nlopes/slack"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

// HandleMessage handle a received message for team and returns if the bot shall continue to process the message or stop
// continue = true
// stop = false
func (b *Bot) HandleTeamEditionMessage(event *slack.MessageEvent) bool {
	// "edit team [name]"
	// [name == name of the team]
	// editing team [name]. if you want to abort say stop scrum

	if context, ok := b.userContexts[event.User]; ok {
		return context.HandleMessage(event)
	}

	if strings.HasPrefix(strings.ToLower(event.Text), "edit team") {
		return b.startTeamEdition(event)
	}

	if strings.HasPrefix(strings.ToLower(event.Text), "add team") {
		return b.startTeamCreation(event)
	}

	if strings.HasPrefix(strings.ToLower(event.Text), "remove team") {
		return b.startTeamDeletion(event)
	}

	return true
}

func (b *Bot) cancelTeamEdition(event *slack.MessageEvent) bool {
	_, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		return false
	}

	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Team edition was cancelled. Better luck next time!", false), AsUser)
	return false
}

func (b *Bot) startTeamDeletion(event *slack.MessageEvent) bool {
	_, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("There was an error editing the team, please try again", false), AsUser)
		return false
	}

	return b.chooseTeamToEdit(event, func(event *slack.MessageEvent, team string) bool {
		return b.chosenTeamToDelete(event, team)
	})
}

func (b *Bot) chosenTeamToDelete(event *slack.MessageEvent, team string) bool {
	expected := "remove team " + team
	msg := "Type `" + expected + "` to delete the team or type `quit`"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		if event.Text == expected {
			author, err := b.slackBotAPI.GetUserInfo(event.User)
			if err != nil {
				b.logSlackRelatedError(event, err, "Fail to get user information.")
				return false
			}

			b.scrum.DeleteTeam(team)

			b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("I've deleted the team "+team, false), AsUser)
			log.WithFields(log.Fields{
				"team":   team,
				"doneBy": author.Name,
			}).Info("Team was deleted.")

			b.unsetUserContext(event.User)
			return false
		}

		return b.chosenTeamToDelete(event, team)
	}))

	return false
}

func (b *Bot) startTeamCreation(event *slack.MessageEvent) bool {
	_, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("There was an error editing the team, please try again", false), AsUser)
		return false
	}

	return b.chooseTeamName(event)
}

func (b *Bot) chooseTeamName(event *slack.MessageEvent) bool {
	msg := "What should be the team name?"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		teams := b.scrum.GetTeams()
		newTeamName := event.Text

		for _, team := range teams {
			if team == newTeamName {
				b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Team already exists, choose a new name or type `quit`", false), AsUser)
				b.chooseTeamName(event)
				return false
			}
		}

		author, err := b.slackBotAPI.GetUserInfo(event.User)
		if err != nil {
			b.logSlackRelatedError(event, err, "Fail to get user information.")
			return false
		}

		members := []string{author.Name}
		defaultCron := "0 5 9 * * MON,WED,FRI"
		var firstReminderBefore time.Duration = -50 * time.Minute
		var lastReminderBefore time.Duration = -5 * time.Minute
		schedule, _ := cron.Parse(defaultCron)

		questions := []*scrum.QuestionSet{&scrum.QuestionSet{
			Questions:                 []string{"What did you do yesterday?", "What will you do today?", "Are you being blocked by someone for a review? who? why?"},
			FirstReminderBeforeReport: firstReminderBefore,
			LastReminderBeforeReport:  lastReminderBefore,
			ReportScheduleCron:        defaultCron,
			ReportSchedule:            schedule,
		}}

		b.scrum.AddTeam(&scrum.Team{
			Name:          newTeamName,
			Channel:       "@" + author.Name,
			Members:       members,
			SplitReport:   true,
			OutOfOffice:   []string{},
			QuestionsSets: questions,
		})

		return b.choosenTeamToEdit(event, newTeamName)
	}))
	return false
}

func (b *Bot) startTeamEdition(event *slack.MessageEvent) bool {
	_, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("There was an error editing the team, please try again", false), AsUser)
		return false
	}

	return b.chooseTeamToEdit(event, func(event *slack.MessageEvent, team string) bool {
		return b.choosenTeamToEdit(event, team)
	})
}

type ChosenTeamFunc func(event *slack.MessageEvent, team string) bool

func (b *Bot) chooseTeamToEdit(event *slack.MessageEvent, cb ChosenTeamFunc) bool {
	user, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		return false
	}

	teams := b.scrum.GetTeamsForUser(user.Name)
	if len(teams) == 0 {
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("There is no teams, use 'add team' to create a new team", false), AsUser)
		return false
	}

	if len(teams) == 1 {
		return cb(event, teams[0])
	}

	choices := make([]string, len(teams))
	sort.Strings(teams)
	for i, team := range teams {
		choices[i] = fmt.Sprintf("%d - %s", i, team)
	}

	msg := fmt.Sprintf("Choose a team :\n%s", strings.Join(choices, "\n"))
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		i, err := strconv.Atoi(event.Text)

		if i < 0 || i >= len(teams) || err != nil {
			b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Wrong choices, please try again :p or type `quit`", false), AsUser)
			b.chooseTeamToEdit(event, cb)
			return false
		}

		return cb(event, teams[i])
	}))

	return false
}

func (b *Bot) choosenTeamToEdit(event *slack.MessageEvent, team string) bool {
	message := slack.Attachment{
		MarkdownIn: []string{"text"},
		Text: "" +
			"- `add @name`: Add *@name* to team\n" +
			"- `remove @name`: Remove *@name* from team\n" +
			"- `edit channel`: Edit the channel in which the scrum is posted\n" +
			"- `edit schedule`: Edit the schedule of the scrum\n" +
			"- `edit first reminder`: Edit the length of time before scrum to at which the users should be warned the first time\n" +
			"- `edit last reminder`: Edit the length of time before scrum to at which the users should be warned the second time\n" +
			"- `edit questions`: Edit the questions asked during scrum\n" +
			"- `quit`: Stop editing and go back to your normal life",
	}

	_, _, err := b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("What do you want to do with team "+team+"?", false), AsUser, slack.MsgOptionAttachments([]slack.Attachment{message}...))
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to post message to slack.")
		return false
	}

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		params := getParams(`(?i)(?P<action>add|remove|edit) <?(?P<entity>.+[^>])>?\s*`, event.Text)
		fmt.Println(params)

		if len(params) == 0 || params["action"] == "" || params["entity"] == "" {
			b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Wrong choices, please try again :p or type `quit`", false), AsUser)
			b.choosenTeamToEdit(event, team)
			return false
		}
		action := params["action"]
		entity := params["entity"]

		// Handle user stuff
		if strings.HasPrefix(entity, "@") && (action == "add" || action == "remove") {
			rawUserName := strings.Replace(entity, "@", "", -1)
			user, err := b.slackBotAPI.GetUserInfo(rawUserName)

			// If the user does not exist, we can still try to remove the user.
			if err != nil && action == "add" {
				b.logSlackRelatedError(event, err, "Fail to get user information.")
				b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Hmmmm, I couldn't find the user. Try again!", false), AsUser)
				b.choosenTeamToEdit(event, team)
				return false
			}
			username := rawUserName
			if user != nil {
				username = user.Name
			}

			return b.ChangeUserAction(event, team, params["action"], username)
		} else if action == "edit" && entity == "schedule" {
			return b.changeScrumSchedule(event, team)

		} else if action == "edit" && entity == "questions" {
			return b.changeScrumQuestions(event, team)

		} else if action == "edit" && entity == "channel" {
			return b.changeTeamChannel(event, team)
		} else if action == "edit" && entity == "first reminder" {
			return b.changeFirstReminder(event, team)
		} else if action == "edit" && entity == "last reminder" {
			return b.changeSecondReminder(event, team)
		}

		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Wrong choices, please try again :p or type `quit`", false), AsUser)
		b.choosenTeamToEdit(event, team)
		return false
	}))

	return false
}

func (b *Bot) changeTeamChannel(event *slack.MessageEvent, team string) bool {
	author, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		return false
	}

	msg := "In which channel should the team " + team + " scrum appear? Don't forget to invite me!"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		newChannel, err := getChannelFromMessage(event.Text)
		if err != nil {
			b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Wrong channel name. You can use `@someuser` or `#somechannel`. Please try again :p or type `quit`", false), AsUser)
			b.changeTeamChannel(event, team)
			return false
		}

		b.scrum.ChangeTeamChannel(team, newChannel)

		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("Done, I've updated the channel for the team "+team+"! I posted a message in the channel. If you didn't see the message, invite me in the channel!", false), AsUser)
		b.slackBotAPI.PostMessage(newChannel, slack.MsgOptionText("The future scrums of the team "+team+" will appear in this channel!", false), AsUser)
		log.WithFields(log.Fields{
			"channel": newChannel,
			"team":    team,
			"doneBy":  author.Name,
		}).Info("Changed team channel.")

		b.unsetUserContext(event.User)
		b.choosenTeamToEdit(event, team)

		return false
	}))

	return false
}

func (b *Bot) changeScrumQuestions(event *slack.MessageEvent, team string) bool {
	_, err := b.slackBotAPI.GetUserInfo(event.User)
	if err != nil {
		b.logSlackRelatedError(event, err, "Fail to get user information.")
		return false
	}

	if b.doesTeamHaveMultipleScrums(event, team) {
		msg := "Team `" + team + "` has multiple scrums. Edition is currently not supported."
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

		return false
	}

	b.printTeamQuestions(event, team)

	return b.changeScrumQuestion(event, team, []string{})
}

func (b *Bot) printTeamQuestions(event *slack.MessageEvent, team string) {
	qs := b.scrum.GetQuestionSetsForTeam(team)[0]
	questions := make([]string, len(qs.Questions))
	for i, question := range qs.Questions {
		questions[i] = fmt.Sprintf("%d - %s", i, question)
	}

	msg := fmt.Sprintf("Current list of questions:\n%s", strings.Join(questions, "\n"))
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
}

func (b *Bot) changeScrumQuestion(event *slack.MessageEvent, team string, questions []string) bool {
	msg := "What is the next question? Type `done` when you are finished"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		if strings.ToLower(event.Text) == "done" {
			b.scrum.ReplaceScrumQuestionsInTeam(team, questions)

			b.printTeamQuestions(event, team)

			b.unsetUserContext(event.User)
			b.choosenTeamToEdit(event, team)

			return false
		}

		questions = append(questions, event.Text)
		return b.changeScrumQuestion(event, team, questions)
	}))

	return false
}

func (b *Bot) ChangeUserAction(event *slack.MessageEvent, team string, action string, username string) bool {
	if action == "add" {
		users := b.scrum.GetUsersForTeam(team)
		for _, user := range users {
			if user == username {
				b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("@"+username+" is already in the team "+team+". Do you need :eyeglasses:?", false), AsUser)
				b.choosenTeamToEdit(event, team)

				return false
			}
		}

		b.scrum.AddToTeam(team, username)

		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("I've added @"+username+" to team "+team, false), AsUser)

		author, err := b.slackBotAPI.GetUserInfo(event.User)
		if err != nil {
			b.logSlackRelatedError(event, err, "Fail to get user information.")
			return false
		}

		b.slackBotAPI.PostMessage("@"+username, slack.MsgOptionText("You've been added in team "+team+" by @"+author.Name+".", false), AsUser)
		log.WithFields(log.Fields{
			"user":   username,
			"team":   team,
			"doneBy": author.Name,
		}).Info("User was added to team.")

	} else if action == "remove" {
		users := b.scrum.GetUsersForTeam(team)
		isInTeam := false
		for _, user := range users {
			isInTeam = isInTeam || user == username
		}

		if !isInTeam {
			b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("@"+username+" is not in the team "+team+". I'm no magician!", false), AsUser)
			b.choosenTeamToEdit(event, team)

			return false
		}

		b.scrum.RemoveFromTeam(team, username)

		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText("I've removed @"+username+" to team "+team, false), AsUser)

		author, err := b.slackBotAPI.GetUserInfo(event.User)
		if err != nil {
			b.logSlackRelatedError(event, err, "Fail to get user information.")
			return false
		}

		b.slackBotAPI.PostMessage("@"+username, slack.MsgOptionText("You've been removed from team "+team+" by @"+author.Name+".", false), AsUser)
		log.WithFields(log.Fields{
			"user":   username,
			"team":   team,
			"doneBy": author.Name,
		}).Info("User was removed from team.")
	}

	b.unsetUserContext(event.User)
	b.choosenTeamToEdit(event, team)

	return false
}

func (b *Bot) changeScrumSchedule(event *slack.MessageEvent, team string) bool {
	if b.doesTeamHaveMultipleScrums(event, team) {
		msg := "Team `" + team + "` has multiple scrums. Edition is currently not supported."
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
	}

	msg := ":warning: *Modifying a scrum schedule while a scrum is in progress will reset all entered scrum reports!* :warning:\n" +
		"Please enter a cron expression for the schedule.\n" +
		"The format is the following:"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
	msg = "```" +
		` ┌───────────── seconds (0 - 59)
 | ┌───────────── minute (0 - 59)
 | │ ┌───────────── hour (0 - 23)
 | │ │ ┌───────────── day of month (1 - 31)
 | │ │ │ ┌───────────── month (1 - 12) - JAN to DEC are also allowed
 | │ │ │ │ ┌───────────── day of week (0 - 6 => Sunday to Saturday) - SUN to SAT are also allowed  
 | │ │ │ │ │                                    
 | │ │ │ │ │
 | │ │ │ │ │
 • • • • • • ` + "```"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
	msg = "For example, a scrum with a deadline of 9:05 AM on monday, wednesday and friday would be `0 5 9 * * MON,WED,FRI`\n" +
		"See https://godoc.org/github.com/robfig/cron for more information!"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		scheduleString := event.Text
		fmt.Println(scheduleString)

		schedule, err := cron.Parse(scheduleString)
		if err != nil {
			b.logSlackRelatedError(event, err, "Sorry, I do not understand the schedule format you sent me. Please try again, or type `quit`")
			b.changeScrumSchedule(event, team)
			return false
		}

		b.scrum.ReplaceScrumScheduleInTeam(team, schedule, scheduleString)
		msg = "Schedule successfully changed!"
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
		b.unsetUserContext(event.User)

		b.choosenTeamToEdit(event, team)

		return false
	}))
	return false
}

func (b *Bot) changeFirstReminder(event *slack.MessageEvent, team string) bool {
	if b.doesTeamHaveMultipleScrums(event, team) {
		msg := "Team `" + team + "` has multiple scrums. Edition is currently not supported."
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
	}

	msg := ":warning: *Modifying a scrum reminder while a scrum is in progress will reset all entered scrum reports!* :warning:\n" +
		"Please enter a duration expression for the schedule.\n" +
		"The format is `-XhYmZs` (for example, `-1m30s`)\n" +
		"See https://golang.org/pkg/time/#ParseDuration for more information!"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		durationString := event.Text
		fmt.Println(durationString)

		if !strings.HasPrefix(durationString, "-") {
			durationString = "-" + durationString
		}
		duration, err := time.ParseDuration(durationString)
		if err != nil {
			b.logSlackRelatedError(event, err, "Sorry, I do not understand the duration format you sent me. Please try again, or type `quit`")
			b.changeFirstReminder(event, team)
			return false
		}

		b.scrum.ReplaceFirstReminderInTeam(team, duration)
		msg = "First reminder duration successfully changed! The users will be warned " + strings.Replace(duration.String(), "-", "", -1) + " before the scrum."
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
		b.unsetUserContext(event.User)
		b.choosenTeamToEdit(event, team)

		return false
	}))
	return false
}

func (b *Bot) changeSecondReminder(event *slack.MessageEvent, team string) bool {
	if b.doesTeamHaveMultipleScrums(event, team) {
		msg := "Team `" + team + "` has multiple scrums. Edition is currently not supported."
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
	}

	msg := ":warning: *Modifying a scrum reminder while a scrum is in progress will reset all entered scrum reports!* :warning:\n" +
		"Please enter a duration expression for the schedule.\n" +
		"The format is `-XhYmZs` (for example, `-1m30s`)\n" +
		"See https://golang.org/pkg/time/#ParseDuration for more information!"
	b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)

	b.setUserContext(event.User, b.canQuitBotContextHandlerFunc(func(event *slack.MessageEvent) bool {
		durationString := event.Text
		fmt.Println(durationString)

		if !strings.HasPrefix(durationString, "-") {
			durationString = "-" + durationString
		}
		duration, err := time.ParseDuration(durationString)
		if err != nil {
			b.logSlackRelatedError(event, err, "Sorry, I do not understand the duration format you sent me. Please try again, or type `quit`")
			b.changeSecondReminder(event, team)
			return false
		}

		b.scrum.ReplaceLastReminderInTeam(team, duration)
		msg = "Last reminder duration successfully changed! The users will be warned " + strings.Replace(duration.String(), "-", "", -1) + " before the scrum."
		b.slackBotAPI.PostMessage(event.Channel, slack.MsgOptionText(msg, false), AsUser)
		b.unsetUserContext(event.User)
		b.choosenTeamToEdit(event, team)

		return false
	}))
	return false
}

func (b *Bot) doesTeamHaveMultipleScrums(event *slack.MessageEvent, team string) bool {
	teamState, err := b.scrum.GetTeamByName(team)
	if err != nil {
		b.logSlackRelatedError(event, err, "Failed to get team information.")
		return false
	}
	return len(teamState.QuestionsSets) > 1
}
