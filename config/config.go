package config

import (
	"gopkg.in/ini.v1"
	"html"
	"io/ioutil"
	"os"
)

const (
	VERSION = "0.2.6"
	LAYER   = 2
)

// ReadConfig reads the configuration file.
func ReadConfig(path string) (error, *Irc, *Telegram, *Irchuu) {
	cfg, err := ini.InsensitiveLoad(path)
	cfg.BlockMode = false
	tg, irc, irchuu := new(Telegram), new(Irc), new(Irchuu)
	err = cfg.Section("telegram").MapTo(tg)
	if err != nil {
		return err, irc, tg, irchuu
	}
	err = cfg.Section("irc").MapTo(irc)
	if err != nil {
		return err, irc, tg, irchuu
	}

	tg.Prefix = html.EscapeString(tg.Prefix)
	tg.Postfix = html.EscapeString(tg.Postfix)

	err = cfg.Section("irchuu").MapTo(irchuu)
	if err != nil {
		return err, irc, tg, irchuu
	}

	return nil, irc, tg, irchuu
}

// PopulateConfig copies the sample config to <path>.
func PopulateConfig(file string) error {
	config := `# IRChuu configuration file. See https://github.com/26000/irchuu for help.
[irchuu]
dburi = # URI of your PostgreSQL database
        # if blank, logging and kicking Telegram users from IRC will be unavailable
sendstats = true # send usage statistics
                 # data what you will share:
                 # - the hashes of your Telegram group id and IRC channel
		 # - your IRChuu version
                 # - your IP
checkupdates = true # check for updates on each start

[telegram]
token = myToken
group = 7654321

TTL = 300 # (seconds) If message was sent more than <TTL> seconds ago, it won't be relayed
          # 0 to disable

prefix = < # will be added before nicks
postfix = > # will be added after nicks

allowbots = true # allow sending messages without nick prefix (/bot command)
allowinvites = false # allow invites to the IRC channel from Telegram
moderation = true # allow moderators in Telegram to kick users from IRC
                  # (bot needs to have permissions for that in IRC)

downloadmedia = false # downloads media to $XDG_DATA_HOME/irchuu or 
                      # ~/.local/share/irchuu/
servemedia = false # serve mediafiles over HTTP and append links to the messages
                   # requires downloadmedia to be true
readtimeout = 100 # (seconds)
writetimeout = 20 # (seconds)
serverport = 8080 # port for media server
baseurl = http://localhost:8080 # usually your protocol plus IP or domain plus the port
                                # WITHOUT THE TRAILING SLASH

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

moderation = true # allow ops in IRC to kick users from Telegram
                  # (bot needs to be a moderator in Telegram)
                  # works only when dbURI is set

kickpermission = 4 # who can kick users from the Telegram group:
                   # 1 — everybody, 2 — voices, 3 — halfops, 4 — ops, 5 — protected/admins, 6 — the owner

NamesUpdateInterval = 600 # (seconds) how often to poll the server for the
                          # users list

maxhist = 40 # maximum number of messages sent on ./hist command in IRC
             # works only when dbURI is set
`
	return ioutil.WriteFile(file, []byte(config), os.FileMode(0600))
}

// Irchuu is the struct of common part in config.
type Irchuu struct {
	DBURI        string
	SendStats    bool
	CheckUpdates bool
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

	Moderation          bool
	KickPermission      int
	MaxHist             int
	NamesUpdateInterval int

	Debug bool
}

// Telegram is the struct of Telegram part in config.
type Telegram struct {
	Token string
	Group int64

	TTL int64

	Prefix  string
	Postfix string

	AllowBots    bool
	AllowInvites bool
	Moderation   bool

	DownloadMedia bool
	ServeMedia    bool
	ServerPort    uint16
	ReadTimeout   int
	WriteTimeout  int
	BaseURL       string
	DataDir       string
}

// muDeiPt5mAI8Ue==
