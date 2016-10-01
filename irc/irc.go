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
	irchuu.Password = c.Password
	irchuu.Log = logger

	go logErrors(logger, irchuu.ErrorChan())

	irchuu.ReplaceCallback("CTCP VERSION", 0, func(event *irc.Event) {
		logger.Printf("Incoming CTCP_VERSION from %v\n", event.Nick)
		irchuu.Noticef(event.Nick, "\001VERSION IRChuu! v%v (https://github.com/26000/irchuu), based on %v\001", config.VERSION, irc.VERSION)
	})

	irchuu.AddCallback("CTCP", func(event *irc.Event) {
		logger.Printf("Incoming unknown CTCP %v from %v\n", event.Arguments[1], event.Nick)
	})

	irchuu.AddCallback("INVITE", func(event *irc.Event) {
		logger.Printf("Invited to %v by %v\n", event.Arguments[1], event.Nick)
		if c.Channel == event.Arguments[0] {
			irchuu.Join(fmt.Sprintf("#%v %v", c.Channel, c.ChanPassword))
		}
	})

	irchuu.AddCallback("PRIVMSG", func(event *irc.Event) {
		if event.Arguments[0] == "#"+c.Channel {
			r.IRCh <- relay.Message{
				Nick: event.Nick,
				Text: event.Message(),
			}
		}
	})

	// Errors
	irchuu.AddCallback("461", func(event *irc.Event) {
		logger.Printf("Error: ERR_NEEDMOREPARAMS\n")
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

	err := irchuu.Connect(fmt.Sprintf("%v:%d", c.Server, c.Port))
	if err != nil {
		logger.Printf("Cannot connect: %v\n", err)
	}
	irchuu.Join(fmt.Sprintf("#%v %v", c.Channel, c.ChanPassword))
	go relayMessagesToIRC(r, c, irchuu)
}

// relayMessagesToIRC listens to the Telegram channel and sends every message
// into IRC.
func relayMessagesToIRC(r *relay.Relay, c *config.Irc, irchuu *irc.Connection) {
	for message := range r.TeleCh {
		irchuu.Privmsgf("#"+c.Channel, "<%v> %v", message.Nick, message.Text)
	}
}

// logErrors listens to an error channel and logs errors.
func logErrors(logger *log.Logger, ch chan error) {
	for e := range ch {
		logger.Printf("Error: %v\n", e)
	}
}
