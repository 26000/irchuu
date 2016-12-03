package imgur

import (
	"github.com/26000/irchuu/config"
	"net/http"
	"gopkg.in/telegram-bot-api.v4"
	"bytes"
	"encoding/json"
)

func upload(bot *tgbotapi.BotAPI, id string, c *config.Imgur) (url string, err error) {
	file, err := bot.GetFileDirectURL(id)
	if err != nil {
		return
	}
	//Succesfully copied from StackOverFlow by fufik :^)
	baseurl := "https://api.imgur.com/3/upload"
	jsonStr := []byte(`{"image":"`+ file +`"}`)
	req, err := http.NewRequest("POST", baseurl, bytes.NewBuffer(jsonStr))

	req.Header.Set("Authorization", "Content-ID " + c.ClientID)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var dat map[string]interface{}
	if err := json.Unmarshal(resp.Body, &dat); err != nil {
		panic(err)
	}
	url = dat["data"]["link"].(string)
	return
}
