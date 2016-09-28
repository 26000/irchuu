package main

import (
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/irc"
	"github.com/26000/irchuu/paths"
	"github.com/26000/irchuu/telegram"
	"log"
	"os"
	"sync"
)

const VERSION = "0.0.0"

func main() {
	fmt.Printf("IRChuu! v%v (https://github.com/26000/irchuu)\n", VERSION)

	configFile, dataDir := paths.GetPaths()
	err := paths.MakePaths(configFile, dataDir)
	if err != nil {
		os.Exit(1)
	}

	log.Printf("Using configuration file: %v\n", configFile)
	log.Printf("Using data directory: %v\n", dataDir)
	err, irc, tg := config.ReadConfig(configFile)
	if err != nil {
		log.Fatalf("Unable to parse the config: %v\n", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go irchuu.Launch(irc, &wg)
	go telegram.Launch(tg, &wg)
	wg.Wait()
}
