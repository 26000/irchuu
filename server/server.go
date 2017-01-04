package mediaserver

import (
	"github.com/26000/irchuu/config"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func Serve(c *config.Telegram) {
	logger := log.New(os.Stdout, "SRV ", log.LstdFlags)
	s := &http.Server{
		Addr:           ":" + strconv.FormatUint(uint64(c.ServerPort), 10),
		Handler:        http.FileServer(http.Dir(c.DataDir)),
		ReadTimeout:    time.Duration(c.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(c.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
		ErrorLog:       logger,
	}

	if c.CertFilePath != "" && c.KeyFilePath != "" {
		logger.Println(s.ListenAndServeTLS(c.CertFilePath, c.KeyFilePath))
	} else {
		logger.Println(s.ListenAndServe())
	}
}
