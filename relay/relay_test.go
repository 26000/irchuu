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
	}
)

func TestMessage_Name(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(testMessages[0].Name(), "irchuu")
}
