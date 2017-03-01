// irchuu contains everything related to the IRC part of IRChuu.
package irchuu

import (
	"code.cloudfoundry.org/bytefmt"
	"database/sql"
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/db"
	"github.com/26000/irchuu/relay"
	"github.com/thoj/go-ircevent"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Launch starts the IRC bot and waits for messages.
func Launch(c *config.Irc, wg *sync.WaitGroup, r *relay.Relay, db *sql.DB) {
	defer wg.Done()

	startTime := time.Now()

	logger := log.New(os.Stdout, "IRC ", log.LstdFlags)
	irchuu := irc.IRC(c.Nick, "IRChuu")
	irchuu.UseTLS = c.SSL
	irchuu.Password = c.ServerPassword

	// 0 — not on channel
	// 1 — normal
	// 2 — voice (+v, +)
	// 3 — halfop (+h, %)
	// 4 — op (+o, @)
	// 5 — protected/admin (+a, &)
	// 6 — owner (+q, ~)
	names := make(map[string]int)
	tempNames := make(map[string]int)
	nameQueryStarted := false

	if c.SASL {
		irchuu.UseSASL = true
		irchuu.SASLLogin = c.Nick
		irchuu.SASLPassword = c.Password
	}

	irchuu.Debug = c.Debug
	irchuu.Log = logger
	irchuu.QuitMessage = "IRChuu!bye"
	irchuu.Version = fmt.Sprintf("IRChuu! v%v (https://github.com/26000/irchuu), based on %v", config.VERSION, irc.VERSION)

	/* START CALLBACKS */
	irchuu.AddCallback("CTCP_VERSION", func(event *irc.Event) {
		logger.Printf("CTCP VERSION from %v\n", event.Nick)
	})

	irchuu.AddCallback("CTCP", func(event *irc.Event) {
		if event.Arguments[1] == "UPTIME" {
			irchuu.SendRawf("NOTICE %s :\x01UPTIME %s\x01", event.Nick,
				time.Since(startTime))
			logger.Printf("CTCP %v from %v\n", event.Arguments[1],
				event.Nick)
		} else {
			logger.Printf("Unknown CTCP %v from %v\n", event.Arguments[1],
				event.Nick)
		}
	})

	irchuu.RemoveCallback("CTCP_CLIENTINFO", 0)
	irchuu.AddCallback("CTCP_CLIENTINFO", func(event *irc.Event) {
		irchuu.SendRawf("NOTICE %s :\x01CLIENTINFO PING VERSION TIME UPTIME USERINFO CLIENTINFO\x01",
			event.Nick)
	})

	irchuu.AddCallback("NOTICE", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			f := formatMessage(event.Nick, event.Message(), "NOTICE")
			r.IRCh <- f
			if db != nil {
				go irchuubase.Log(f, db, logger)
			}
		} else {
			logger.Printf("Notice from %v: %v\n",
				event.Nick, event.Message())
			// No auto-replies for notices (as defined in RFC)
		}
	})

	// SASL Authentication status
	irchuu.AddCallback("903", func(event *irc.Event) {
		logger.Printf("%v\n", event.Arguments[1])
	})

	// Errors
	irchuu.AddCallback("461", func(event *irc.Event) {
		logger.Printf("Error: ERR_NEEDMOREPARAMS\n")
	})

	irchuu.AddCallback("433", func(event *irc.Event) {
		logger.Printf("Nickname already in use, changed to %v\n",
			irchuu.GetNick())
		irchuu.Join(fmt.Sprintf("%v %v", c.Channel, c.ChanPassword))
	})

	irchuu.AddCallback("473", func(event *irc.Event) {
		logger.Printf("The channel is invite-only, please invite me\n")
	})

	irchuu.AddCallback("471", func(event *irc.Event) {
		logger.Printf("The channel is full\n")
	})

	irchuu.AddCallback("403", func(event *irc.Event) {
		logger.Printf("The channel doesn't exist\n")
	})

	irchuu.AddCallback("474", func(event *irc.Event) {
		logger.Printf("%v is banned on the channel\n", irchuu.GetNick())
	})

	irchuu.AddCallback("475", func(event *irc.Event) {
		logger.Printf("The channel password is incorrect\n")
	})

	irchuu.AddCallback("476", func(event *irc.Event) {
		logger.Printf("Error: ERR_BADCHANMASK\n")
	})

	irchuu.AddCallback("405", func(event *irc.Event) {
		logger.Printf("Error: ERR_TOOMANYCHANNELS\n")
	})

	// You are not channel operator
	irchuu.AddCallback("482", func(event *irc.Event) {
		if event.Arguments[1] == c.Channel {
			r.IRCServiceCh <- relay.ServiceMessage{"announce", []string{"I need to be an operator in IRC for that action."}}
		}
	})

	irchuu.AddCallback("INVITE", func(event *irc.Event) {
		logger.Printf("Invited to %v by %v\n", event.Arguments[1], event.Nick)
		if c.Channel == event.Arguments[1] {
			irchuu.Join(fmt.Sprintf("%v %v", c.Channel, c.ChanPassword))
		}
	})

	irchuu.AddCallback("341", func(event *irc.Event) {
		s := relay.ServiceMessage{"announce",
			[]string{fmt.Sprintf("Invited %v to %v.", event.Arguments[1],
				event.Arguments[2])}}

		r.IRCServiceCh <- s
		r.TeleServiceCh <- s
	})

	// TODO: add bold for these messages
	irchuu.AddCallback("443", func(event *irc.Event) {
		s := relay.ServiceMessage{"announce",
			[]string{fmt.Sprintf("User %v is already on channel.",
				event.Arguments[1])}}

		r.IRCServiceCh <- s
	})

	/*
		irchuu.AddCallback("401", func(event *irc.Event) {
			s := relay.ServiceMessage{"announce",
				[]string{fmt.Sprintf("No such nick: %v.", event.Arguments[1])}}

			r.IRCServiceCh <- s
			r.TeleServiceCh <- s
		})
	*/

	// On joined...
	irchuu.AddCallback("JOIN", func(event *irc.Event) {
		if event.Nick == irchuu.GetNick() {
			logger.Printf("Joined %v\n", event.Arguments[0])
			if event.Arguments[0] == c.Channel {
				go relayMessagesToIRC(r, c, irchuu)
				go listenService(r, c, irchuu, &names)
				if !nameQueryStarted {
					go updateNames(irchuu, c)
					nameQueryStarted = true
				}
			}
		} else {
			if event.Arguments[0] == c.Channel {
				f := formatMessage(event.Nick, "", "JOIN")
				if c.RelayJoinsParts {
					r.IRCh <- f
				}
				if db != nil {
					go irchuubase.Log(f, db, logger)
				}
				names[event.Nick] = 1
			}
		}
	})

	if c.AnnounceTopic {
		// Topic
		irchuu.AddCallback("332", func(event *irc.Event) {
			if event.Arguments[1] == c.Channel {
				r.IRCServiceCh <- relay.ServiceMessage{"announce",
					[]string{fmt.Sprintf("The topic for %v is %v.",
						c.Channel, event.Arguments[2])}}
			}
		})

		// No topic
		irchuu.AddCallback("331", func(event *irc.Event) {
			if event.Arguments[1] == c.Channel {
				r.IRCServiceCh <- relay.ServiceMessage{"announce",
					[]string{"No topic is set."}}
			}
		})
	}

	// Names
	irchuu.AddCallback("353", func(event *irc.Event) {
		if event.Arguments[2] == c.Channel {
			for _, name := range strings.Split(event.Arguments[3], " ") {
				if len(name) == 0 {
					continue
				}
				switch name[0] {
				case '+':
					tempNames[name[1:]] = 2
				case '%':
					tempNames[name[1:]] = 3
				case '@':
					tempNames[name[1:]] = 4
				case '&':
					tempNames[name[1:]] = 5
				case '~':
					tempNames[name[1:]] = 6
				default:
					tempNames[name] = 1
				}
			}
		}
	})

	// End of names
	irchuu.AddCallback("366", func(event *irc.Event) {
		if event.Arguments[1] == c.Channel {
			names = tempNames
		}
	})

	irchuu.AddCallback("PRIVMSG", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			f := formatMessage(event.Nick, event.Message(), "")
			r.IRCh <- f
			if db != nil {
				go irchuubase.Log(f, db, logger)
			}
			if strings.HasPrefix(event.Message(), c.Nick) {
				processCmd(event, irchuu, c, r, db, &names)
			}
		} else {
			logger.Printf("Message from %v: %v\n",
				event.Nick, event.Message())
			if names[event.Nick] != 0 {
				processPMCmd(event, irchuu, c, r, db)
			} else {
				noticeOrMsg(irchuu, c.SendNotices, event.Nick,
					"I work only for my channel members."+
						" https://github.com/26000/irchuu"+
						" for more info.")
			}
		}
	})

	irchuu.AddCallback("CTCP_ACTION", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			f := formatMessage(event.Nick, event.Message(), "ACTION")
			r.IRCh <- f
			if db != nil {
				go irchuubase.Log(f, db, logger)
			}
		} else {
			logger.Printf("CTCP ACTION from %v: %v\n",
				event.Nick, event.Message())
		}
	})

	irchuu.AddCallback("KICK", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			f := formatMessage(event.Nick, event.Arguments[1], "KICK")
			r.IRCh <- f // TODO: kick reasons are not saved
			if db != nil {
				go irchuubase.Log(f, db, logger)
			}
			names[event.Arguments[1]] = 0
			if event.Arguments[1] == irchuu.GetNick() {
				stopMsg := relay.Message{Extra: map[string]string{"break": "true"}}
				stopCmd := relay.ServiceMessage{Command: "break"}
				r.TeleCh <- stopMsg
				r.TeleServiceCh <- stopCmd
				if c.KickRejoin {
					irchuu.Join(fmt.Sprintf("%v %v", c.Channel, c.ChanPassword))
				}
			}
		}
	})

	irchuu.AddCallback("NICK", func(event *irc.Event) {
		f := formatMessage(event.Nick, event.Arguments[0], "NICK")
		r.IRCh <- f
		if db != nil {
			go irchuubase.Log(f, db, logger)
		}
		names[event.Arguments[0]] = names[event.Nick]
		names[event.Nick] = 0
	})

	irchuu.AddCallback("PART", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			var reason string
			if len(event.Arguments) > 1 {
				reason = event.Arguments[1]
			}
			f := formatMessage(event.Nick, reason, "PART")
			if c.RelayJoinsParts {
				r.IRCh <- f
			}
			if db != nil {
				go irchuubase.Log(f, db, logger)
			}
			names[event.Nick] = 0
		}
	})

	irchuu.AddCallback("QUIT", func(event *irc.Event) {
		var reason string
		if len(event.Arguments) > 0 {
			reason = event.Arguments[0]
		}
		f := formatMessage(event.Nick, reason, "QUIT")
		if c.RelayJoinsParts {
			r.IRCh <- f
		}
		if db != nil {
			go irchuubase.Log(f, db, logger)
		}
		names[event.Nick] = 0
	})

	irchuu.AddCallback("MODE", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			f := formatMessage(event.Nick, strings.Join(event.Arguments, " "), "MODE")
			if c.RelayModes {
				r.IRCh <- f
			}
			if db != nil {
				go irchuubase.Log(f, db, logger)
			}
			if len(event.Arguments) > 2 {
				for k, o := range parseMode(event) {
					names[k] = o
				}
			}
		}
	})

	irchuu.AddCallback("TOPIC", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			f := formatMessage(event.Nick, event.Arguments[1], "TOPIC")
			r.IRCh <- f
			if db != nil {
				go irchuubase.Log(f, db, logger)
			}
		}
	})

	// On connected...
	irchuu.AddCallback("001", func(event *irc.Event) {
		if !c.SASL && c.Password != "" {
			logger.Println("Trying to authenticate via NickServ")
			irchuu.Privmsgf("NickServ", "IDENTIFY %v", c.Password)
		}

		irchuu.Join(fmt.Sprintf("%v %v", c.Channel, c.ChanPassword))
	})
	/* CALLBACKS END */

	err := irchuu.Connect(fmt.Sprintf("%v:%d", c.Server, c.Port))
	if err != nil {
		logger.Fatalf("Cannot connect: %v\n", err)
	}

	irchuu.Loop()
}

