package main

import (
	"flag"
	"fmt"
	"log"

	"math/rand"
	"time"

	"github.com/scrumpolice/scrumpolice"
)

const header = " ____                            ____        _\n" +
	"/ ___|  ___ _ __ _   _ _ __ ___ | __ )  ___ | |_\n" +
	"\\___ \\ / __| '__| | | | '_ ` _ \\|  _ \\ / _ \\| __|\n" +
	" ___) | (__| |  | |_| | | | | | | |_) | (_) | |_\n" +
	"|____/ \\___|_|   \\__,_|_| |_| |_|____/ \\___/ \\__|"

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
