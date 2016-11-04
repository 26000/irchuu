package relay

import "time"

// NewRelay creates a new Relay.
func NewRelay() *Relay {
	teleCh := make(chan Message, 100)
	irCh := make(chan Message, 100)
	teleSCh := make(chan ServiceMessage, 20)
	ircSCh := make(chan ServiceMessage, 20)
	return &Relay{teleCh, teleSCh, irCh, ircSCh}
}

// Relay contains three channels: for IRC messages read by Telegram, for
// Telegram messages read by IRC and for both read by the logger.
// ServiceCh's are for command messages.
type Relay struct {
	TeleCh        chan Message
	TeleServiceCh chan ServiceMessage
	IRCh          chan Message
	IRCServiceCh  chan ServiceMessage
}

// Message represents a generic message which may be either from TG or IRC.
type Message struct {
	Date   time.Time // Time
	Source bool      // IRC (false) or TG (true)
	Nick   string    // Nickname in both IRC and Telegram
	Text   string

	ID        int    // Message ID, Telegram only
	FromID    int    // From user ID, Telegram only
	FirstName string // Realname, Telegram only
	LastName  string // Realname, Telegram only

	// In IRC: CTCP (ACTION), kick, topic
	// In Telegram: medias, replies, forwards, pins, edits, new/left members...
	Extra map[string]string
}

// ServiceMessage represents a service message, which is not relayed.
type ServiceMessage struct {
	Command   string
	Arguments []string
}

// Name returns string representation of the sender.
func (message *Message) Name() (nick string) {
	if message.Nick == "" {
		nick = message.FirstName
		if message.LastName != "" {
			nick += " " + message.LastName
		}
	} else {
		nick = message.Nick
	}
	return
}
