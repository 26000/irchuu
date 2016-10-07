package telegram

import (
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/relay"
	"gopkg.in/telegram-bot-api.v4"
	"html"
	"log"
	"os"
	"regexp"
	"strings"
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
	go listenService(r, c, bot)
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil && update.EditedMessage != nil {
			update.Message = update.EditedMessage
			update.EditedMessage = nil
		} else if update.Message == nil {
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
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("I'm not configured to work in this group (group id: %d).",
				message.Chat.ID))
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		bot.LeaveChat(tgbotapi.ChatConfig{ChatID: message.Chat.ID})
		logger.Printf("Was added to %v #%v (%v)\n", message.Chat.Type,
			message.Chat.ID, message.Chat.Title)
		return
	}
	if c.TTL == 0 || c.TTL > (time.Now().Unix()-int64(message.Date)) {
		f := formatMessage(message, bot.Self.ID, c.Prefix)
		r.TeleCh <- f
		r.LogCh <- f
		if cmd := message.Command(); cmd != "" {
			processCmd(bot, c, message, cmd, r)
		}
	}
}

// listenService listens to service messages and executes them.
func listenService(r *relay.Relay, c *config.Telegram, bot *tgbotapi.BotAPI) {
	for f := range r.IRCServiceCh {
		switch f.Command {
		case "announce":
			m := tgbotapi.NewMessage(c.Group, f.Arguments)
			bot.Send(m)
		}
	}
}

// processCmd works with commands starting with '/'.
func processCmd(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message, cmd string, r *relay.Relay) {
	arg := message.CommandArguments()
	switch cmd {
	case "kick":
		if c.Moderation {
			user, err := bot.GetChatMember(
				tgbotapi.ChatConfigWithUser{ChatID: c.Group,
					UserID: message.From.ID})
			if err == nil {
				switch user.Status {
				case "administrator":
					fallthrough
				case "creator":
					f := relay.ServiceMessage{"kick", arg}
					r.TeleServiceCh <- f
				case "member":
					m := tgbotapi.NewMessage(c.Group,
						"Insufficient permission.")
					bot.Send(m)
				case "left":
					fallthrough
				case "kicked":
					m := tgbotapi.NewMessage(c.Group,
						">/kick "+arg+"\n\nOh you.")
					bot.Send(m)
				}
			}
		}
	case "ops":
		f := relay.ServiceMessage{"ops", arg}
		r.TeleServiceCh <- f
	case "bot":
		if c.AllowBots {
			f := relay.ServiceMessage{"bot", arg}
			r.TeleServiceCh <- f
		}
	case "invite":
		if c.AllowInvites {
			f := relay.ServiceMessage{"invite", arg}
			r.TeleServiceCh <- f
		}
	case "topic":
		f := relay.ServiceMessage{"topic", arg}
		r.TeleServiceCh <- f
	case "version":
		m := tgbotapi.NewMessage(c.Group, "IRChuu v"+config.VERSION)
		bot.Send(m)
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
	msg := tgbotapi.NewMessage(message.Chat.ID,
		"I only work in my group.\nIf you want to know more about me, visit my [GitHub](https://github.com/26000/irchuu).")
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
	message.Text = reconstructMarkup(message.Text)
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
		m = tgbotapi.NewMessage(c.Group, fmt.Sprintf("%s<b>%v</b>%s %v",
			c.Prefix, message.Nick, c.Postfix, message.Text))
	}
	m.ParseMode = "HTML"
	return m
}

// formatMessage maps the message onto the universal message struct
// (relay.Message).
func formatMessage(message *tgbotapi.Message, id int, prefix string) relay.Message {
	extra := make(map[string]string)
	if message.Text == "" {
		message.Text = message.Caption
	} else {
		message.Text = translateMarkup(*message)
	}

	if message.ReplyToMessage != nil && message.ReplyToMessage.From.ID == id && message.ReplyToMessage.Entities != nil && len(*message.ReplyToMessage.Entities) > 0 && strings.HasPrefix(message.ReplyToMessage.Text, html.UnescapeString(prefix)) {
		extra["reply"] = getEntity(message.ReplyToMessage.Text,
			(*message.ReplyToMessage.Entities)[0])
		extra["replyID"] = string(message.ReplyToMessage.MessageID)
	} else if message.ReplyToMessage != nil {
		extra["reply"] = message.ReplyToMessage.From.String()
		extra["replyID"] = string(message.ReplyToMessage.MessageID)
		extra["replyUserID"] = string(message.ReplyToMessage.From.ID)
	}

	if message.ForwardFrom != nil {
		extra["forward"] = message.ForwardFrom.String()
		extra["forwardUserID"] = string(message.ForwardFrom.ID)
		extra["forwardDate"] = string(message.ForwardDate)
	}

	if message.EditDate != 0 {
		extra["edit"] = string(message.EditDate)
	}

	name := message.From.FirstName
	if message.From.LastName != "" {
		name = name + " " + message.From.LastName
	}

	return relay.Message{
		Date:   int64(message.Date),
		Nick:   message.From.UserName,
		Source: "TG",
		Text:   message.Text,

		ID:     message.MessageID,
		Name:   name,
		FromID: message.From.ID,
		Extra:  extra,
	}
}

