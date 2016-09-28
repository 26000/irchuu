package telegram

import (
	"fmt"
	"github.com/26000/irchuu/config"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"sync"
)

// Launch launches the Telegram bot and receives updates in an endless loop.
func Launch(c *config.Telegram, wg *sync.WaitGroup) {
	defer wg.Done()
	bot, err := tgbotapi.NewBotAPI(c.Token)
	if err != nil {
		log.Fatalf("Failed to connect to Telegram: %v\n", err)
	}
	log.Printf("Telegram: authorized on account %s\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Chat.ID != int64(update.Message.From.ID) {
			processChatMessage(bot, c, update.Message)
		} else {
			processPM(bot, c, update.Message)
		}
	}
}

func processChatMessage(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message) {
	if message.Chat.ID != c.Group {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("I'm not configured to work in this group (group id: %d)", message.Chat.ID))
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		bot.LeaveChat(tgbotapi.ChatConfig{ChatID: message.Chat.ID})
	}
}

func processPM(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message) {
	var name string

	if message.From.UserName != "" {
		name = "@" + message.From.UserName
	} else {
		name = fmt.Sprintf("%v %v", message.From.FirstName, message.From.LastName)
	}
	log.Printf("Incoming PM from %v: %v\n", name, message.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, "Hi! I work only in groups. An only group to be exact.")
	bot.Send(msg)
}
