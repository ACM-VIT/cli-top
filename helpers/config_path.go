package helpers

import (
	"os"
	"path/filepath"
	"sync"
)

const (
	configFileName = "cli-top-config.env"
	configDirName  = "cli-top"
)

var (
	configPathOnce   sync.Once
	cachedConfigPath string
)

// ConfigFilePath returns the path to the cli-top configuration file, checking the
// current working directory first and then falling back to the executable
// directory.
func ConfigFilePath() string {
	configPathOnce.Do(func() {
		if cwd, err := os.Getwd(); err == nil {
			candidate := filepath.Join(cwd, configFileName)
			if _, err := os.Stat(candidate); err == nil {
				cachedConfigPath = candidate
				return
			}
		}

		if userConfigDir, err := os.UserConfigDir(); err == nil {
			configDir := filepath.Join(userConfigDir, configDirName)
			if err := os.MkdirAll(configDir, 0o700); err == nil {
				cachedConfigPath = filepath.Join(configDir, configFileName)
				return
			}
		}

		if homeDir, err := os.UserHomeDir(); err == nil {
			cachedConfigPath = filepath.Join(homeDir, configFileName)
			return
		}

		if exePath, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exePath)
			cachedConfigPath = filepath.Join(exeDir, configFileName)
			return
		}

		cachedConfigPath = configFileName
	})
	return cachedConfigPath
}
