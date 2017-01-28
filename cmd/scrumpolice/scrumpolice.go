package main

import (
	"flag"
	"fmt"
	"log"

	"math/rand"
	"time"

	"github.com/scrumpolice/scrumpolice"
)

const header = "                                           _ _\n" +
	" ___  ___ _ __ _   _ _ __ ___  _ __   ___ | (_) ___ ___\n" +
	"/ __|/ __| '__| | | | '_ ` _ \\| '_ \\ / _ \\| | |/ __/ _ \\\n" +
	"\\__ \\ (__| |  | |_| | | | | | | |_) | (_) | | | (_|  __/\n" +
	"|___/\\___|_|   \\__,_|_| |_| |_| .__/ \\___/|_|_|\\___\\___|\n" +
	"                              |_|"

func main() {
	configFile := "config.toml"
	flag.StringVar(&configFile, "config", configFile, "the configuration file to load/use")

	flag.Parse()

	fmt.Println(header)
	fmt.Println("Version", scrumpolice.Version)
	fmt.Println("")

	rand.Seed(time.Now().UTC().UnixNano())

	configStore, err := scrumpolice.NewTOMLConfigStore(configFile)
	if err != nil {
		log.Fatalln("Cannot Load configuration", configFile, err)
	}

	scrumpolice.Run(configStore)
}
