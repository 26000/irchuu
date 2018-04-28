package irchuubase

import (
	"database/sql"
	"encoding/json"
	"github.com/26000/irchuu/relay"
	_ "github.com/lib/pq"
	"log"
	"os"
	"time"
	"unicode/utf8"
)

// Log inserts a message into the DB.
func Log(msg relay.Message, db *sql.DB, logger *log.Logger) {
	extraString, err := json.Marshal(msg.Extra)
	if err != nil {
		logger.Printf("An error occurred while marshalling the extra data: %v.",
			err)
	}
	if !utf8.Valid([]byte(msg.Text)) || !utf8.Valid([]byte(extraString)) || !utf8.Valid([]byte(msg.FirstName)) || !utf8.Valid([]byte(msg.Nick)) || !utf8.Valid([]byte(msg.LastName)) {
		logger.Printf("Invalid Unicode byte sequence detected, "+
			"refusing to log: %v: '%v'\n", msg.FromID, msg.Text)
		return
	}
	if msg.Source {
		// Telegram
		rows1, err := db.Query("INSERT INTO"+
			" messages(date, source, \"text\", from_id, msg_id, extra)"+
			" VALUES($1, $2, $3, $4, $5, $6);",
			msg.Date, 1, msg.Text, msg.FromID, msg.ID,
			extraString)
		defer rows1.Close()
		handleErrors(err, logger)

		rows2, err := db.Query("INSERT INTO"+
			" tg_users(id, nick, first_name, last_name, last_active)"+
			" VALUES($1, NULLIF($2, ''), $3, $4, $5) ON CONFLICT (id) DO UPDATE"+
			" SET id = $1, nick = NULLIF($2, ''), first_name = $3,"+
			" last_name = $4, last_active = $5;",
			msg.FromID, msg.Nick, msg.FirstName, msg.LastName, msg.Date)
		defer rows2.Close()
		handleErrors(err, logger)
	} else {
		// IRC
		rows, err := db.Query("INSERT INTO"+
			" messages(date, source, nick, \"text\", extra)"+
			" VALUES($1, $2, $3, $4, $5);",
			msg.Date, 0, msg.Nick, msg.Text,
			extraString)
		defer rows.Close()
		handleErrors(err, logger)
	}
}

// Init creates tables needed for IRChuu and returns true on success.
func Init(dbURI string) *sql.DB {
	logger := log.New(os.Stdout, " DB ", log.LstdFlags)

	db, err := sql.Open("postgres", dbURI)
	if !handleErrors(err, logger) {
		return nil
	}
	rows1, err := db.Query("CREATE TABLE IF NOT EXISTS tg_users" +
		" (id INT PRIMARY KEY NOT NULL, nick VARCHAR(32)," +
		" first_name TEXT, last_name TEXT, last_active TIMESTAMP" +
		" WITH TIME ZONE);")
	if !handleErrors(err, logger) {
		return nil
	}
	defer rows1.Close()
	rows2, err := db.Query("CREATE TABLE IF NOT EXISTS messages" +
		" (id BIGSERIAL, date TIMESTAMP WITH TIME ZONE NOT NULL," +
		" source BOOLEAN, nick TEXT, \"text\" TEXT, from_id INT," +
		" msg_id INT, extra JSONB);")
	if !handleErrors(err, logger) {
		return nil
	}
	defer rows2.Close()
	logger.Println("Successfully initialized")
	return db
}

// GetMessages gets n last messages and returns them in a slice of relay.Message.
func GetMessages(db *sql.DB, n int) ([]relay.Message, error) {
	msgs := make([]relay.Message, 0, n)
	rows, err := db.Query(`SELECT date, source, coalesce(messages.nick,
tg_users.nick, ''), text, coalesce(msg_id, 0), coalesce(from_id, 0),
coalesce(first_name, ' '), coalesce(last_name, ' '), extra FROM messages
LEFT JOIN tg_users
ON tg_users.id = messages.from_id ORDER BY date DESC LIMIT $1;`, n)
	defer rows.Close()

	if err != nil {
		return msgs, err
	}
	for rows.Next() {
		var (
			date      time.Time
			source    bool
			nick      string
			text      string
			ID        int
			fromID    int
			firstName string
			lastName  string
			extras    []byte
			extra     map[string]string
		)
		err = rows.Scan(&date, &source, &nick, &text, &ID, &fromID, &firstName,
			&lastName, &extras)
		if err != nil {
			log.Println(err)
		}
		err = json.Unmarshal(extras, &extra)
		if err != nil {
			log.Println(err)
		}
		msgs = append(msgs, relay.Message{date, source, nick, text, ID,
			fromID, firstName, lastName, extra})
	}
	return msgs, nil
}

// handleErrors logs the error and returns false if it is not nil. Otherwise
// returns true.
func handleErrors(err error, logger *log.Logger) bool {
	if err != nil {
		logger.Printf("Error: %v\n", err)
		return false
	}
	return true
}

func FindUser(name string, db *sql.DB) (id int, foundName string, err error) {
	err = db.QueryRow("SELECT id, coalesce(nick, first_name || ' ' || last_name)"+
		"  FROM tg_users WHERE nick LIKE $1 || '%'"+
		" OR first_name || ' ' || last_name LIKE $1 || '%' "+
		" ORDER BY last_active DESC LIMIT 1", name).Scan(&id, &foundName)
	return
}
