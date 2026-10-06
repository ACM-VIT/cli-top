package helpers

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/lpernett/godotenv"
	"github.com/spf13/viper"
)

func useTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), configFileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	configPathOnce.Do(func() {})
	previousPath := cachedConfigPath
	cachedConfigPath = path
	viper.Reset()
	viper.SetConfigType("env")
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("read config: %v", err)
	}
	t.Cleanup(func() {
		cachedConfigPath = previousPath
		viper.Reset()
	})
	t.Setenv(autoSelectSemesterEnvKey, "")
	return path
}

func TestAutoSelectSettingSurvivesLaterViperWrite(t *testing.T) {
	path := useTempConfig(t, "UNREGISTERED_UUID=pending\n")

	if err := SetAutoSelectSemester(false); err != nil {
		t.Fatalf("SetAutoSelectSemester() error = %v", err)
	}
	if err := SaveConfigValues(map[string]string{"UUID": "registered", "UNREGISTERED_UUID": ""}); err != nil {
		t.Fatalf("SaveConfigValues() error = %v", err)
	}

	values, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if values[autoSelectSemesterEnvKey] != "false" {
		t.Fatalf("%s = %q after later write, want %q", autoSelectSemesterEnvKey, values[autoSelectSemesterEnvKey], "false")
	}
	if values["UUID"] != "registered" {
		t.Fatalf("UUID = %q, want %q", values["UUID"], "registered")
	}
}

func TestConcurrentConfigWritesKeepAllValues(t *testing.T) {
	path := useTempConfig(t, "UNREGISTERED_UUID=pending\n")

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := SaveConfigValues(map[string]string{"UUID": "registered"}); err != nil {
			t.Errorf("SaveConfigValues() error = %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := SetAutoSelectSemester(false); err != nil {
			t.Errorf("SetAutoSelectSemester() error = %v", err)
		}
	}()
	wg.Wait()

	values, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if values["UUID"] != "registered" || values[autoSelectSemesterEnvKey] != "false" {
		t.Fatalf("config = %#v, want both UUID and %s preserved", values, autoSelectSemesterEnvKey)
	}
}