// noticeOrMsg sends NOTICE if the second arg is true or PRIVMSG else.
func noticeOrMsg(irc *irc.Connection, notice bool, target, message string) {
	if notice {
		irc.Notice(target, message)
	} else {
		irc.Privmsg(target, message)
	}
}

// noticeOrMsgf sends formatted NOTICE or PRIVMSG.
func noticeOrMsgf(irc *irc.Connection, notice bool, target, format string, a ...interface{}) {
	if notice {
		irc.Noticef(target, format, a...)
	} else {
		irc.Privmsgf(target, format, a...)
	}
}

// parseMode parses the MODE event and returns users with their ranks
// (map[string]int)
func parseMode(event *irc.Event) map[string]int {
	// 0 — not on channel
	// 1 — normal
	// 2 — voice (+v, +)
	// 3 — halfop (+h, %)
	// 4 — op (+o, @)
	// 5 — protected/admin (+a, &)
	// 6 — owner (+q, ~)
	m := make(map[string]int)
	var mode bool
	n := 1
	for _, l := range event.Arguments[1] {
		switch l {
		case '+':
			mode = true
		case '-':
			mode = false
		case 'v':
			n++
			if mode {
				m[event.Arguments[n]] = 2
			} else {
				m[event.Arguments[n]] = 1
			}
		case 'h':
			n++
			if mode {
				m[event.Arguments[n]] = 3
			} else {
				m[event.Arguments[n]] = 1
			}
		case 'o':
			n++
			if mode {
				m[event.Arguments[n]] = 4
			} else {
				m[event.Arguments[n]] = 1
			}
		case 'a':
			n++
			if mode {
				m[event.Arguments[n]] = 5
			} else {
				m[event.Arguments[n]] = 1
			}
		case 'q':
			n++
			if mode {
				m[event.Arguments[n]] = 6
			} else {
				m[event.Arguments[n]] = 1
			}
		case 'I':
			fallthrough
		case 'e':
			fallthrough
		case 'b':
			fallthrough
		case 'k':
			if mode {
				n++
			}
		}
	}
	return m
}

