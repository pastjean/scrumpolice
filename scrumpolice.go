package scrumpolice

import (
	"log"
	"os"

	"github.com/k0kubun/pp"
	"github.com/nlopes/slack"
)

func Run(configStore ConfigStore) {
	newScrumPolice(configStore).Start()
}

type scrumPolice struct {
	configStore          ConfigStore
	chatAdapter          ChatAdapter
	receivedMessagesChan chan Message
}

func init() {
	logger := log.New(os.Stdout, "scrum-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)
}

func newScrumPolice(configStore ConfigStore) *scrumPolice {
	receivedMessagesChan := make(chan Message)

	chatAdapter := NewSlackAdapter(configStore, receivedMessagesChan)

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
