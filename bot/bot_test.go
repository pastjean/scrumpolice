package bot

import "testing"
import "github.com/slack-go/slack"

func TestHandleMessageIgnoreBotMessages(t *testing.T) {
	bot := Bot{}

	bot.handleMessage(&slack.MessageEvent{Msg: slack.Msg{BotID: "test"}})
}

func TestCanTellIfMessageAddressedToBotByIdWithCorrectSyntax(t *testing.T) {
	bot := Bot{id: "scrumpolice",
		name: "Sylvain"}

	result := bot.addressedToMe("<@scrumpolice> hello")
	if !result {
		t.Fail()
	}
}

func TestCanTellIfMessageAddressedToBotByIdWithCorrectSyntaxButNotPrefixing(t *testing.T) {
	bot := Bot{id: "scrumpolice",
		name: "Sylvain"}

	result := bot.addressedToMe("oh, it's you <@scrumpolice> hello")
	if result {
		t.Fail()
	}
}

func TestCanTellIfMessageAddressedToBotByIdWithWrongPrefix(t *testing.T) {
	bot := Bot{id: "scrumpolice",
		name: "Sylvain"}

	result := bot.addressedToMe("@scrumpolice hello")
	if result {
		t.Fail()
	}
}

func TestCanTellIfMessageAddressedToBotByNameWithCorrectSyntax(t *testing.T) {
	bot := Bot{id: "scrumpolice",
		name: "Sylvain"}

	result := bot.addressedToMe("sylvain hello")
	if !result {
		t.Fail()
	}
}

func TestCanTellIfMessageAddressedToBotByNameWithWrongName(t *testing.T) {
	bot := Bot{id: "scrumpolice",
		name: "Sylvain"}

	result := bot.addressedToMe("Rod Stewart hello")
	if result {
		t.Fail()
	}
}

func TestCanTellIfMessageAddressedToBotByNameWithCorrectSyntaxButNotPrefixing(t *testing.T) {
	bot := Bot{id: "scrumpolice",
		name: "Sylvain"}

	result := bot.addressedToMe("oh, it's you sylvain hello")
	if result {
		t.Fail()
	}
}

func TestTrimBotNameAndIdFromMessage(t *testing.T) {
	bot := Bot{id: "scrumpolice",
		name: "Sylvain"}

	trimmedMessage := bot.trimBotNameInMessage("<@scrumpolice> sylvain it's you!!")
	if trimmedMessage != "it's you!!" {
		t.Fail()
	}
}

func TestHandleConnectedSetsBotNameAndId(t *testing.T) {
	bot := Bot{id: "scrumpolice",
		name: "Sylvain"}

	bot.handleConnected(&slack.ConnectedEvent{Info: &slack.Info{User: &slack.UserDetails{ID: "notscrumpolice", Name: "obiwan"}}})

	if bot.id != "notscrumpolice" || bot.name != "obiwan" {
		t.Fail()
	}
}
