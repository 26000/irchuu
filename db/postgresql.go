package irchuubase

import (
	"database/sql"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/26000/irchuu/relay"
	"log"
	"os"
	"time"
)

// Log inserts a message into the DB.
func Log(msg relay.Message, db *sql.DB, logger *log.Logger) {
	extraString, err := json.Marshal(msg.Extra)
	if err != nil {
		logger.Printf("An error occurred while marshalling the extra data: %v.",
			err)
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
			" VALUES($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE"+
			" SET id = $1, nick = $2, first_name = $3,"+
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
	defer rows1.Close()
	if !handleErrors(err, logger) {
		return nil
	}
	rows2, err := db.Query("CREATE TABLE IF NOT EXISTS messages" +
		" (id BIGSERIAL, date TIMESTAMP WITH TIME ZONE NOT NULL," +
		" source BOOLEAN, nick TEXT, \"text\" TEXT, from_id INT," +
		" msg_id INT, extra JSONB);")
	defer rows2.Close()
	if !handleErrors(err, logger) {
		return nil
	}
	log.Println("Successfully initialized DB")
	return db
}

// GetMessages gets n last messages and returns them in a slice of relay.Message.
func GetMessages(db *sql.DB, n int) ([]relay.Message, error) {
	msgs := make([]relay.Message, n)
	rows, err := db.Query(`SELECT date, source, coalesce(messages.nick,
tg_users.nick), text, msg_id, from_id, first_name, last_name, extra FROM messages
LEFT JOIN tg_users
ON tg_users.id = messages.from_id ORDER BY date DESC LIMIT $1;`, n)
	defer rows.Close()

	if err != nil {
		return msgs, err
	}
	i := 0
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
		rows.Scan(&date, &source, &nick, &text, &ID, &fromID, &firstName,
			&lastName, &extras)
		err = json.Unmarshal(extras, extra)
		msgs[i] = relay.Message{date, source, nick, text, ID,
			fromID, firstName, lastName, extra}
		i++
	}
	return msgs, nil
}

// handleErrors logs the error and returns false if it is not nil. Otherwise
// returns true.
func handleErrors(err error, logger *log.Logger) bool {
	if err, ok := err.(*pq.Error); ok {
		logger.Printf("Database error: %v\n %v\n", err.Code.Name(), err)
		return false
	}
	return true
}

func FindUser(name string, db *sql.DB) (id int, err error) {
	err = db.QueryRow("SELECT id FROM tg_users WHERE nick LIKE $1 || '%'"+
		" ORDER BY last_active DESC LIMIT 1", name).Scan(&id)
	return
}
