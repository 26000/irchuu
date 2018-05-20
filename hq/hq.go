package hq

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/26000/irchuu/config"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// URI of the HQ endpoint.
const URI = "https://26000.github.io/irchuu/version.json"

// Report checks for a new version sending data if enabled.
func Report(irchuu *config.Irchuu, tg *config.Telegram, irc *config.Irc) {
	if irchuu.CheckUpdates || irchuu.SendStats {
		resp, err := http.Get(URI)
		if err != nil {
			log.Printf("Failed to connect to HQ entrance (check for updates and/or share stats): %v.\n",
				err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to connect to HQ entrance (check for updates and/or share stats): %v.\n",
				err)
			return
		}

		arr := make([]string, 4)
		err = json.Unmarshal(body, arr)
		if err != nil {
			log.Printf("HQ entrance is crazy! Could not check for updates: %v.\n", err)
		}

		layer, err := strconv.Atoi(arr[0])
		if err != nil {
			log.Printf("failed to decode current layer: %v.\n", err)
			return
		}
		if layer > config.LAYER {
			log.Println("New version available, please check https://github.com/26000/irchuu (or use `go get -u github.com/26000/irchuu`.")
		} else {
			log.Println("Using the latest version of IRChuu.")
		}

		if irchuu.SendStats {
			//var data url.Values
			//data = captureData(tg, irc)
		}
	}
}

// captureData generates data to be sent on server.
func captureData(tg *config.Telegram, irc *config.Irc) url.Values {
	tgHash := sha256.Sum256([]byte(strconv.FormatInt(tg.Group, 10)))
	ircHash := sha256.Sum256([]byte(irc.Channel))
	return url.Values{"tg": {base64.StdEncoding.EncodeToString(tgHash[:31])},
		"irc":   {base64.StdEncoding.EncodeToString(ircHash[:31])},
		"layer": {strconv.Itoa(config.LAYER)}}
}
