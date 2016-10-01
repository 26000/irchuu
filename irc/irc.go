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

func Launch(c *config.Irc, wg *sync.WaitGroup, r *relay.Relay) {
	defer wg.Done()

	logger := log.New(os.Stdout, "IRC ", log.LstdFlags)
	irchuu := irc.IRC(c.Nick, "IRChuu~")
	irchuu.UseTLS = c.SSL
	irchuu.Password = c.Password
	irchuu.Log = logger

	irchuu.ReplaceCallback("CTCP_VERSION", 0, func(event *irc.Event) {
		logger.Printf("Incoming CTCP_VERSION from %v\n", event.Nick)
		irchuu.Noticef(event.Nick, "\001VERSION IRChuu! v%v (https://github.com/26000/irchuu), based on go-ircevent\001", config.VERSION)
	})

	irchuu.AddCallback("CTCP", func(event *irc.Event) {
		logger.Printf("Incoming unknown CTCP from %v\n", event.Nick)
	})

	irchuu.AddCallback("PRIVMSG", func(event *irc.Event) {
		if event.Arguments[0] == "#"+c.Channel {
			r.IRCh <- relay.Message{
				Nick: event.Nick,
				Text: event.Message(),
			}
		}
	})

	err := irchuu.Connect(fmt.Sprintf("%v:%d", c.Server, c.Port))
	if err != nil {
		logger.Printf("Cannot connect: %v", err)
	}
	irchuu.Join(fmt.Sprintf("#%v %v", c.Channel, c.ChanPassword))
	go relayMessagesToIRC(r, c, irchuu)
}

func relayMessagesToIRC(r *relay.Relay, c *config.Irc, irchuu *irc.Connection) {
	for message := range r.TeleCh {
		irchuu.Privmsgf("#"+c.Channel, "<%v> %v", message.Nick, message.Text)
	}
}
