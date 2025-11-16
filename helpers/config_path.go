package helpers

import (
    "os"
    "path/filepath"
    "sync"
)

const configFileName = "cli-top-config.env"

var (
    configPathOnce sync.Once
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

        if exePath, err := os.Executable(); err == nil {
            exeDir := filepath.Dir(exePath)
            cachedConfigPath = filepath.Join(exeDir, configFileName)
            return
        }

        cachedConfigPath = configFileName
    })
    return cachedConfigPath
}
