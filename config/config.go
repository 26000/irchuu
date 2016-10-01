package config

import (
	"gopkg.in/ini.v1"
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
	return nil, irc, tg
}

// PopulateConfig copies the sample config to <path>.
func PopulateConfig(file string) error {
	config := `# IRChuu configuration file. See https://github.com/26000/irchuu for help.
[telegram]
token = myToken
group = 7654321

[irc]
server = irc.rizon.net
port = 6667
ssl = false
serverpassword = # leave blank if not set

nick = irchuu
password = # if not blank, will use NickServ to identify
sasl = false # if true, will use SASL instead of NickServ

channel = irchuu # without '#'!
chanpassword = # leave blank if not set
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
}

// Telegram is the struct of Telegram part in config.
type Telegram struct {
	Token string
	Group int64
}
