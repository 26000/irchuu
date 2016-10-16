// irchuu contains everything related to the IRC part of IRChuu.
package irchuu

import (
	"database/sql"
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/db"
	"github.com/26000/irchuu/relay"
	"github.com/thoj/go-ircevent"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Launch starts the IRC bot and waits for messages.
func Launch(c *config.Irc, wg *sync.WaitGroup, r *relay.Relay, db *sql.DB) {
	defer wg.Done()

	logger := log.New(os.Stdout, "IRC ", log.LstdFlags)
	irchuu := irc.IRC(c.Nick, "IRChuu~")
	irchuu.UseTLS = c.SSL
	irchuu.Password = c.ServerPassword

	var names []string

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
		logger.Printf("Unknown CTCP %v from %v\n", event.Arguments[1],
			event.Nick)
	})

	irchuu.AddCallback("NOTICE", func(event *irc.Event) {
		logger.Printf("Notice from %v: %v\n",
			event.Nick, event.Message())
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
			r.IRCServiceCh <- relay.ServiceMessage{"announce", "I need to be an operator in IRC for that action."}
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
			fmt.Sprintf("Invited %v to %v.", event.Arguments[1],
				event.Arguments[2])}

		r.IRCServiceCh <- s
		r.TeleServiceCh <- s
	})

	// TODO: add bold for these messages
	irchuu.AddCallback("443", func(event *irc.Event) {
		s := relay.ServiceMessage{"announce",
			fmt.Sprintf("User %v is already on channel.",
				event.Arguments[1])}

		r.IRCServiceCh <- s
	})

	irchuu.AddCallback("401", func(event *irc.Event) {
		s := relay.ServiceMessage{"announce",
			fmt.Sprintf("No such nick: %v.", event.Arguments[1])}

		r.IRCServiceCh <- s
		r.TeleServiceCh <- s
	})

	// On joined...
	irchuu.AddCallback("JOIN", func(event *irc.Event) {
		logger.Printf("Joined %v\n", event.Arguments[0])
		go relayMessagesToIRC(r, c, irchuu)
		go listenService(r, c, irchuu)

	})

	// Topic
	irchuu.AddCallback("332", func(event *irc.Event) {
		if event.Arguments[1] == c.Channel {
			r.IRCServiceCh <- relay.ServiceMessage{"announce",
				fmt.Sprintf("The topic for %v is %v.",
					c.Channel, event.Arguments[2])}
		}
	})

	// No topic
	irchuu.AddCallback("331", func(event *irc.Event) {
		if event.Arguments[1] == c.Channel {
			r.IRCServiceCh <- relay.ServiceMessage{"announce",
				"No topic is set."}
		}
	})

	// Names
	irchuu.AddCallback("353", func(event *irc.Event) {
		if event.Arguments[2] == c.Channel {
			names = append(names, strings.Split(event.Arguments[3],
				" ")...)
		}
	})

	// End of names
	irchuu.AddCallback("366", func(event *irc.Event) {
		if event.Arguments[1] == c.Channel {
			ops := "Operators online: "
			for _, name := range names {
				if name[0] == '~' || name[0] == '&' || name[0] == '@' || name[0] == '%' {
					ops += name + " "
				}
			}
			names = nil
			r.IRCServiceCh <- relay.ServiceMessage{"announce", ops}
		}
	})

	irchuu.AddCallback("PRIVMSG", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			f := formatMessage(event.Nick, event.Message(), "")
			r.IRCh <- f
			if db != nil {
				go irchuubase.Log(f, db, logger)
			}
			if strings.HasPrefix(event.Message(), c.CMDPrefix) {
				processCmd(event, irchuu, c, r)
			}
		} else {
			logger.Printf("Message from %v: %v\n",
				event.Nick, event.Message())
			irchuu.Privmsg(event.Nick,
				"I work only on my channel. "+
					"https://github.com/26000/irchuu"+
					"for more info.")
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
			r.IRCh <- f
			if db != nil {
				go irchuubase.Log(f, db, logger)
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

// relayMessagesToIRC listens to the Telegram channel and sends every message
// into IRC.
func relayMessagesToIRC(r *relay.Relay, c *config.Irc, irchuu *irc.Connection) {
	for message := range r.TeleCh {
		messages := formatIRCMessages(message, c)
		for m := range messages {
			irchuu.Privmsg(c.Channel, messages[m])
			if c.FloodDelay != 0 {
				time.Sleep(time.Duration(c.FloodDelay) * time.Millisecond)
			}
		}
	}
}

// listenService listens to service messages and executes them.
func listenService(r *relay.Relay, c *config.Irc, irchuu *irc.Connection) {
	for f := range r.TeleServiceCh {
		switch f.Command {
		case "announce":
			fallthrough
		case "bot":
			irchuu.Privmsg(c.Channel, f.Arguments)
		case "kick":
			if f.Arguments != irchuu.GetNick() {
				irchuu.Kick(f.Arguments, c.Channel,
					"relayed from Telegram")
			}
		case "ops":
			irchuu.SendRawf("NAMES %v", c.Channel)
		case "invite":
			irchuu.SendRawf("INVITE %v %v", f.Arguments, c.Channel)
		case "topic":
			irchuu.SendRawf("TOPIC %v", c.Channel)
		}

		if c.FloodDelay != 0 {
			time.Sleep(time.Duration(c.FloodDelay) * time.Millisecond)
		}
	}
}

// formatIRCMessage translates universal messages into IRC.
// TODO: find and colorize mentions
func formatIRCMessages(message relay.Message, c *config.Irc) []string {
	nick := c.Prefix + formatNick(message, c) + c.Postfix
	// 512 - 2 for CRLF - 7 for "PRIVMSG" - 4 for spaces - 9 just in case - 50 just in case
	acceptibleLength := 440 - len(nick) - len(c.Channel)

	if c.Ellipsis != "" {
		message.Text = strings.Replace(message.Text, "\n", c.Ellipsis, -1)
	}

	if message.Extra["forward"] != "" {
		message.Text = fmt.Sprintf("[\x0311fwd\x0f from @%v] %v",
			colorizeNick(message.Extra["forward"], c), message.Text)
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

	messages := splitLines(message.Text, acceptibleLength, nick+" ")
	return messages
}

// processCmd executes commands.
func processCmd(event *irc.Event, irchuu *irc.Connection, c *config.Irc, r *relay.Relay) {
	cmd := strings.Split(event.Message()[len(c.CMDPrefix):], " ")
	switch cmd[0] {
	case "help":
		irchuu.Privmsg(c.Channel, "Available commands:")
		irchuu.Privmsgf(c.Channel, "%vhelp — show this help",
			c.CMDPrefix)
		irchuu.Privmsgf(c.Channel,
			"%vhist [n] — get [n] last messages in PM", c.CMDPrefix)
		irchuu.Privmsgf(c.Channel,
			"%vuhist [n] — get [n] last messages with user IDs in PM",
			c.CMDPrefix)
		irchuu.Privmsgf(c.Channel,
			"%vkick [id] — kick a user in Telegram", c.CMDPrefix)
		irchuu.Privmsgf(c.Channel,
			"%vops — show moderators in Telegram", c.CMDPrefix)
		irchuu.Privmsgf(c.Channel,
			"%vcount — show users count in Telegram", c.CMDPrefix)
		irchuu.Privmsgf(c.Channel,
			"%vunban [id] — unban a user in Telegram", c.CMDPrefix)
		irchuu.Privmsgf(c.Channel,
			"/ctcp %v VERSION — get version info", irchuu.GetNick())
	case "uhist":
	case "hist":
	case "kick":
	case "ops":
		r.IRCServiceCh <- relay.ServiceMessage{"ops", ""}
	case "count":
		r.IRCServiceCh <- relay.ServiceMessage{"count", ""}
	case "unban":
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
	var nick string
	addAt := true

	if message.Nick == "" {
		name := message.FirstName
		if message.LastName != "" {
			name += " " + message.LastName
		}
		message.Nick = name
		addAt = false
	}

	if c.Colorize {
		nick = colorizeNick(message.Nick, c)
	} else {
		nick = message.Nick
	}

	if c.MaxLength != 0 && len(message.Nick) > c.MaxLength {
		message.Nick = message.Nick[:c.MaxLength-1] + "…"
	}

	if addAt {
		nick = "@" + nick
	}
	return nick
}

// formatMessage creates a Message in the universal format of an IRC message.
func formatMessage(nick string, text string, action string) relay.Message {
	extra := make(map[string]string)

	switch action {
	case "":
	case "ACTION":
		extra["CTCP"] = "ACTION"
	case "KICK":
		extra["KICK"] = "true"
	case "TOPIC":
		extra["TOPIC"] = "true"
	}

	return relay.Message{
		Date:   time.Now(),
		Source: false,
		Nick:   nick,
		Text:   text,
		Extra:  extra,
	}
}

// djb2 hashes the string and returns an integer.
func djb2(nick string) int {
	hash := 5381
	for s := 0; s < len(nick); s++ {
		hash = ((hash << 5) + hash) + int(nick[s])
	}
	return hash
}

// colorizeNick adds color codes to the nickname.
func colorizeNick(s string, c *config.Irc) string {
	i := djb2(s) % len(c.Palette)
	if i < 0 {
		i += len(c.Palette)
	}
	return "\x03" + c.Palette[i] + s + "\x03"
}
