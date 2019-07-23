package config

import (
	"html"
	"io/ioutil"
	"os"

	"gopkg.in/ini.v1"
)

const (
	// VERSION contains the IRChuu~ version.
	VERSION = "0.9.1"
	// LAYER contains IRChuu~ version in an integer (for comparison with
	// HQ server's last version).
	LAYER = 14
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
	if tg.KomfPublicURL == "" {
		tg.KomfPublicURL = tg.Komf
	}

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

# URI of your PostgreSQL database
# if blank, logging and kicking Telegram users from IRC will be unavailable
# (you will have to specify ?sslmode=disable if your database doesn't have TLS)
#
# examples:
# postgres://user:password@example.org:5432/database
# postgres://irchuu:irchuu@localhost/irchuu?sslmode=disable
dburi = 

# send usage statistics
# data what you will share:
# - the hashes of your Telegram group id and IRC channel
# - your IRChuu version
sendstats = true

# check for updates on each start
checkupdates = true

[telegram]
token = myToken
group = 7654321

# If message was sent more than <TTL> seconds ago, it won't be relayed
# 0 to disable
TTL = 300 # (seconds)

# prefix and postfix will be added before and after nicks
prefix = <
postfix = >

# allow sending messages without nick prefix (/bot command)
allowbots = true

# allow invites to the IRC channel from Telegram
allowinvites = false

# allow moderators in Telegram to kick users from IRC
# (bot needs to have permissions for that in IRC)
moderation = true

# download all media files to $XDG_DATA_HOME/irchuu or 
downloadmedia = false

# 'none', 'server' or 'pomf', where to store mediafiles from Telegram to show
# them in IRC
#
# 'server' will serve files over HTTP(S), needs 'serverport', 'baseurl',
# 'readtimeout' and 'writetimeout' to be set, 'downloadmedia' must be true
#
# 'pomf' will upload all media files to a pomf clone, needs 'pomf' to be set
#
# 'komf' will upload all media files to a komf (https://github.com/koto-bank/komf), needs 'komf' to be set
storage = none

## SERVER
# if certfilepath and keyfilepath are not nil, then will serve using HTTPS
certfilepath =
keyfilepath =

# request timeouts for the media server
readtimeout = 100 # (seconds)
writetimeout = 20 # (seconds)

# port for the media file server
serverport = 8080

# usually your protocol plus IP or domain plus the port, WITHOUT THE TRAILING SLASH
# don't forget to change http to https if enabled
baseurl = http://localhost:8080

## POMF
# the pomf clone url
# the following should work with irchuu:
# - https://p.fuwafuwa.moe
# - https://cocaine.ninja
# and many more. But some are retarded and won't.
pomf = https://p.fuwafuwa.moe

## KOMF
# a komf site url, you can set up your own: https://github.com/koto-bank/komf
komf =

# public url for your komf hosting, leave blank if it's same with above
# (needed if the domain used for static is different or if you're uploading files
# to a host in your local network)
komfpublicurl =

# how much time will the file be stored for? (day, week, month)
komfdate = week

[irc]
server = irc.rizon.net
port = 6667
ssl = false
serverpassword =

nick = irchuu

# if not blank, will use NickServ to identify
password =

# if true, will use SASL instead of NickServ
sasl = false

# must be surrounded with backticks
channel = ` + "`" + `#irchuu` + "`" + `
chanpassword =

# colorize nicknames? (based on djb2)
colorize = true

# colors to be used, either codes or names
palette = 1,2,3,4,5,6,9,10,11,12,13

# prefix and postfix will be added before and after nicks
prefix = <
postfix = >

# maximum username length allowed, will be ellipsised if longer (0 to disable)
maxlength = 18

# lines in multi-line messages will be divided with this
# leave blank to send them as separate messages
ellipsis = "… "

# delay with which parts of multi-line message are sent to prevent anti-flood from kicking the bot
flooddelay = 500 # (milliseconds)

# allow ops in IRC to kick users from Telegram
# (bot needs to be a moderator in Telegram, also needs a database to be configured)
moderation = true

# who can kick users from the Telegram group:
# 1 — everybody, 2 — voices, 3 — halfops, 4 — ops, 5 — protected/admins, 6 — the owner
kickpermission = 4

# allow sending stickers from IRC? (by id)
allowstickers = true

# how often to poll the server for the users list
namesupdateinterval = 600 # (seconds)

# maximum number of messages sent on 'hist'command in IRC, works only with dburi set
maxhist = 40

# will send NOTICEs for private messages (help, hist, user count, etc) instead of PRIVMSGs
sendnotices = true

# forward join and part messages to Telegram
relayjoinsparts = true

# forward mode messages to Telegram
relaymodes = true

# rejoin automatically when kicked
kickrejoin = true

# announce the current topic to Telegram on join
announcetopic = true
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

	Colorize      bool
	Palette       []string
	Prefix        string
	Postfix       string
	MaxLength     int
	Ellipsis      string
	FloodDelay    int
	AllowStickers bool

	Moderation          bool
	KickPermission      int
	MaxHist             int
	NamesUpdateInterval int
	SendNotices         bool
	RelayJoinsParts     bool
	RelayModes          bool
	KickRejoin          bool
	AnnounceTopic       bool

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
	Storage       string
	CertFilePath  string
	KeyFilePath   string
	ServerPort    uint16
	ReadTimeout   int
	WriteTimeout  int
	BaseURL       string
	DataDir       string
	Pomf          string
	Komf          string
	KomfPublicURL string
	KomfDate      string
}

// muDeiPt5mAI8Ue==
