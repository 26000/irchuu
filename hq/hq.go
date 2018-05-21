package hq

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/26000/irchuu/config"
	"io/ioutil"
	"log"
	"net/http"
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
		err = json.Unmarshal(body, &arr)
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
			tgHash := sha256.Sum256([]byte(strconv.FormatInt(tg.Group, 10)))
			ircHash := sha256.Sum256([]byte(irc.Channel))

			// not nice, but relatively readable...
			resp, err := http.Post(arr[1], "application/json",
				bytes.NewReader([]byte(fmt.Sprintf(`{ "text": "launched with tg: %v, irc: %v, layer: %v",
				"format": "plain", "displayName": "IRChuu~" }`,
					base64.StdEncoding.EncodeToString(tgHash[:31]),
					base64.StdEncoding.EncodeToString(ircHash[:31]),
					config.LAYER))))
			if err != nil {
				log.Printf("Failed to send usage stats: %v.\n", err)
			}
			if resp.Body != nil {
				resp.Body.Close()
			}
		}
	}
}