// getEntity returns the text of an entity.
func getEntity(text string, ent tgbotapi.MessageEntity) string {
	return string([]rune(text)[ent.Offset : ent.Offset+ent.Length])
}

// getFullName returns the First name or First name [space] Last name.
func getFullName(user *tgbotapi.User) string {
	name := user.FirstName
	if user.LastName != "" {
		name = name + " " + user.LastName
	}
	return name
}

// translateMarkup turns Telegram's entities into IRC's codes.
func translateMarkup(message tgbotapi.Message) string {
	messageText := []rune(message.Text)
	if message.Entities != nil {
		off := 0
		for i := 0; i < len(*message.Entities); i++ {
			e := (*message.Entities)[i]
			e.Offset += off
			switch e.Type {
			case "italic":
				messageText = surroundRunes(messageText,
					e.Offset, e.Length, rune('\x1d'),
					rune('\x0f'))
				off += 2
			case "bold":
				messageText = surroundRunes(messageText,
					e.Offset, e.Length, rune('\x02'),
					rune('\x0f'))
				off += 2
			case "text_link":
				var newMessageText []rune
				newMessageText = append(newMessageText, messageText[:e.Offset]...)
				newMessageText = append(newMessageText, []rune(e.URL)...)
				newMessageText = append(newMessageText, []rune(" (")...)
				newMessageText = append(newMessageText, messageText[e.Offset:e.Offset+e.Length]...)
				newMessageText = append(newMessageText, []rune(") ")...)
				newMessageText = append(newMessageText, messageText[e.Offset+e.Length:]...)
				off += 4 + len(e.URL)
				messageText = newMessageText
			}
		}
	}
	return string(messageText)
}

// reconstructMarkup translates IRC markup to HTML.
func reconstructMarkup(text string) string {

	// strip the colors (or they'll be shown as numbers)
	regex, _ := regexp.Compile("\x03(?:\\d{1,2}(?:,\\d{1,2})?)?")
	text = regex.ReplaceAllLiteralString(text, "")

	newText := ""
	// 0 is plain, 1 is bold (\x02), 2 is italic (\x1d), 3 is color (\x03)
	// if telegram starts supporting it, underline (\x1f) will be added
	// IRC also has reverse (\x16) and color (\x03), which we don't need
	// plain (\x0f) stops all the previous modifiers
	state := 0
	for _, v := range text {
		switch state {
		case 0:
			switch v {
			case '\x02':
				newText += "<b>"
				state = 1
			case '\x1d':
				newText += "<i>"
				state = 2
			default:
				newText += string(v)
			}
		case 1:
			if v == '\x0f' {
				newText += "</b>"
				state = 0
			} else {
				newText += string(v)
			}
		case 2:
			if v == '\x0f' {
				newText += "</i>"
				state = 0
			} else {
				newText += string(v)
			}
		}
	}

	// if the tags were not closed, HTML will be invalid (but it's ok for IRC)
	switch state {
	case 1:
		newText += "</b>"
	case 2:
		newText += "</i>"
	}

	return newText
}

// surroundRunes adds two runes before a substring and after that substring.
func surroundRunes(runes []rune, offset int, length int, rune1 rune, rune2 rune) []rune {
	var newRunes []rune
	newRunes = append(newRunes, runes[:offset]...)
	newRunes = append(newRunes, rune1)
	newRunes = append(newRunes, runes[offset:offset+length]...)
	newRunes = append(newRunes, rune2)
	newRunes = append(newRunes, runes[offset+length:]...)
	return newRunes
}
