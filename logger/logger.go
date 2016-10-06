package logger

import (
	"encoding/json"
	"github.com/nibogd/irchuu/relay"
	"log"
	"os"
	"sync"
)

// Launch starts logging.
func Launch(file string, wg *sync.WaitGroup, r *relay.Relay) {
	defer wg.Done()

	logger := log.New(os.Stdout, "LOG ", log.LstdFlags)
	_, err := os.Stat(file)
	if err != nil {
		f, err := os.Create(file)
		if err != nil {
			logger.Println("Failed to create the log file. Your chats won't be logged")
			return
		} else {
			f.Close()
		}
	}

	for m := range r.LogCh {
		json, err := json.Marshal(m)
		if err != nil {
			logger.Println(err)
		} else {
			if err := appendBytes(file, json); err != nil {
				logger.Println(err)
			}
		}
	}
}

// appendBytes adds bytes to the file.
func appendBytes(file string, bytes []byte) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	bytes = append(bytes, byte(10))

	defer f.Close()

	if _, err = f.Write(bytes); err != nil {
		return err
	}

	return nil
}
