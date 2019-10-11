package cfg

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/olebedev/config"
)

const (
	// XdgConfigDir defines the path to the minimal XDG-compatible configuration directory
	XdgConfigDir = "~/.config/"

	// WtfConfigDirV1 defines the path to the first version of configuration. Do not use this
	WtfConfigDirV1 = "~/.wtf/"

	// WtfConfigDirV2 defines the path to the second version of the configuration. Use this.
	WtfConfigDirV2 = "~/.config/wtf/"

	// WtfConfigFile defines the name of the default config file
	WtfConfigFile = "config.yml"

	// WtfSecretsFile defines the file in which to store API Keys and other values you may want to keep out of config.yml
	WtfSecretsFile = "secrets.yml"
)

/* -------------------- Exported Functions -------------------- */

// CreateFile creates the named file in the config directory, if it does not already exist.
// If the file exists it does not recreate it.
// If successful, eturns the absolute path to the file
// If unsuccessful, returns an error
func CreateFile(fileName string) (string, error) {
	configDir, err := WtfConfigDir()
	if err != nil {
		return "", err
	}

	filePath := fmt.Sprintf("%s/%s", configDir, fileName)

	// Check if the file already exists; if it does not, create it
	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			_, err = os.Create(filePath)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	return filePath, nil
}

// Initialize takes care of settings up the initial state of WTF configuration
// It ensures necessary directories and files exist
func Initialize(hasCustom bool) {
	if hasCustom == false {
		migrateOldConfig()
	}

	// These always get created because this is where modules should write any permanent
	// data they need to persist between runs (i.e.: log, textfile, etc.)
	createXdgConfigDir()
	createWtfConfigDir()

	if hasCustom == false {
		createWtfConfigFile()
		chmodConfigFile()
		chmodSecretsFile()
	}
}

// WtfConfigDir returns the absolute path to the configuration directory
func WtfConfigDir() (string, error) {
	configDir, err := expandHomeDir(WtfConfigDirV2)
	if err != nil {
		return "", err
	}

	return configDir, nil
}

// LoadWtfConfigFile loads the specified config file
func LoadWtfConfigFile(filePath string) *config.Config {
	absPath, _ := expandHomeDir(filePath)

	cfg, err := config.ParseYamlFile(absPath)
	if err != nil {
		displayWtfConfigFileLoadError(absPath, err)
		os.Exit(1)
	}

	return cfg
}

// LoadWtfSecretsFile loads the specified secrets file
func LoadWtfSecretsFile(filePath string) *config.Config {
	absPath, _ := expandHomeDir(filePath)

	secrets, err := config.ParseYamlFile(absPath)
	if err != nil {
		displayWtfConfigFileLoadError(absPath, err)
		os.Exit(1)
	}

	return secrets
}

/* -------------------- Unexported Functions -------------------- */

// chmodConfigFile sets the mode of the config file to r+w for the owner only
func chmodConfigFile() {
	relPath := fmt.Sprintf("%s%s", WtfConfigDirV2, WtfConfigFile)
	absPath, _ := expandHomeDir(relPath)

	_, err := os.Stat(absPath)
	if err != nil && os.IsNotExist(err) {
		return
	}

	err = os.Chmod(absPath, 0600)
	if err != nil {
		return
	}
}

// chmodSecretsFile sets the mode of the Secrets file to r+w for the owner only
func chmodSecretsFile() {
	relPath := fmt.Sprintf("%s%s", WtfConfigDirV2, WtfSecretsFile)
	absPath, _ := expandHomeDir(relPath)

	_, err := os.Stat(absPath)
	if err != nil && os.IsNotExist(err) {
		return
	}

	err = os.Chmod(absPath, 0600)
	if err != nil {
		return
	}
}

// createXdgConfigDir creates the necessary base directory for storing the config file
// If ~/.config is missing, it will try to create it
func createXdgConfigDir() {
	xdgConfigDir, _ := expandHomeDir(XdgConfigDir)

	if _, err := os.Stat(xdgConfigDir); os.IsNotExist(err) {
		err := os.Mkdir(xdgConfigDir, os.ModePerm)
		if err != nil {
			displayXdgConfigDirCreateError(err)
			os.Exit(1)
		}
	}
}

// createWtfConfigDir creates the necessary directories for storing the default config file
// If ~/.config/wtf is missing, it will try to create it
func createWtfConfigDir() {
	wtfConfigDir, _ := WtfConfigDir()

	if _, err := os.Stat(wtfConfigDir); os.IsNotExist(err) {
		err := os.Mkdir(wtfConfigDir, os.ModePerm)
		if err != nil {
			displayWtfConfigDirCreateError(err)
			os.Exit(1)
		}
	}
}

// createWtfConfigFile creates a simple config file in the config directory if
// one does not already exist
func createWtfConfigFile() {
	filePath, err := CreateFile(WtfConfigFile)
	if err != nil {
		displayDefaultConfigCreateError(err)
		os.Exit(1)
	}

	// If the file is empty, write to it
	file, _ := os.Stat(filePath)

	if file.Size() == 0 {
		if ioutil.WriteFile(filePath, []byte(defaultConfigFile), 0600) != nil {
			displayDefaultConfigWriteError(err)
			os.Exit(1)
		}
	}
}

// createWtfSecretsFile creates a simple config file in the config directory if
// one does not already exist
func createWtfSecretsFile() {
	filePath, err := CreateFile(WtfSecretsFile)
	if err != nil {
		displayDefaultConfigCreateError(err)
		os.Exit(1)
	}

	// If the file is empty, write to it
	file, _ := os.Stat(filePath)

	if file.Size() == 0 {
		if ioutil.WriteFile(filePath, []byte(defaultSecretsFile), 0600) != nil {
			displayDefaultConfigWriteError(err)
			os.Exit(1)
		}
	}
}

// Expand expands the path to include the home directory if the path
// is prefixed with `~`. If it isn't prefixed with `~`, the path is
// returned as-is.
func expandHomeDir(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", errors.New("cannot expand user-specific home dir")
	}

	dir, err := home()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, path[1:]), nil
}

// Dir returns the home directory for the executing user.
// An error is returned if a home directory cannot be detected.
func home() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	if currentUser.HomeDir == "" {
		return "", errors.New("cannot find user-specific home dir")
	}

	return currentUser.HomeDir, nil
}

// migrateOldConfig copies any existing configuration from the old location
// to the new, XDG-compatible location
func migrateOldConfig() {
	srcDir, _ := expandHomeDir(WtfConfigDirV1)
	destDir, _ := expandHomeDir(WtfConfigDirV2)

	// If the old config directory doesn't exist, do not move
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return
	}

	// If the new config directory already exists, do not move
	if _, err := os.Stat(destDir); err == nil {
		return
	}

	// Time to move
	err := Copy(srcDir, destDir)
	if err != nil {
		panic(err)
	}

	// Delete the old directory if the new one exists
	if _, err := os.Stat(destDir); err == nil {
		err := os.RemoveAll(srcDir)
		if err != nil {
			fmt.Println(err)
		}
	}
}
