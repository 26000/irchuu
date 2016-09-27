package main

import (
	"fmt"
	//"github.com/zpatrick/go-config"
	"../../config"
	"log"
)

const VERSION = "0.0.0"

func main() {
	fmt.Printf("IRChuu! v%v (https://github.com/26000/irchuu)\n", VERSION)

	configFile, dataDir := config.GetPaths()
	log.Printf("Using configuration file: %v\n", configFile)
	log.Printf("Using data directory: %v\n", dataDir)
}
