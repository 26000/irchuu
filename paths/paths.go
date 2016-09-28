package paths

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
)

// GetPaths gets config file and data directory paths.
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
	return
}

// MakePaths creates the configuration file and directories if needed.
func MakePaths(configFile string, dataDir string) error {
	err := CreateDir(dataDir, os.FileMode(0700))
	if err != nil {
		log.Printf("Failed to create directory: %v\n", dataDir)
		return err
	}
	err = CreateDir(path.Dir(configFile), os.FileMode(0700))
	if err != nil {
		log.Printf("Failed to create directory: %v\n", path.Dir(configFile))
		return err
	}

	_, err = os.Stat(configFile)
	if err != nil {
		err = PopulateConfig(configFile)
		if err == nil {
			log.Printf("New configuration file was populated. Edit %v and run `irchuu` again!\n", configFile)
			defer os.Exit(0)
		} else {
			log.Fatalf("Failed to populate config: %v\n", err)
		}
	}
	return nil
}

// CreateDir creates the directory if it doesn't exist.
func CreateDir(dir string, mode os.FileMode) error {
	_, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, mode)
		if err != nil {
			return err
		} else {
			log.Printf("Created directory: %v\n", dir)
		}
	}
	return nil
}

// PopulateConfig copies the sample config to <path>.
func PopulateConfig(file string) error {
	config := `# IRChuu configuration file. See https://github.com/26000/irchuu for help.
[telegram]
token = myToken
group = 7654321

[irc]
server = irc.rizon.net
port = 6777
channel = irchuu # without '#'!
password = # leave blank if not set
`
	return ioutil.WriteFile(file, []byte(config), os.FileMode(0600))
}
