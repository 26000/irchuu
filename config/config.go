package config

import (
	"gopkg.in/ini.v1"
	"html"
	"io/ioutil"
	"os"
)

const VERSION = "0.0.0"

// ReadConfig reads the configuration file.
func ReadConfig(path string) (error, *Irc, *Telegram) {
	cfg, err := ini.InsensitiveLoad(path)
	cfg.BlockMode = false
	tg, irc := new(Telegram), new(Irc)
	err = cfg.Section("telegram").MapTo(tg)
	if err != nil {
		return err, irc, tg
	}
	err = cfg.Section("irc").MapTo(irc)
	if err != nil {
		return err, irc, tg
	}

	tg.Prefix = html.EscapeString(tg.Prefix)
	tg.Postfix = html.EscapeString(tg.Postfix)

	return nil, irc, tg
}

// PopulateConfig copies the sample config to <path>.
func PopulateConfig(file string) error {
	config := `# IRChuu configuration file. See https://github.com/26000/irchuu for help.
[telegram]
token = myToken
group = 7654321

TTL = 300 # (seconds) If message was sent more than <TTL> seconds ago, it won't be relayed
          # 0 to disable

prefix = < # will be added before nicks
postfix = > # will be added after nicks

[irc]
server = irc.rizon.net
port = 6667
ssl = false
serverpassword = # leave blank if not set

nick = irchuu
password = # if not blank, will use NickServ to identify
sasl = false # if true, will use SASL instead of NickServ

channel = ` + "`" + `#irchuu` + "`" + ` # must be surrounded with backticks
chanpassword = # leave blank if not set

colorize = true # colorize nicknames? (based on djb2)
palette = 1,2,3,4,5,6,9,10,11,12,13 # colors to be used, either codes or names
prefix = < # will be added before nicks
postfix = > # will be added after nicks
maxlength = 18 # maximum username length allowed, will be ellipsised if longer
               # set to 0 to disable
ellipsis = "… " # lines in multi-line messages will be divided with this
                # leave blank to send them as separate messages

flooddelay = 500 # (milliseconds) delay with which parts of multi-line message
                 # are sent to prevent anti-flood from kicking the bot
`
	return ioutil.WriteFile(file, []byte(config), os.FileMode(0600))
}

// Irc is the stuct of IRC part in config.
type Irc struct {
	Server         string
	Port           uint16
	SSL            bool
	ServerPassword string

	Nick     string
	Password string
	SASL     bool

	Channel      string
	ChanPassword string

	Colorize   bool
	Palette    []string
	Prefix     string
	Postfix    string
	MaxLength  int
	Ellipsis   string
	FloodDelay int

	Debug bool
}

// Telegram is the struct of Telegram part in config.
type Telegram struct {
	Token string
	Group int64

	TTL int64

	Prefix  string
	Postfix string
}
