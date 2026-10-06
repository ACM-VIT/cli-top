package helpers

import (
	"os"
	"sync"

	"github.com/lpernett/godotenv"
	"github.com/spf13/viper"
)

var configWriteMu sync.Mutex

func SaveConfigValues(values map[string]string) error {
	configWriteMu.Lock()
	defer configWriteMu.Unlock()
	for key, value := range values {
		viper.Set(key, value)
	}
	return viper.WriteConfigAs(ConfigFilePath())
}

func writeEnvValue(key, value string) error {
	configWriteMu.Lock()
	defer configWriteMu.Unlock()
	viper.Set(key, "\""+value+"\"")
	configPath := ConfigFilePath()
	values, err := godotenv.Read(configPath)
	if err != nil {
		values = make(map[string]string)
	}
	values[key] = value
	if err := godotenv.Write(values, configPath); err != nil {
		return err
	}
	return os.Chmod(configPath, 0o600)
}
