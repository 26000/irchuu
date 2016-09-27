package config

import (
	"log"
	"os"
	"os/user"
	"path"
)

// GetPaths gets config file and data directory paths, creating directories and
// the config file if needed.
func GetPaths() (configFile string, dataDir string) {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Unable to get the user directory: %v", err)
	}

	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData == "" {
		dataDir = path.Join(usr.HomeDir, ".local/share/", "irchuu/")
	} else {
		dataDir = path.Join(xdgData, "irchuu/")
	}
	if xdgConfig == "" {
		configFile = path.Join(usr.HomeDir, ".config/", "irchuu.conf")
	} else {
		configFile = path.Join(xdgConfig, "irchuu.conf")
	}

	CreateDirOrExit(dataDir, os.FileMode(0700))
	CreateDirOrExit(path.Dir(configFile), os.FileMode(0700))
	return
}

// CreateDirOrExit creates the directory if it doesn't exist, exits if failed.
func CreateDirOrExit(dir string, mode os.FileMode) {
	_, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, mode)
		if err != nil {
			log.Fatalf("Failed to create directory: %v\n", dir)
		} else {
			log.Printf("Created directory: %v\n", dir)
		}
	}
}
