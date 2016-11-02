package hq

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/26000/irchuu/config"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

const URI = "https://kotobank.ch/irchuu/z"

// Checks for a new version sending data if enabled.
func Report(irchuu *config.Irchuu, tg *config.Telegram, irc *config.Irc) {
	if irchuu.CheckUpdates || irchuu.SendStats {
		var data url.Values
		if irchuu.SendStats {
			data = captureData(tg, irc)
		}
		resp, err := http.PostForm(URI, data)
		if err != nil {
			log.Printf("Failed to connect to HQ (check for updates and/or share stats): %v.\n",
				err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to connect to HQ (check for updates and/or share stats): %v.\n",
				err)
			return
		}
		if !irchuu.CheckUpdates {
			return
		}
		layer, err := strconv.Atoi(string(body))
		if err != nil {
			log.Printf("Server is crazy, can't check for updates: %v.\n")
			return
		}
		if layer > config.LAYER {
			log.Println("New version available, please check https://github.com/26000/irchuu.")
		} else {
			log.Println("Using the latest version of IRChuu.")
		}
	}
}

// Generates data to be sent on server.
func captureData(tg *config.Telegram, irc *config.Irc) url.Values {
	tgHash := sha256.Sum256([]byte(strconv.FormatInt(tg.Group, 10)))
	ircHash := sha256.Sum256([]byte(irc.Channel))
	return url.Values{"tg": {base64.StdEncoding.EncodeToString(tgHash[:31])},
		"irc":   {base64.StdEncoding.EncodeToString(ircHash[:31])},
		"layer": {strconv.Itoa(config.LAYER)}}
}
