package hq

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/26000/irchuu/config"
)

// URI of the HQ endpoint.
const URI = "https://26000.github.io/irchuu/version.json"

// Report checks for a new version sending data if enabled.
func Report(irchuu *config.Irchuu, tg *config.Telegram, irc *config.Irc) {
	if !irchuu.CheckUpdates && !irchuu.SendStats {
		return
	}

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
		log.Printf("Failed to decode current layer: %v.\n", err)
		return
	}
	if layer > config.LAYER {
		log.Println("New version available, please check https://github.com/26000/irchuu\n  or use `go install github.com/26000/irchuu/...@latest` to update")
		getChangelog(arr[2])
	} else {
		log.Println("Using the latest version of IRChuu.")
	}

	if irchuu.SendStats {
		sendTelemetry(arr[1], strconv.FormatInt(tg.Group, 10), irc.Channel)
	}
}

func sendTelemetry(uri string, tgGroup string, ircChannel string) {
	if uri == "" {
		return
	}

	tgHash := sha256.Sum256([]byte(tgGroup))
	ircHash := sha256.Sum256([]byte(ircChannel))

	// not nice, but relatively readable...
	resp, err := http.Post(uri, "application/json",
		bytes.NewReader([]byte(fmt.Sprintf(`{ "text": "launched with tg: %v, irc: %v, layer: %v",
				"format": "plain", "displayName": "IRChuu~" }`,
			base64.StdEncoding.EncodeToString(tgHash[:31]),
			base64.StdEncoding.EncodeToString(ircHash[:31]),
			config.LAYER))))
	if err != nil {
		log.Printf("Failed to send usage stats: %v.\n", err)
	} else if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// getChangelog downloads a changelog from the server and prints new entries.
func getChangelog(uri string) {
	resp, err := http.Get(uri)
	if err != nil {
		log.Printf("Failed to download changelog: %v.\n",
			err)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to download changelog: %v.\n",
			err)
		return
	}

	changelogFull := string(body)
	changelogRegexStr := fmt.Sprintf("\n--- VERSION[ 0-9\\.]+\\/ LAYER %v ---\n",
		config.LAYER)
	changelogRegex := regexp.MustCompile(changelogRegexStr)
	changelogParts := changelogRegex.Split(changelogFull, 2)
	if len(changelogParts) == 0 {
		return
	}

	//            --- VERSION 0.11.0 / LAYER 17 --- (same length)
	fmt.Printf("\n=========== CHANGELOG ===========\n%v\n=================================\n\n",
		changelogParts[0])
}
