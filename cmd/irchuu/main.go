package main

import (
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/paths"
	//"github.com/26000/irchuu/telegram"
	"fmt"
	"log"
	"os"
	//"sync"
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
	println(irc.Channel)
	println(tg.Token)

	/*
		var wg sync.WaitGroup
		wg.Add(1)
		go telegram.Launch("c", &wg)
		wg.Wait()
	*/
}
