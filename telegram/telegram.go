package telegram

import (
	"fmt"
	"github.com/26000/irchuu/config"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"os"
	"sync"
)

// Launch launches the Telegram bot and receives updates in an endless loop.
func Launch(c *config.Telegram, wg *sync.WaitGroup) {
	defer wg.Done()
	logger := log.New(os.Stdout, " TG ", log.LstdFlags)
	bot, err := tgbotapi.NewBotAPI(c.Token)
	if err != nil {
		logger.Fatalf("Failed to connect to Telegram: %v\n", err)
	}
	logger.Printf("Authorized on account %s\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Chat.ID != int64(update.Message.From.ID) {
			processChatMessage(bot, c, update.Message, logger)
		} else {
			processPM(bot, c, update.Message, logger)
		}
	}
}

func processChatMessage(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message, logger *log.Logger) {
	if message.Chat.ID != c.Group {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("I'm not configured to work in this group (group id: %d)", message.Chat.ID))
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		bot.LeaveChat(tgbotapi.ChatConfig{ChatID: message.Chat.ID})
	}
}

func processPM(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message, logger *log.Logger) {
	var name string

	if message.From.UserName != "" {
		name = "@" + message.From.UserName
	} else {
		name = fmt.Sprintf("%v %v", message.From.FirstName, message.From.LastName)
	}
	logger.Printf("Incoming PM from %v: %v\n", name, message.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, "Hi! I work only in groups. An only group to be exact.")
	bot.Send(msg)
}