// updateNames tries to update the name list occasionally.
func updateNames(irchuu *irc.Connection, c *config.Irc) {
	for {
		time.Sleep(time.Second * time.Duration(c.NamesUpdateInterval))
		irchuu.SendRawf("NAMES %v", c.Channel)
	}
}

// relayMessagesToIRC listens to the Telegram channel and sends every message
// into IRC.
func relayMessagesToIRC(r *relay.Relay, c *config.Irc, irchuu *irc.Connection) {
	for message := range r.TeleCh {
		if message.Extra["break"] == "true" {
			break
		}
		var messages []string
		if message.Extra["special"] == "" {
			messages = formatIRCMessages(message, c, 0)
		} else {
			messages = formatSpecialIRCMessages(message, c)
		}
		for _, m := range messages {
			irchuu.Privmsg(c.Channel, m)
			if c.FloodDelay != 0 {
				time.Sleep(time.Duration(c.FloodDelay) * time.Millisecond)
			}
		}
	}
}

// listenService listens to service messages and executes them.
func listenService(r *relay.Relay, c *config.Irc, irchuu *irc.Connection, names *map[string]int) {
	for f := range r.TeleServiceCh {
		switch f.Command {
		case "break":
			break
		case "announce":
			fallthrough
		case "bot":
			if len(f.Arguments) != 0 {
				irchuu.Privmsg(c.Channel, f.Arguments[0])
			}
		case "action":
			irchuu.Action(c.Channel, f.Arguments[0])
		case "kick":
			if len(f.Arguments) == 2 && f.Arguments[0] != irchuu.GetNick() {
				irchuu.Kick(f.Arguments[0], c.Channel,
					"by "+f.Arguments[1])
			}
		case "ops":
			ops := "Operators online: "
			for name, rank := range *names {
				if rank > 1 {
					ops += name + " "
				}
			}
			r.IRCServiceCh <- relay.ServiceMessage{"announce", []string{ops}}
		case "invite":
			if len(f.Arguments) != 0 {
				irchuu.SendRawf("INVITE %v %v", f.Arguments[0], c.Channel)
			}
		case "topic":
			irchuu.SendRawf("TOPIC %v", c.Channel)
		}

		if c.FloodDelay != 0 {
			time.Sleep(time.Duration(c.FloodDelay) * time.Millisecond)
		}
	}
}

