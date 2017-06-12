// telegram contains everything related to the Telegram part of IRChuu.
package telegram

import (
	"database/sql"
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/db"
	"github.com/26000/irchuu/paths"
	"github.com/26000/irchuu/relay"
	"github.com/26000/irchuu/upload"
	"gopkg.in/telegram-bot-api.v4"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

// Launch launches the Telegram bot and receives updates in an endless loop.
func Launch(c *config.Telegram, wg *sync.WaitGroup, r *relay.Relay, db *sql.DB) {
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
			processChatMessage(bot, c, update.Message, logger, r, db)
		} else {
			processPM(bot, c, update.Message, logger)
		}
	}
}

// processChatMessage processes messages from public groups, sending them to
// IRC and Log channels.
func processChatMessage(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message, logger *log.Logger, r *relay.Relay, db *sql.DB) {
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
		if f.Extra["mediaID"] != "" {
			switch {
			case c.Storage == "pomf":
				url, err := upload.Pomf(bot, f.Extra["mediaID"], c)
				if err != nil {
					logger.Printf("Could not upload media %v: %v\n",
						f.Extra["mediaID"], err)
				} else {
					f.Extra["url"] = url
				}
			case c.Storage == "komf":
				url, err := upload.Komf(bot, f.Extra["mediaID"], c)
				if err != nil {
					logger.Printf("Could not upload media %v: %v\n",
						f.Extra["mediaID"], err)
				} else {
					f.Extra["url"] = url
				}
			case c.DownloadMedia:
				url, err := download(bot, f.Extra["mediaID"], c)
				if err != nil {
					logger.Printf("Could not download media %v: %v\n",
						f.Extra["mediaID"], err)
				} else if c.Storage == "server" {
					f.Extra["url"] = url
				}
			}
		}
		r.TeleCh <- f
		if db != nil {
			go irchuubase.Log(f, db, logger)
		}
		if cmd := message.Command(); cmd != "" {
			processCmd(bot, c, message, cmd, r)
		}
	}
}

