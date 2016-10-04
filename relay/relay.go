package relay

// NewRelay creates a new Relay.
func NewRelay() *Relay {
	teleCh := make(chan Message, 100)
	irCh := make(chan Message, 100)
	logCh := make(chan Message, 100)
	return &Relay{teleCh, irCh, logCh}
}

// Relay contains three channels: for IRC messages read by Telegram, for
// Telegram messages read by IRC and for both read by the logger.
type Relay struct {
	TeleCh chan Message
	IRCh   chan Message
	LogCh  chan Message
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
	// Forward: from; Pin: true, Edit: date
	Extra map[string]string
}