// formatIRCMessage translates universal messages into IRC.
func formatIRCMessages(message relay.Message, c *config.Irc, prefixLen int) []string {
	var nick string

	if !message.Source {
		nick = c.Prefix + colorizeNick(message.Nick, c) + c.Postfix
	} else {
		nick = c.Prefix + formatNick(message, c) + c.Postfix
	}
	// 512 - 2 for CRLF - 7 for "PRIVMSG" - 4 for spaces - 9 just in case - 50 just in case
	acceptibleLength := 440 - len(nick) - len(c.Channel) - prefixLen

	if c.Ellipsis != "" {
		message.Text = strings.Replace(message.Text, "\n", c.Ellipsis, -1)
	}

	if message.Extra["forward"] != "" {
		message.Text = fmt.Sprintf("[\x0310fwd\x0f from @%v] %v",
			colorizeNick(message.Extra["forward"], c), message.Text)
	} else if message.Extra["forwardChat"] != "" {
		message.Text = fmt.Sprintf("[\x0310fwd\x0f from channel @%v] %v",
			colorizeNick(message.Extra["forwardChat"], c), message.Text)
	} else if message.Extra["forwardChatTitle"] != "" {
		message.Text = fmt.Sprintf("[\x0310fwd\x0f from channel %v] %v",
			colorizeNick(message.Extra["forwardChatTitle"], c), message.Text)
	} else if message.Extra["reply"] != "" && message.Extra["replyUserID"] != "" {
		message.Text = fmt.Sprintf("@%v, %v",
			colorizeNick(message.Extra["reply"], c), message.Text)
	} else if message.Extra["reply"] != "" {
		message.Text = fmt.Sprintf("%v, %v",
			colorizeNick(message.Extra["reply"], c), message.Text)
	}

	if message.Extra["edit"] != "" {
		message.Text = "\x034[edited]\x0f " + message.Text
	}

	if message.Extra["media"] != "" {
		message.Text = formatMediaMessage(message, c)
	}

	messages := splitLines(message.Text, acceptibleLength, nick+" ")
	return messages
}

