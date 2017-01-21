package scrumpolice

import (
	"github.com/k0kubun/pp"
	"github.com/scrumpolice/scrumpolice/chat"
	"github.com/scrumpolice/scrumpolice/chat/slack"
)

func Run(configStore ConfigStore) {
	newScrumPolice(configStore).Start()
}

type scrumPolice struct {
	configStore          ConfigStore
	chatAdapter          chat.Adapter
	receivedMessagesChan chan chat.Message
}

func newScrumPolice(configStore ConfigStore) *scrumPolice {
	receivedMessagesChan := make(chan chat.Message)
	config, _ := configStore.Read()
	chatAdapter := slack.NewChatAdapter(config.SlackToken, receivedMessagesChan)

	return &scrumPolice{configStore, chatAdapter, receivedMessagesChan}
}

func (bot *scrumPolice) Start() {
	go bot.chatAdapter.Run()

	for {
		select {
		case msg := <-bot.receivedMessagesChan:
			// TODO implement logic in here
			pp.Println(msg, "hello")
		}
	}
}

func (bot *scrumPolice) Stop() {
	bot.chatAdapter.Disconnect()
}
