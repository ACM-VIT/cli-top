package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cli-top/debug"
	types "cli-top/types"
	"github.com/spf13/viper"
)

const maxRetries = 3

func RegisterUUID(uuid string) error {
	data := types.RegisterData{UUID: uuid}
	jsonData, err := json.Marshal(data)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error marshaling registration data:", err)
		}
		return err
	}

	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest("POST", CalendarServerURL+"/register", bytes.NewBuffer(jsonData))
		if err != nil {
			if debug.Debug {
				fmt.Println("Error creating registration request:", err)
			}
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)

		if err != nil {
			if debug.Debug {
				fmt.Printf("Attempt %d: Error sending registration request: %v\n", i+1, err)
			}
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusCreated {
			if debug.Debug {
				fmt.Println("UUID registered successfully.")
			}
			viper.Set("UUID", uuid)
			viper.Set("UNREGISTERED_UUID", "")
			if err := viper.WriteConfig(); err != nil && debug.Debug {
				fmt.Println("Error updating config after registration:", err)
			}
			return nil
		}

		if resp.StatusCode == http.StatusConflict {
			if debug.Debug {
				fmt.Println("UUID already registered.")
			}
			viper.Set("UUID", uuid)
			viper.Set("UNREGISTERED_UUID", "")
			if err := viper.WriteConfig(); err != nil && debug.Debug {
				fmt.Println("Error updating config after conflict:", err)
			}
			return nil
		}

		if debug.Debug {
			fmt.Println("Unexpected response status during registration:", resp.Status)
		}
	}

	if debug.Debug {
		fmt.Println("Registration failed after retries. UUID remains unregistered.")
	}
	return fmt.Errorf("failed to register UUID after %d attempts", maxRetries)
}