// formatMediaMessage formats media messages.
// TODO: implement as a method?
// TODO: clean the code, reuse parts
// TODO: colorize
func formatMediaMessage(message relay.Message, c *config.Irc) string {
	text := message.Text
	if text != "" {
		text += " "
	}
	if message.Extra["url"] != "" {
		text += "( " + message.Extra["url"] + " ) "
	}
	intSize, _ := strconv.ParseUint(message.Extra["size"], 10, 64)
	size := bytefmt.ByteSize(intSize)
	switch message.Extra["media"] {
	case "sticker":
		fallthrough
	case "photo":
		text += fmt.Sprintf("(%v, %v×%v, %viB)",
			message.Extra["media"], message.Extra["width"],
			message.Extra["height"], size)
	case "document":
		text += fmt.Sprintf("(\"%v\", %viB)", message.Extra["mediaName"],
			size)
	case "audio":
		text += fmt.Sprintf("%v — %v (%vs, %viB)",
			message.Extra["performer"], message.Extra["mediaName"],
			message.Extra["duration"], size)
	case "video":
		text += fmt.Sprintf("(%v, %v×%v×%vs, %viB)",
			message.Extra["media"], message.Extra["width"],
			message.Extra["height"], message.Extra["duration"],
			size)
	case "voice":
		text += fmt.Sprintf("(%v, %vs, %viB)",
			message.Extra["media"], message.Extra["duration"],
			size)
	}
	return text
}

