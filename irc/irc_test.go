package irchuu

import (
	"github.com/26000/irchuu/config"
	"github.com/26000/irchuu/relay"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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
		"irchuu":    [2]string{"\x036irchuu\x03", "\x033irchuu\x03"},
		"26000":     [2]string{"\x03326000\x03", "\x031526000\x03"},
		"nick":      [2]string{"\x032nick\x03", "\x037nick\x03"},
		"irc":       [2]string{"\x037irc\x03", "\x036irc\x03"},
		"github":    [2]string{"\x032github\x03", "\x032github\x03"},
		"kotobank":  [2]string{"\x037kotobank\x03", "\x032kotobank\x03"},
		"koto":      [2]string{"\x036koto\x03", "\x035koto\x03"},
		"crypto":    [2]string{"\x031crypto\x03", "\x037crypto\x03"},
		"Athena":    [2]string{"\x037Athena\x03", "\x035Athena\x03"},
		"a_word":    [2]string{"\x034a_word\x03", "\x0315a_word\x03"},
		"snowflake": [2]string{"\x037snowflake\x03", "\x0315snowflake\x03"},
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
		assert.Equal(arr[0], colorizeNick(nick, ircConf1))
		assert.Equal(arr[1], colorizeNick(nick, ircConf2))
	}
}

func TestDjb2(t *testing.T) {
	assert := assert.New(t)
	for i, k := range djb2TestData {
		assert.Equal(k, djb2(i))
	}
}
