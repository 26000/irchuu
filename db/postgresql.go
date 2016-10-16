package irchuubase

import (
	"database/sql"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/26000/irchuu/relay"
	"log"
	"os"
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

// handleErrors logs the error and returns false if it is not nil. Otherwise
// returns true.
func handleErrors(err error, logger *log.Logger) bool {
	if err, ok := err.(*pq.Error); ok {
		logger.Printf("Database error: %v\n %v\n", err.Code.Name(), err)
		return false
	}
	return true
}
