package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/26000/irchuu/config"
	irchuubase "github.com/26000/irchuu/db"
	"github.com/26000/irchuu/hq"
	irchuu "github.com/26000/irchuu/irc"
	"github.com/26000/irchuu/paths"
	"github.com/26000/irchuu/relay"
	mediaserver "github.com/26000/irchuu/server"
	"github.com/26000/irchuu/telegram"
)

func main() {
	fmt.Printf("IRChuu! v%v (https://github.com/26000/irchuu)\n", config.VERSION)

	configFile, dataDir := paths.GetPaths()

	flag.StringVar(&configFile, "config", configFile, "path to the configuration file (will be created if not exists)")
	flag.StringVar(&dataDir, "data", dataDir, "path to the data dir")

	flag.Parse()

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

	if irchuuConf.DBURI != "" {
		irchuubase.Init(irchuuConf.DBURI)
	}

	tg.DataDir = dataDir

	if tg.Storage == "server" {
		go mediaserver.Serve(tg)
	}

	hq.Report(irchuuConf, tg, irc)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go sigNotify(sigCh, r)

	var wg sync.WaitGroup
	wg.Add(2)
	go irchuu.Launch(irc, &wg, r)
	go telegram.Launch(tg, &wg, r)
	wg.Wait()
}

func sigNotify(sigCh chan os.Signal, r *relay.Relay) {
	sig := <-sigCh
	log.Printf("Caught signal: %v, exiting... (press Ctrl + C again to force)\n", sig)
	r.TeleAlwaysCh <- relay.ServiceMessage{"shutdown", []string{}}

	sig = <-sigCh
	os.Exit(1)
}
