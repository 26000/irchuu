package irchuu

import (
	"fmt"
	"github.com/26000/irchuu/config"
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
func Launch(c *config.Irc, wg *sync.WaitGroup, r *relay.Relay) {
	defer wg.Done()

	logger := log.New(os.Stdout, "IRC ", log.LstdFlags)
	irchuu := irc.IRC(c.Nick, "IRChuu~")
	irchuu.UseTLS = c.SSL
	irchuu.Password = c.ServerPassword

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
		logger.Printf("Unknown CTCP %v from %v\n", event.Arguments[1], event.Nick)
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
		logger.Printf("Nickname already in use, changed to %v\n", irchuu.GetNick())
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

	irchuu.AddCallback("INVITE", func(event *irc.Event) {
		logger.Printf("Invited to %v by %v\n", event.Arguments[1], event.Nick)
		if c.Channel == event.Arguments[1] {
			irchuu.Join(fmt.Sprintf("%v %v", c.Channel, c.ChanPassword))
		}
	})

	// On joined...
	irchuu.AddCallback("366", func(event *irc.Event) {
		logger.Printf("Joined %v\n", event.Arguments[1])
		go relayMessagesToIRC(r, c, irchuu)
	})

	irchuu.AddCallback("PRIVMSG", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			r.IRCh <- formatMessage(event.Nick, event.Message(), "")
		} else {
			logger.Printf("Message from %v: %v\n",
				event.Nick, event.Message())
		}
	})

	irchuu.AddCallback("CTCP_ACTION", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			r.IRCh <- formatMessage(event.Nick, event.Message(), "ACTION")
		} else {
			logger.Printf("CTCP ACTION from %v: %v\n",
				event.Nick, event.Message())
		}
	})

	irchuu.AddCallback("KICK", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			r.IRCh <- formatMessage(event.Nick, event.Arguments[1], "KICK")
		}
	})

	irchuu.AddCallback("TOPIC", func(event *irc.Event) {
		if event.Arguments[0] == c.Channel {
			r.IRCh <- formatMessage(event.Nick, event.Arguments[1], "TOPIC")
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
}

// relayMessagesToIRC listens to the Telegram channel and sends every message
// into IRC.
func relayMessagesToIRC(r *relay.Relay, c *config.Irc, irchuu *irc.Connection) {
	for message := range r.TeleCh {
		messages := formatIRCMessages(message, c)
		for m := range messages {
			if c.FloodDelay != 0 {
				time.Sleep(time.Duration(c.FloodDelay) * time.Millisecond)
			}
			irchuu.Privmsg(c.Channel, messages[m])
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

	messages := splitLines(message.Text, acceptibleLength, nick+" ")
	/*
		if c.Ellipsis != "" {
			message.Text = strings.Replace(message.Text, "\n", c.Ellipsis, -1)
			if len(message.Text) > acceptibleLength {
				// Number of parts.
				var l int
				if len(message.Text)%acceptibleLength != 0 {
					l = len(message.Text)/acceptibleLength + 1
				} else {
					l = len(message.Text) / acceptibleLength
				}
				//l = int(math.Ceil(float64(len(message.Text)) / acceptibleLength))

				messages = make([]string, l)
				for i := 1; i < l; i++ {
					messages[i-1] = nick + " " + message.Text[(i-1)*acceptibleLength:i*acceptibleLength]
				}
				messages[l-1] = nick + " " + message.Text[(l-1)*acceptibleLength:len(message.Text)]
			} else {
				messages = []string{nick + " " + message.Text}
			}
		} else {
		}*/
	return messages
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
		message.Nick = message.Name
		addAt = false
	}

	if c.Colorize {
		nick = getColoredNick(message.Nick, c)
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
		Source: "IRC",
		Nick:   nick,
		Text:   text,
		Date:   time.Now().Unix(),
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

// getColoredNick adds color codes to the nickname.
func getColoredNick(s string, c *config.Irc) string {
	i := djb2(s) % len(c.Palette)
	if i < 0 {
		i += len(c.Palette)
	}
	return "\x03" + c.Palette[i] + s + "\x03"
}
