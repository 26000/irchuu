package relay

// NewRelay creates a new Relay.
func NewRelay() *Relay {
	teleCh := make(chan Message, 100)
	irCh := make(chan Message, 100)
	logCh := make(chan Message, 100)
	teleSCh := make(chan ServiceMessage, 20)
	ircSCh := make(chan ServiceMessage, 20)
	return &Relay{teleCh, teleSCh, irCh, ircSCh, logCh}
}

// Relay contains three channels: for IRC messages read by Telegram, for
// Telegram messages read by IRC and for both read by the logger.
// ServiceCh's are for command messages.
type Relay struct {
	TeleCh        chan Message
	TeleServiceCh chan ServiceMessage
	IRCh          chan Message
	IRCServiceCh  chan ServiceMessage
	LogCh         chan Message
}

// Message represents a generic message which may be either from TG or IRC.
type Message struct {
	Date   int64  // UNIX time
	Source string // IRC or TG
	Nick   string // Nickname in both IRC and Telegram
	Text   string

	ID     int    // Message ID, Telegram only
	Name   string // Realname, Telegram only
	FromID int    // From user ID, Telegram only
	// In IRC: CTCP (ACTION), kick, topic
	// In Telegram: Media: Type, width x height, size and URL;
	// Forward: forward (nick), forwardDate, forwardUserID
	// Pin: pin (true)
	// Edit: edit (date)
	// Reply: reply (nick), replyId, replyUserID
	Extra map[string]string
}

// ServiceMessage represents a service message, which is not relayed.
type ServiceMessage struct {
	Command   string
	Arguments string
}
