package relay

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	centralTime, _ = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006",
		"Thu Nov 3 15:41:15 +0300 MSK 2016")

	testMessages = []*Message{
		&Message{
			Date:   centralTime,
			Source: false,
			Nick:   "irchuu",
			Text:   "konnichiha!",

			Extra: map[string]string{},
		},
		&Message{
			Date:   centralTime,
			Source: true,
			Nick:   "",
			Text:   "konnichiha!",

			ID:        42,
			FromID:    26,
			FirstName: "IRChuu~",
			LastName:  "Bot",

			Extra: map[string]string{},
		},
	}
)

func TestMessage_Name(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("irchuu", testMessages[0].Name())
	assert.Equal("IRChuu~ Bot", testMessages[1].Name())
}

func TestNewRelay(t *testing.T) {
	assert := assert.New(t)
	r := NewRelay()
	m := make(chan Message)
	s := make(chan ServiceMessage)
	assert.IsType(m, r.TeleCh)
	assert.IsType(m, r.IRCh)
	assert.IsType(s, r.TeleServiceCh)
	assert.IsType(s, r.IRCServiceCh)
}
