package main

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
)

const VERSION = "0.0.0"
const CONFIG = "irchuu.conf"

func main() {
	/*
		if len(os.Args) < 2 {
			fmt.Println("Usage: irchuu <path/to/config>\n       A new config file will be created if no file at <path>.")
			os.Exit(1)
		}
	*/

	fmt.Printf("IRChuu! v%v (https://github.com/26000/irchuu)\n", VERSION)

	viper.SetConfigName(CONFIG)
	viper.AddConfigPath(".")
	if os.Getenv("$XDG_CONFIG_HOME") != "" {
		viper.AddConfigPath("$XDG_CONFIG_HOME")
	}
	viper.AddConfigPath("$HOME/.config/")
	viper.AddConfigPath("/etc/")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Errorf("Error: %s\n", err)
	}
	log.Printf("Using configuration file: %v\n", viper.ConfigFileUsed())
}
