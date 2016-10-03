package telegram

import (
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/relay"
	"gopkg.in/telegram-bot-api.v4"
	"html"
	"log"
	"os"
	"sync"
	"time"
)

// Launch launches the Telegram bot and receives updates in an endless loop.
func Launch(c *config.Telegram, wg *sync.WaitGroup, r *relay.Relay) {
	defer wg.Done()
	logger := log.New(os.Stdout, " TG ", log.LstdFlags)
	bot, err := tgbotapi.NewBotAPI(c.Token)
	if err != nil {
		logger.Fatalf("Failed to connect to Telegram: %v\n", err)
	}
	logger.Printf("Authorized on account %s\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	go relayMessagesToTG(r, c, bot)
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Chat.Type != "private" {
			processChatMessage(bot, c, update.Message, logger, r)
		} else {
			processPM(bot, c, update.Message, logger)
		}
	}
}

// processChatMessage processes messages from public groups, sending them to
// IRC and Log channels.
func processChatMessage(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message, logger *log.Logger, r *relay.Relay) {
	if message.Chat.ID != c.Group {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("I'm not configured to work in this group (group id: %d)", message.Chat.ID))
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		bot.LeaveChat(tgbotapi.ChatConfig{ChatID: message.Chat.ID})
		return
	}
	if c.TTL == 0 || c.TTL > (time.Now().Unix()-int64(message.Date)) {
		r.TeleCh <- formatMessage(message)
	}
}

// processPM replies to private messages from Telegram, sending them info
// about the bot.
func processPM(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message, logger *log.Logger) {
	var name string

	if message.From.UserName != "" {
		name = "@" + message.From.UserName
	} else {
		name = fmt.Sprintf("%v %v", message.From.FirstName, message.From.LastName)
	}
	logger.Printf("Incoming PM from %v: %v\n", name, message.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, "I only work in my group.\nIf you want to know more about me, visit my [GitHub](https://github.com/26000/irchuu).")
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// relayMessagesToTG listens to the channel and sends messages from IRC to
// Telegram.
func relayMessagesToTG(r *relay.Relay, c *config.Telegram, bot *tgbotapi.BotAPI) {
	for message := range r.IRCh {
		m := formatTGMessage(message, c)
		bot.Send(m)
	}
}

// formatTGMessage translates a universal message into Telegram's one.
func formatTGMessage(message relay.Message, c *config.Telegram) tgbotapi.MessageConfig {
	message.Text = html.EscapeString(message.Text)
	var m tgbotapi.MessageConfig
	if message.Extra["TOPIC"] != "" {
		m = tgbotapi.NewMessage(c.Group,
			fmt.Sprintf("<b>%v</b> has set a new topic: %v",
				message.Nick, message.Text))
	} else if message.Extra["KICK"] != "" {
		m = tgbotapi.NewMessage(c.Group,
			fmt.Sprintf("<b>%v</b> has kicked <b>%v</b>",
				message.Nick, message.Text))
	} else if message.Extra["CTCP"] == "ACTION" {
		m = tgbotapi.NewMessage(c.Group, fmt.Sprintf("*<b>%v</b> %v*",
			message.Nick, message.Text))
	} else {
		m = tgbotapi.NewMessage(c.Group, fmt.Sprintf("<b>%v</b>: %v",
			message.Nick, message.Text))
	}
	m.ParseMode = "HTML"
	return m
}

// formatMessage maps the message onto the universal message struct
// (relay.Message).
func formatMessage(m *tgbotapi.Message) relay.Message {
	return relay.Message{
		Nick: m.From.UserName,
		Text: m.Text,
	}
}