// formatSpecialIRCMessage translates special universal messages (service
// messages, e. g. pin, kick, etc) into IRC.
func formatSpecialIRCMessages(message relay.Message, c *config.Irc) (messages []string) {
	switch message.Extra["special"] {
	case "pin":
		if message.Extra["media"] != "" {
			message.Text = formatMediaMessage(message, c)
		}

		txt := []rune(message.Text)
		var s int
		// TODO: make configurable
		if len(txt) > 20 {
			s = 20
		} else {
			s = len(txt) - 1
		}
		messages = []string{fmt.Sprintf("%v pinned %v's message"+
			" \"%v...\".", colorizeNick(message.Extra["pin"], c),
			colorizeNick(message.Name(), c), string(txt[:s]))}
	case "newChatMember":
		if strconv.Itoa(message.FromID) == message.Extra["memberID"] {
			messages = []string{fmt.Sprintf("%v joined the group via invite link.",
				colorizeNick(message.Extra["memberName"], c))}
		} else {
			messages = []string{fmt.Sprintf("%v was added by %v.",
				colorizeNick(message.Extra["memberName"], c),
				colorizeNick(message.Name(), c))}
		}
	case "leftChatMember":
		if strconv.Itoa(message.FromID) == message.Extra["memberID"] {
			messages = []string{fmt.Sprintf("%v left the group.",
				colorizeNick(message.Extra["memberName"], c))}
		} else {
			messages = []string{fmt.Sprintf("%v was removed by %v.",
				colorizeNick(message.Extra["memberName"], c),
				colorizeNick(message.Name(), c))}
		}
	case "newChatTitle":
		messages = []string{fmt.Sprintf("Chat renamed to \"%v\" by %v.",
			message.Extra["title"], colorizeNick(message.Name(), c))}
	case "newChatPhoto":
		messages = []string{fmt.Sprintf("The chat photo has been changed by %v.",
			colorizeNick(message.Name(), c))}
	case "deleteChatPhoto":
		messages = []string{fmt.Sprintf("The chat photo has been deleted by %v.",
			colorizeNick(message.Name(), c))}
	case "KICK":
		messages = []string{fmt.Sprintf("%v was kicked by %v.",
			colorizeNick(message.Text, c),
			colorizeNick(message.Nick, c))}
	case "NICK":
		messages = []string{fmt.Sprintf("%v is now known as %v.",
			colorizeNick(message.Text, c),
			colorizeNick(message.Nick, c))}
	case "TOPIC":
		messages = []string{fmt.Sprintf("%v set the topic to \"%v\".",
			colorizeNick(message.Nick, c), message.Text)}
	case "ACTION":
		messages = []string{fmt.Sprintf("*%v %v*",
			colorizeNick(message.Nick, c), message.Text)}
	}
	return
}

