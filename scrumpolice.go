package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nlopes/slack"
	"github.com/pastjean/scrumpolice/bot"
	"github.com/pastjean/scrumpolice/scrum"
	"github.com/sirupsen/logrus"
)

const header = "                                           _ _\n" +
	" ___  ___ _ __ _   _ _ __ ___  _ __   ___ | (_) ___ ___\n" +
	"/ __|/ __| '__| | | | '_ ` _ \\| '_ \\ / _ \\| | |/ __/ _ \\\n" +
	"\\__ \\ (__| |  | |_| | | | | | | |_) | (_) | | | (_|  __/\n" +
	"|___/\\___|_|   \\__,_|_| |_| |_| .__/ \\___/|_|_|\\___\\___|\n" +
	"                              |_|"

const Version = "0.7.1"

func main() {
	fmt.Println(header)
	fmt.Println("Version", Version)
	fmt.Println("")

	slackBotToken := os.Getenv("SCRUMPOLICE_SLACK_TOKEN")

	if slackBotToken == "" {
		log.Fatalln("slack bot token must be set in SCRUMPOLICE_SLACK_TOKEN")
	}


	logger := logrus.New()

	permanentConfigFile := "db.json"
	flag.StringVar(&permanentConfigFile, "databaseFile", permanentConfigFile, "The permanent database file")
	flag.Parse()
	configFile := "config.json"
	flag.StringVar(&configFile, "config", configFile, "The configuration file")
	flag.Parse()

	slackAPIClient := slack.New(slackBotToken)
	scrumService := scrum.NewService(initConfig(configFile, permanentConfigFile), slackAPIClient)

	// Create and run bot
	b := bot.New(slackAPIClient, logger, scrumService)
	b.Run()
}

func initConfig(configFileName string, permanentDbFileName string) scrum.ConfigurationStorage  {
	var configStorage = scrum.NewFileConfigurationStorage(&permanentDbFileName)

	if _, err := os.Stat(permanentDbFileName); os.IsNotExist(err) {
		log.Println("Permanent config file does not exist. Will try to copy other config file if it exists. ")
		configStorage.Save(scrum.NewFileConfigurationStorage(&configFileName).Load())
	}
	if configStorage.Load() == nil {
		log.Fatal("Could not load proper configuration. Will not boot.")
	}
	return configStorage
}