// listenService listens to service messages and executes them.
// TODO: restructure
func listenService(r *relay.Relay, c *config.Telegram, bot *tgbotapi.BotAPI) {
	for f := range r.IRCServiceCh {
		switch f.Command {
		case "announce":
			m := tgbotapi.NewMessage(c.Group, f.Arguments[0])
			bot.Send(m)
		case "count":
			count, err := bot.GetChatMembersCount(
				tgbotapi.ChatConfig{ChatID: c.Group})
			if err != nil {
				r.TeleServiceCh <- relay.ServiceMessage{
					"announce",
					[]string{"An error occured: \x02" + err.Error()},
				}
			} else {
				r.TeleServiceCh <- relay.ServiceMessage{
					"announce",
					[]string{fmt.Sprintf("There are \x02%v"+
						"\x0f users in the group.",
						count)},
				}
			}
		case "ops":
			ops, err := bot.GetChatAdministrators(
				tgbotapi.ChatConfig{ChatID: c.Group})
			if err != nil {
				r.TeleServiceCh <- relay.ServiceMessage{
					"announce",
					[]string{"An error occured: \x02" +
						err.Error()},
				}
			} else {
				opsStr := ""
				for _, v := range ops {
					opsStr += v.User.String() + " "
				}
				r.TeleServiceCh <- relay.ServiceMessage{
					"announce",
					[]string{fmt.Sprintf(
						"Chat administrators: \x02%v"+
							"\x0f",
						opsStr)},
				}
			}
		case "sticker":
			sticker := tgbotapi.NewStickerShare(c.Group, f.Arguments[0])
			_, err := bot.Send(sticker)
			if err != nil {
				r.TeleServiceCh <- relay.ServiceMessage{
					"announce",
					[]string{"An error occured: \x02" +
						err.Error()},
				}
			} else {
				text := "Sent a sticker"
				switch {
				case c.Storage == "pomf":
					url, err := upload.Pomf(bot, f.Arguments[0], c)
					if err == nil {
						text += " ( " + url + " )"
					}
				case c.Storage == "komf":
					url, err := upload.Komf(bot, f.Arguments[0], c)
					if err == nil {
						text += " ( " + url + " )"
					}
				case c.DownloadMedia:
					url, err := download(bot, f.Arguments[0], c)
					if err == nil && c.Storage == "server" {
						text += " ( " + url + " )"
					}
				}
				r.TeleServiceCh <- relay.ServiceMessage{
					"announce",
					[]string{text},
				}
			}
		case "kick":
			id, _ := strconv.Atoi(f.Arguments[0])
			member := tgbotapi.ChatMemberConfig{
				ChatID: c.Group,
				UserID: id,
			}
			_, err := bot.KickChatMember(member)
			if err != nil {
				r.TeleServiceCh <- relay.ServiceMessage{
					"announce",
					[]string{"Unable to kick: " +
						err.Error() + "."},
				}
			} else {
				r.TeleServiceCh <- relay.ServiceMessage{
					"action",
					[]string{"kicked " + f.Arguments[1] + "."},
				}
			}
		case "unban":
			id, _ := strconv.Atoi(f.Arguments[0])
			member := tgbotapi.ChatMemberConfig{
				ChatID: c.Group,
				UserID: id,
			}
			_, err := bot.UnbanChatMember(member)
			if err != nil {
				r.TeleServiceCh <- relay.ServiceMessage{
					"announce",
					[]string{"Unable to unban: " +
						err.Error() + "."},
				}
			} else {
				r.TeleServiceCh <- relay.ServiceMessage{
					"action",
					[]string{"unbanned " + f.Arguments[1] + "."},
				}
			}
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
					if arg != "" {
						f := relay.ServiceMessage{"kick",
							[]string{arg,
								message.From.String()}}
						r.TeleServiceCh <- f
					}
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
		f := relay.ServiceMessage{"ops", []string{arg}}
		r.TeleServiceCh <- f
	case "bot":
		if c.AllowBots {
			f := relay.ServiceMessage{"bot", []string{arg}}
			r.TeleServiceCh <- f
		}
	case "invite":
		if c.AllowInvites {
			f := relay.ServiceMessage{"invite", []string{arg}}
			r.TeleServiceCh <- f
		}
	case "topic":
		f := relay.ServiceMessage{"topic", nil}
		r.TeleServiceCh <- f
	case "version":
		m := tgbotapi.NewMessage(c.Group, "IRChuu v"+config.VERSION)
		bot.Send(m)
	case "help":
		text := `Available commands:

/help — show this help
/version — show version info
/topic — get IRC channel topic
/ops — view OPs list`
		if c.AllowInvites {
			text += "\n/invite [nick] — invite a user to the IRC channel"
		}
		if c.Moderation {
			text += "\n/kick — kick a user from IRC"
		}
		if c.AllowBots {
			text += "\n/bot [message] — send messages to IRC bots (no nickname prefix)"
		}
		m := tgbotapi.NewMessage(c.Group, text)
		bot.Send(m)
	}
}

// processPM replies to private messages from Telegram, sending them info
// about the bot.
func processPM(bot *tgbotapi.BotAPI, c *config.Telegram, message *tgbotapi.Message, logger *log.Logger) {
	logger.Printf("Incoming PM from %v: %v\n", message.From.String(),
		message.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID,
		"I only work in my group.\nIf you want to know more about me, "+
			"visit my [GitHub](https://github.com/26000/irchuu).")
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
	switch message.Extra["special"] {
	case "TOPIC":
		m = tgbotapi.NewMessage(c.Group,
			fmt.Sprintf("<b>%v</b> has set a new topic: <b>%v</b>.",
				message.Nick, message.Text))
	case "KICK":
		m = tgbotapi.NewMessage(c.Group,
			fmt.Sprintf("<b>%v</b> has kicked <b>%v</b>.",
				message.Nick, message.Text))
	case "NICK":
		m = tgbotapi.NewMessage(c.Group,
			fmt.Sprintf("<b>%v</b> is now known as <b>%v</b>.",
				message.Nick, message.Text))
	case "ACTION":
		m = tgbotapi.NewMessage(c.Group, fmt.Sprintf("*<b>%v</b> %v*",
			message.Nick, message.Text))
	case "MODE":
		m = tgbotapi.NewMessage(c.Group, fmt.Sprintf("<b>%v</b> has set mode <b>%v</b>.",
			message.Nick, message.Text))
	case "PART":
		var text string
		if message.Text == "" {
			text = fmt.Sprintf("<b>%v</b> has left.",
				message.Nick)
		} else {
			text = fmt.Sprintf("<b>%v</b> has left: <b>%v</b>.",
				message.Nick, message.Text)
		}
		m = tgbotapi.NewMessage(c.Group, text)
	case "QUIT":
		var text string
		if message.Text == "" {
			text = fmt.Sprintf("<b>%v</b> has quit.",
				message.Nick)
		} else {
			text = fmt.Sprintf("<b>%v</b> has quit: <b>%v</b>.",
				message.Nick, message.Text)
		}
		m = tgbotapi.NewMessage(c.Group, text)
	case "JOIN":
		m = tgbotapi.NewMessage(c.Group,
			fmt.Sprintf("<b>%v</b> has joined.",
				message.Nick))
	case "NOTICE":
		m = tgbotapi.NewMessage(c.Group, fmt.Sprintf("(notice) %s<b>%v</b>%s %v",
			c.Prefix, message.Nick, c.Postfix, message.Text))
	default:
		m = tgbotapi.NewMessage(c.Group, fmt.Sprintf("%s<b>%v</b>%s %v",
			c.Prefix, message.Nick, c.Postfix, message.Text))
	}
	m.ParseMode = "HTML"
	return m
}

// formatMessage maps the message onto the universal message struct
// (relay.Message).
// TODO: split into several funcs?
func formatMessage(message *tgbotapi.Message, id int, prefix string) relay.Message {
	extra := make(map[string]string)

	if message.PinnedMessage != nil {
		// we save all pin info...
		extra["pinDate"] = strconv.Itoa(message.Date)
		extra["pinUserID"] = strconv.Itoa(message.From.ID)
		extra["pin"] = message.From.String()
		// ...and turn our message into the pinned message
		// it will get processed as any other message, we know that it's
		// a pin
		id := message.MessageID
		message = message.PinnedMessage
		extra["pinID"] = strconv.Itoa(message.MessageID)
		// and anyway, message IDs must be unique
		message.MessageID = id
		extra["special"] = "pin"
	}

	if message.Text == "" {
		message.Text = message.Caption
	} else {
		message.Text = translateMarkup(*message)
	}

	if message.ReplyToMessage != nil && message.ReplyToMessage.From.ID == id && message.ReplyToMessage.Entities != nil && len(*message.ReplyToMessage.Entities) > 0 && strings.HasPrefix(message.ReplyToMessage.Text, html.UnescapeString(prefix)) {
		extra["reply"] = getEntity(message.ReplyToMessage.Text,
			(*message.ReplyToMessage.Entities)[0])
		extra["replyID"] = strconv.Itoa(message.ReplyToMessage.MessageID)
	} else if message.ReplyToMessage != nil {
		extra["reply"] = message.ReplyToMessage.From.String()
		extra["replyID"] = strconv.Itoa(message.ReplyToMessage.MessageID)
		extra["replyUserID"] = strconv.Itoa(message.ReplyToMessage.From.ID)
	}

	if message.ForwardFrom != nil {
		extra["forward"] = message.ForwardFrom.String()
		extra["forwardUserID"] = strconv.Itoa(message.ForwardFrom.ID)
		extra["forwardDate"] = strconv.Itoa(message.ForwardDate)
	} else if message.ForwardFromChat != nil {
		extra["forwardChat"] = message.ForwardFromChat.UserName
		extra["forwardChatTitle"] = message.ForwardFromChat.Title
		extra["forwardChatID"] = strconv.FormatInt(message.ForwardFromChat.ID,
			10)
		extra["forwardDate"] = strconv.Itoa(message.ForwardDate)
	}

	if message.EditDate != 0 {
		extra["edit"] = strconv.Itoa(message.EditDate)
	}

	switch {
	case message.Photo != nil:
		photo := (*message.Photo)[len(*message.Photo)-1]
		extra["media"] = "photo"
		extra["mediaID"] = photo.FileID
		extra["width"] = strconv.Itoa(photo.Width)
		extra["height"] = strconv.Itoa(photo.Height)
		extra["size"] = strconv.Itoa(photo.FileSize)
	case message.Document != nil:
		extra["media"] = "document"
		extra["mediaID"] = message.Document.FileID
		extra["mediaName"] = message.Document.FileName
		extra["mime"] = message.Document.MimeType
		extra["size"] = strconv.Itoa(message.Document.FileSize)
	case message.Sticker != nil:
		extra["media"] = "sticker"
		extra["mediaID"] = message.Sticker.FileID
		message.Text = message.Sticker.Emoji
		extra["width"] = strconv.Itoa(message.Sticker.Width)
		extra["height"] = strconv.Itoa(message.Sticker.Height)
		extra["size"] = strconv.Itoa(message.Sticker.FileSize)
	case message.Audio != nil:
		extra["media"] = "audio"
		extra["mediaID"] = message.Audio.FileID
		extra["mediaName"] = message.Audio.Title
		extra["performer"] = message.Audio.Performer
		extra["duration"] = strconv.Itoa(message.Audio.Duration)
		extra["mime"] = message.Audio.MimeType
		extra["size"] = strconv.Itoa(message.Audio.FileSize)
	case message.Video != nil:
		extra["media"] = "video"
		extra["mediaID"] = message.Video.FileID
		extra["duration"] = strconv.Itoa(message.Video.Duration)
		extra["mime"] = message.Video.MimeType
		extra["width"] = strconv.Itoa(message.Video.Width)
		extra["height"] = strconv.Itoa(message.Video.Height)
		extra["size"] = strconv.Itoa(message.Video.FileSize)
	case message.Voice != nil:
		extra["media"] = "voice"
		extra["mediaID"] = message.Voice.FileID
		extra["duration"] = strconv.Itoa(message.Voice.Duration)
		extra["mime"] = message.Voice.MimeType
		extra["size"] = strconv.Itoa(message.Voice.FileSize)
	case message.Contact != nil:
		// TODO: do something with it
		extra["contactID"] = strconv.Itoa(message.Contact.UserID)
		message.Text += fmt.Sprintf("contact: %v %v (%v)",
			message.Contact.FirstName,
			message.Contact.LastName,
			message.Contact.PhoneNumber)
	case message.Location != nil:
		// TODO: same as above
		extra["lat"] = strconv.FormatFloat(message.Location.Latitude, 'f',
			-1, 64)
		extra["lng"] = strconv.FormatFloat(message.Location.Longitude, 'f',
			-1, 64)
		message.Text += fmt.Sprintf("location: https://www.google.com/maps/@%v,%v,14z",
			extra["lat"], extra["lng"])
	case message.Venue != nil:
		// TODO: same as above
		extra["lat"] = strconv.FormatFloat(message.Venue.Location.Latitude, 'f',
			-1, 64)
		extra["lng"] = strconv.FormatFloat(message.Venue.Location.Longitude, 'f',
			-1, 64)
		extra["venue"] = message.Venue.Title
		extra["address"] = message.Venue.Address
		extra["foursquare"] = message.Venue.FoursquareID
		message.Text += fmt.Sprintf("location \"%v\": https://www.google.com/maps/@%v,%v,14z",
			extra["venue"], extra["lat"], extra["lng"])
	case message.Game != nil:
		extra["game"] = message.Game.Title
		message.Text = "[game]" + extra["game"] + "\n" + message.Game.Text
	case message.NewChatMember != nil:
		extra["special"] = "newChatMember"
		extra["memberID"] = strconv.Itoa(message.NewChatMember.ID)
		extra["memberName"] = message.NewChatMember.String()
	case message.LeftChatMember != nil:
		extra["special"] = "leftChatMember"
		extra["memberID"] = strconv.Itoa(message.LeftChatMember.ID)
		extra["memberName"] = message.LeftChatMember.String()
	case message.NewChatTitle != "":
		extra["special"] = "newChatTitle"
		extra["title"] = message.NewChatTitle
	case message.NewChatPhoto != nil:
		extra["special"] = "newChatPhoto"
	case message.DeleteChatPhoto != false:
		extra["special"] = "deleteChatPhoto"
	}

	return relay.Message{
		Date:   message.Time(),
		Source: true,
		Nick:   message.From.UserName,
		Text:   message.Text,

		ID:        message.MessageID,
		FromID:    message.From.ID,
		FirstName: message.From.FirstName,
		LastName:  message.From.LastName,
		Extra:     extra,
	}
}

// download gets the media link from Telegram and downloads its contents.
func download(bot *tgbotapi.BotAPI, id string, c *config.Telegram) (url string, err error) {
	file, err := bot.GetFileDirectURL(id)
	if err != nil {
		return
	}
	fileStrings := strings.Split(file, "/")
	fileName := strings.Split(fileStrings[len(fileStrings)-1], ".")

	var ext string
	if len(fileName) > 1 {
		ext = "." + fileName[len(fileName)-1]
	}
	localUrl := path.Join(c.DataDir, id+ext)
	if paths.Exists(localUrl) {
		url = c.BaseURL + "/" + id + ext
		return
	}
	downloadable, err := http.Get(file)
	defer downloadable.Body.Close()
	if err != nil {
		return
	}
	res, err := os.Create(localUrl)
	if err != nil {
		return
	}
	defer res.Close()
	io.Copy(res, downloadable.Body)
	url = c.BaseURL + "/" + id + ext
	return
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
	messageText := utf16.Encode([]rune(message.Text))
	if message.Entities != nil {
		off := 0
		for i := 0; i < len(*message.Entities); i++ {
			e := (*message.Entities)[i]
			e.Offset += off
			switch e.Type {
			case "italic":
				// \x1d, \x0f
				messageText = surroundCodePoints(messageText,
					e.Offset, e.Length, []uint16{29},
					[]uint16{15})
				off += 2
			case "bold":
				// \x02, \x0f
				messageText = surroundCodePoints(messageText,
					e.Offset, e.Length, []uint16{2},
					[]uint16{15})
				off += 2
			case "text_link":
				var newMessageText []uint16
				url := utf16.Encode([]rune(e.URL))
				newMessageText = append(newMessageText,
					messageText[:e.Offset+e.Length]...)
				newMessageText = append(newMessageText,
					[]uint16{32, 40}...) // " ("
				newMessageText = append(newMessageText, url...)
				newMessageText = append(newMessageText,
					[]uint16{41, 32}...) // ") "
				newMessageText = append(newMessageText,
					messageText[e.Offset+e.Length:]...)
				off += 4 + len(url)
				messageText = newMessageText
			}
		}
	}
	return string(utf16.Decode(messageText))
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

// surroundCodePoints surrounds part of a uint16 slice with two other slices.
func surroundCodePoints(slice []uint16, offset int, length int, slice1 []uint16, slice2 []uint16) []uint16 {
	var new []uint16
	new = append(new, slice[:offset]...)
	new = append(new, slice1...)
	new = append(new, slice[offset:offset+length]...)
	new = append(new, slice2...)
	new = append(new, slice[offset+length:]...)
	return new
}
