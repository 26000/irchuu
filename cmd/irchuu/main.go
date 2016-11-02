package main

import (
	"database/sql"
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/db"
	"github.com/26000/irchuu/hq"
	"github.com/26000/irchuu/irc"
	"github.com/26000/irchuu/paths"
	"github.com/26000/irchuu/relay"
	"github.com/26000/irchuu/server"
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
	err, irc, tg, irchuuConf := config.ReadConfig(configFile)
	if err != nil {
		log.Fatalf("Unable to parse the config: %v\n", err)
	}

	r := relay.NewRelay()

	var db *sql.DB

	if irchuuConf.DBURI != "" {
		db = irchuubase.Init(irchuuConf.DBURI)
	}

	tg.DataDir = dataDir

	if tg.ServeMedia {
		go mediaserver.Serve(tg)
	}

	hq.Report(irchuuConf, tg, irc)

	var wg sync.WaitGroup
	wg.Add(2)
	go irchuu.Launch(irc, &wg, r, db)
	go telegram.Launch(tg, &wg, r, db)
	wg.Wait()
}