// processCmd executes commands.
func processCmd(event *irc.Event, irchuu *irc.Connection, c *config.Irc, r *relay.Relay, db *sql.DB, names *map[string]int) {
	cmd := strings.SplitN(event.Message(), " ", 3)
	if len(cmd) < 2 {
		return
	}
	switch cmd[1] {
	case "help":
		texts := make([]string, 10)
		texts[0] = "Available commands:"
		texts[1] = c.Nick + " \x02help\x0f — show this help"
		texts[2] = c.Nick + " \x02ops\x0f — show Telegram group ops"
		texts[3] = c.Nick + " \x02count\x0f — show Telegram group user count"
		texts[8] = "\x02/ctcp " + irchuu.GetNick() +
			" version\x0f — get version"
		texts[9] = "Some of these commands are available in PM."
		if c.AllowStickers {
			texts[7] = c.Nick + " \x02sticker [id]\x0f — send a sticker"
		}
		if db != nil {
			texts[4] = c.Nick +
				" \x02hist [n]\x0f — get [n] last messages in PM"
			if c.Moderation {
				texts[5] = c.Nick +
					" \x02kick [nick || full name]\x0f —" +
					" kick a user from the Telegram group"
				texts[6] = c.Nick +
					" \x02unban [nick || full name]\x0f — unban a user"
			}
		}
		for _, text := range texts {
			if text != "" {
				irchuu.Privmsg(c.Channel, text)
				if c.FloodDelay != 0 {
					time.Sleep(time.Duration(c.FloodDelay) * time.Millisecond)
				}
			}
		}
	case "hist":
		if db != nil {
			var n int
			if len(cmd) > 2 && cmd[2] != "" {
				n, _ = strconv.Atoi(cmd[2])
			}
			go sendHistory(db, event.Nick, irchuu, c, n)
		}
	case "kick":
		if c.Moderation && len(cmd) > 2 {
			if (*names)[event.Nick] >= c.KickPermission {
				modifyUser(db, irchuu, r, cmd[2], c.Channel, false)
			} else {
				irchuu.Privmsg(c.Channel, "Insufficient permission.")
			}
		}
	case "ops":
		r.IRCServiceCh <- relay.ServiceMessage{"ops", nil}
	case "sticker":
		if c.AllowStickers && len(cmd) > 2 {
			time.Sleep(time.Duration(50) * time.Millisecond)
			r.IRCServiceCh <- relay.ServiceMessage{"sticker", []string{cmd[2]}}
		}
	case "count":
		r.IRCServiceCh <- relay.ServiceMessage{"count", nil}
	case "unban":
		if c.Moderation && len(cmd) > 2 {
			if (*names)[event.Nick] >= c.KickPermission {
				modifyUser(db, irchuu, r, cmd[2], c.Channel, true)
			} else {
				irchuu.Privmsg(c.Channel, "Insufficient permission.")
			}
		}
	}
}

// modifyUser kicks a Telegram user from the groupchat or unbans them. Mode true
// unbans, mode false kicks.
func modifyUser(db *sql.DB, irchuu *irc.Connection, r *relay.Relay, name, channel string, mode bool) {
	id, foundName, err := irchuubase.FindUser(name, db)
	if err == sql.ErrNoRows {
		irchuu.Privmsg(channel, "No such user.")
		return
	} else if err != nil {
		irchuu.Privmsg(channel, "An error occurred.")
		return
	}
	if mode {
		r.IRCServiceCh <- relay.ServiceMessage{"unban", []string{strconv.Itoa(id), foundName}}
	} else {
		r.IRCServiceCh <- relay.ServiceMessage{"kick", []string{strconv.Itoa(id), foundName}}
	}
}

