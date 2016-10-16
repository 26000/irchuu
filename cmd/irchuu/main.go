package main

import (
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/db"
	"github.com/26000/irchuu/irc"
	"github.com/26000/irchuu/paths"
	"github.com/26000/irchuu/relay"
	"github.com/26000/irchuu/telegram"
	"log"
	"os"
	"sync"
)

func main() {
	fmt.Printf("IRChuu! v%v (https://github.com/26000/irchuu)\n", config.VERSION)

	configFile, dataDir := paths.GetPaths()
	err := paths.MakePaths(configFile, dataDir)
	if err != nil {
		os.Exit(1)
	}

	log.Printf("Using configuration file: %v\n", configFile)
	log.Printf("Using data directory: %v\n", dataDir)
	err, irc, tg, dbURI := config.ReadConfig(configFile)
	if err != nil {
		log.Fatalf("Unable to parse the config: %v\n", err)
	}

	r := relay.NewRelay()

	if dbURI != "" {
		// if failed to initialize the database, don't use it
		if !irchuubase.Init(dbURI) {
			dbURI = ""
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go irchuu.Launch(irc, &wg, r, dbURI)
	go telegram.Launch(tg, &wg, r, dbURI)
	wg.Wait()
}
