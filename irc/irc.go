package irchuu

import (
	"fmt"
	"github.com/26000/irchuu/config"
	"github.com/thoj/go-ircevent"
	"log"
	"os"
	"sync"
)

func Launch(c *config.Irc, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := log.New(os.Stdout, "IRC ", log.LstdFlags)
	irchuu := irc.IRC(c.Nick, "IRChuu~")
	irchuu.UseTLS = c.SSL
	irchuu.Password = c.Password
	irchuu.Log = logger

	irchuu.RemoveCallback("CTCP_VERSION", 0)
	irchuu.AddCallback("CTCP_VERSION", func(event *irc.Event) {
		println(event.Nick)
		println(event.Message())
	})

	err := irchuu.Connect(fmt.Sprintf("%v:%d", c.Server, c.Port))
	if err != nil {
		logger.Printf("Cannot connect: %v", err)
	}
	irchuu.Join(fmt.Sprintf("#%v %v", c.Channel, c.ChanPassword))
}
