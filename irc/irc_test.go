package irchuu

import (
	"testing"
	"time"

	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/relay"

	"github.com/stretchr/testify/assert"
)

var (
	ircConf1 = &config.Irc{
		Server:         "irc.rizon.net",
		Port:           6667,
		SSL:            false,
		ServerPassword: "",

		Nick:     "irchuu",
		Password: "passwordless",
		SASL:     false,

		Channel:      "irchuu",
		ChanPassword: "irchuu",

		Colorize:   true,
		Palette:    []string{"1", "2", "3", "4", "5", "6", "7"},
		Prefix:     "<",
		Postfix:    ">",
		MaxLength:  24,
		Ellipsis:   "… ",
		FloodDelay: 500,

		Moderation:          true,
		KickPermission:      4,
		MaxHist:             42,
		NamesUpdateInterval: 300,

		Debug: false,
	}

	ircConf2 = &config.Irc{
		Server:         "irc.rizon.net",
		Port:           6697,
		SSL:            true,
		ServerPassword: "",

		Nick:     "irchuuTwo",
		Password: "passwordful",
		SASL:     false,

		Channel:      "irchuu",
		ChanPassword: "irchuu",

		Colorize:   true,
		Palette:    []string{"2", "3", "5", "6", "7", "15"},
		Prefix:     "",
		Postfix:    "",
		MaxLength:  0,
		Ellipsis:   "",
		FloodDelay: 250,

		Moderation:          true,
		KickPermission:      4,
		MaxHist:             44,
		NamesUpdateInterval: 300,

		Debug: false,
	}

	centralTime, _ = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006",
		"Thu Nov 3 15:41:15 +0300 MSK 2016")

	testMessages = []*relay.Message{
		&relay.Message{
			Date:   centralTime,
			Source: false,
			Nick:   "irchuu",
			Text:   "konnichiha!",

			Extra: map[string]string{},
		},
	}

	colorizeNickTestData = map[string][2]string{
		"irchuu":    [2]string{"\x036irchuu\x0f", "\x033irchuu\x0f"},
		"26000":     [2]string{"\x03326000\x0f", "\x031526000\x0f"},
		"nick":      [2]string{"\x032nick\x0f", "\x037nick\x0f"},
		"irc":       [2]string{"\x037irc\x0f", "\x036irc\x0f"},
		"github":    [2]string{"\x032github\x0f", "\x032github\x0f"},
		"kotobank":  [2]string{"\x037kotobank\x0f", "\x032kotobank\x0f"},
		"koto":      [2]string{"\x036koto\x0f", "\x035koto\x0f"},
		"crypto":    [2]string{"\x031crypto\x0f", "\x037crypto\x0f"},
		"Athena":    [2]string{"\x037Athena\x0f", "\x035Athena\x0f"},
		"a_word":    [2]string{"\x034a_word\x0f", "\x0315a_word\x0f"},
		"snowflake": [2]string{"\x037snowflake\x0f", "\x0315snowflake\x0f"},
	}

	// djb2 worked as designed as of 2016-11-02 21:55:29, so i've generated test
	// data using the function itself... https://play.golang.org/p/g08Au2v2Vg
	djb2TestData = map[string]int32{
		"irchuu":    85175221,
		"26000":     195442781,
		"nick":      2090544394,
		"irc":       193495203,
		"github":    -3157944,
		"kotobank":  -1302459138,
		"koto":      2090443682,
		"crypto":    -148837850,
		"Athena":    -1477692490,
		"a_word":    -249714175,
		"snowflake": 1205184623,
	}
)

func TestColorizeNick(t *testing.T) {
	assert := assert.New(t)
	for nick, arr := range colorizeNickTestData {
		ircConf = ircConf1
		assert.Equal(arr[0], colorizeNick(nick))
		ircConf = ircConf2
		assert.Equal(arr[1], colorizeNick(nick))
	}
}

func TestDjb2(t *testing.T) {
	assert := assert.New(t)
	for i, k := range djb2TestData {
		assert.Equal(k, djb2(i))
	}
}
