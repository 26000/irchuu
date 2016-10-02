package irchuu

import (
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/relay"
	"github.com/thoj/go-ircevent"
	"log"
	"os"
	"sync"
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

	irchuu.Log = logger
	irchuu.QuitMessage = "IRChuu!bye"
	irchuu.Version = fmt.Sprintf("IRChuu! v%v (https://github.com/26000/irchuu), based on %v", config.VERSION, irc.VERSION)

	//go logErrors(logger, irchuu.ErrorChan())

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
		irchuu.Join(fmt.Sprintf("#%v %v", c.Channel, c.ChanPassword))
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
		if "#"+c.Channel == event.Arguments[1] {
			irchuu.Join(fmt.Sprintf("#%v %v", c.Channel, c.ChanPassword))
		}
	})

	// On joined...
	irchuu.AddCallback("353", func(event *irc.Event) {
		logger.Printf("Joined %v. Users online: %v\n", event.Arguments[2], event.Arguments[3])
		go relayMessagesToIRC(r, c, irchuu)
	})

	irchuu.AddCallback("PRIVMSG", func(event *irc.Event) {
		if event.Arguments[0] == "#"+c.Channel {
			r.IRCh <- relay.Message{
				Nick: event.Nick,
				Text: event.Message(),
			}
		} else {
			logger.Printf("Message from %v: %v\n",
				event.Nick, event.Message())
		}
	})

	// On connected...
	irchuu.AddCallback("001", func(event *irc.Event) {
		if !c.SASL && c.Password != "" {
			logger.Println("Trying to authenticate via NickServ")
			irchuu.Privmsgf("NickServ", "IDENTIFY %v", c.Password)
		}

		irchuu.Join(fmt.Sprintf("#%v %v", c.Channel, c.ChanPassword))
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
		var nick string
		if c.Colorize {
			nick = getColoredNick(message.Nick, c)
		} else {
			nick = message.Nick
		}
		irchuu.Privmsgf("#"+c.Channel, "<%v> %v", nick, message.Text)
	}
}

// logErrors listens to an error channel and logs errors.
func logErrors(logger *log.Logger, ch chan error) {
	for e := range ch {
		logger.Printf("Error: %v\n", e)
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