// processPMCmd executes commands sent in private.
func processPMCmd(event *irc.Event, irchuu *irc.Connection, c *config.Irc, r *relay.Relay, db *sql.DB) {
	cmd := strings.Split(event.Message(), " ")
	if len(cmd) < 1 {
		return
	}
	switch cmd[0] {
	case "help":
		texts := make([]string, 5)
		texts[0] = "Available commands:"
		texts[1] = "\x02help\x0f — show this help"
		texts[3] = "\x02/ctcp " + irchuu.GetNick() +
			" version\x0f — get version info"
		texts[4] = "More commands are available in the channel."
		if db != nil {
			texts[2] = "\x02hist [n]\x0f — get [n] last messages"
		}
		for _, text := range texts {
			if text != "" {
				noticeOrMsg(irchuu, c.SendNotices, event.Nick, text)
				if c.FloodDelay != 0 {
					time.Sleep(time.Duration(c.FloodDelay) * time.Millisecond)
				}
			}
		}
	case "hist":
		if db != nil {
			var n int
			if len(cmd) > 1 && cmd[1] != "" {
				n, _ = strconv.Atoi(cmd[1])
			}
			go sendHistory(db, event.Nick, irchuu, c, n)
		}
	default:
		noticeOrMsg(irchuu, c.SendNotices, event.Nick, "No such command. Enter"+
			" \x02help\x0f for the list of commands.")
	}
}

// sendHistory retrieves the message history from DB and sends it to <nick>.
func sendHistory(db *sql.DB, nick string, irchuu *irc.Connection, c *config.Irc, n int) {
	if n == 0 || n > c.MaxHist {
		n = c.MaxHist
	}
	var msgs []relay.Message
	msgs, err := irchuubase.GetMessages(db, n)
	if err != nil {
		irchuu.Privmsgf(c.Channel, "%v: an error occurred during your request.",
			nick)
		return
	}
	l := len(msgs) - 1
	for m := range msgs {
		msg := msgs[l-m]
		date := "[\x0310" + msg.Date.Format("15:04:05") + "\x0f] "
		var rawMsgs []string
		if msg.Extra["special"] == "" {
			rawMsgs = formatIRCMessages(msg, c, 14)
		} else {
			rawMsgs = formatSpecialIRCMessages(msg, c)
		}
		for rawMsg := range rawMsgs {
			noticeOrMsg(irchuu, c.SendNotices, nick, date+rawMsgs[rawMsg])
			if c.FloodDelay != 0 {
				time.Sleep(time.Duration(c.FloodDelay) * time.Millisecond)
			}
		}
	}
}

// splitLines splits Unicode lines so that they are not longer than max bytes.
func splitLines(text string, max int, prefix string) []string {
	var lines []string
	size := 0
	var runes []rune
	for len(text) > 0 {
		r, s := utf8.DecodeRuneInString(text)
		if r == '\n' {
			lines = append(lines, prefix+string(runes))
			runes = nil
			size = 0
			text = text[s:]
		} else if size+s > max {
			lines = append(lines, prefix+string(runes))
			runes = nil
			size = 0
		} else {
			size += s
			runes = append(runes, r)
			text = text[s:]
		}
	}
	lines = append(lines, prefix+string(runes))
	return lines
}

// formatNick processes nicknames.
func formatNick(message relay.Message, c *config.Irc) string {
	nick := message.Name()

	if c.MaxLength != 0 && len(nick) > c.MaxLength {
		nick = nick[:c.MaxLength-1] + "…"
	}

	if c.Colorize {
		nick = colorizeNick(nick, c)
	}

	nick = "@" + nick
	return nick
}

// formatMessage creates a Message in the universal format of an IRC message.
func formatMessage(nick string, text string, action string) relay.Message {
	extra := make(map[string]string)
	extra["special"] = action

	return relay.Message{
		Date:   time.Now(),
		Source: false,
		Nick:   nick,
		Text:   text,
		Extra:  extra,
	}
}

// djb2 hashes the string and returns an integer.
func djb2(nick string) int32 {
	hash := int32(5381)
	for s := 0; s < len(nick); s++ {
		hash = ((hash << 5) + hash) + int32(nick[s])
	}
	return hash
}

// colorizeNick adds color codes to the nickname.
func colorizeNick(s string, c *config.Irc) string {
	if !c.Colorize {
		return s
	}
	i := djb2(s) % int32(len(c.Palette))
	if i < 0 {
		i += int32(len(c.Palette))
	}
	return "\x03" + c.Palette[i] + s + "\x03"
}
