package main

import (
	"../../paths/"
	"fmt"
	"github.com/zpatrick/go-config"
	"log"
)

const VERSION = "0.0.0"

func main() {
	fmt.Printf("IRChuu! v%v (https://github.com/26000/irchuu)\n", VERSION)

	configFile, dataDir := paths.GetPaths()
	log.Printf("Using configuration file: %v\n", configFile)
	log.Printf("Using data directory: %v\n", dataDir)
	iniFile := config.NewINIFile(configFile)
	c := config.NewConfig([]config.Provider{iniFile})
	if err := c.Load(); err != nil {
		log.Fatalf("Could not load the configuration: %v\n", err)
	}
}
