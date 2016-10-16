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
func Log(msg relay.Message, dbURI string, logger *log.Logger) {
	db, err := sql.Open("postgres", dbURI)
	handleErrors(err, logger)

	extraString, err := json.Marshal(msg.Extra)
	if err != nil {
		logger.Printf("An error occurred while marshalling the extra data: %v.",
			err)
	}
	if msg.Source {
		// Telegram
	} else {
		// IRC
		_, err := db.Query("INSERT INTO"+
			" messages(date, source, nick, \"text\", extra)"+
			" VALUES($1, $2, $3, $4, $5)",
			msg.Date, 0, msg.Nick, msg.Text,
			extraString)
		handleErrors(err, logger)
	}
}

// Init creates tables needed for IRChuu and returns true on success.
func Init(dbURI string) bool {
	logger := log.New(os.Stdout, " DB ", log.LstdFlags)

	db, err := sql.Open("postgres", dbURI)
	if !handleErrors(err, logger) {
		return false
	}
	_, err = db.Query("CREATE TABLE IF NOT EXISTS tg_users" +
		" (id INT PRIMARY KEY NOT NULL, nick VARCHAR(32)," +
		" first_name TEXT, last_name TEXT);")
	if !handleErrors(err, logger) {
		return false
	}
	_, err = db.Query("CREATE TABLE IF NOT EXISTS messages" +
		" (id BIGSERIAL, date TIMESTAMP NOT NULL," +
		" source BOOLEAN, nick TEXT, \"text\" TEXT, from_id INT," +
		" msg_id INT, extra JSONB);")
	if !handleErrors(err, logger) {
		return false
	}
	log.Println("Successfully initialized DB")
	return true
}

// handleErrors logs the error and returns false if it is not nil. Otherwise
// returns true.
func handleErrors(err error, logger *log.Logger) bool {
	if err, ok := err.(*pq.Error); ok {
		logger.Println("Database error:", err.Code.Name())
		return false
	}
	return true
}
