package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cli-top/debug"
	types "cli-top/types"
	"github.com/spf13/viper"
)

const maxRetries = 3

const (
	versionTrackingMaxAttempts = 3
	versionTrackingBaseTimeout = 10 * time.Second
	versionTrackingTimeoutStep = 5 * time.Second
	versionTrackingRetryDelay  = 250 * time.Millisecond
)

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

// SendVersionTrackingData delivers telemetry about which command/version was executed.
// It retries transient network failures with short timeouts so CLI users are not blocked
// by slow analytics endpoints.
func SendVersionTrackingData(data types.VersionTrackingData) {
	if data.UUID == "" {
		return
	}

	payload, err := json.Marshal(data)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error marshaling version tracking data:", err)
		}
		return
	}

	client := GetHTTPClient()
	endpoint := CalendarServerURL + "/version-track"
	var lastErr error

	for attempt := 1; attempt <= versionTrackingMaxAttempts; attempt++ {
		timeout := versionTrackingBaseTimeout + time.Duration(attempt-1)*versionTrackingTimeoutStep
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if reqErr != nil {
			cancel()
			if debug.Debug {
				fmt.Println("Error creating version tracking request:", reqErr)
			}
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", data.UUID)

		resp, respErr := client.Do(req)
		if respErr != nil {
			lastErr = respErr
			cancel()
			if debug.Debug {
				fmt.Printf("Attempt %d/%d failed sending version tracking data: %v\n", attempt, versionTrackingMaxAttempts, respErr)
			}
		} else {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			cancel()

			if resp.StatusCode == http.StatusOK {
				if debug.Debug {
					fmt.Println("Version tracking data sent successfully.")
				}
				return
			}

			lastErr = fmt.Errorf("version tracking request failed with status %d", resp.StatusCode)
			if resp.StatusCode < http.StatusInternalServerError {
				break
			}
		}

		if attempt < versionTrackingMaxAttempts {
			time.Sleep(time.Duration(attempt) * versionTrackingRetryDelay)
		}
	}

	if lastErr != nil && debug.Debug {
		fmt.Println("Version tracking failed:", lastErr)
	}
}
