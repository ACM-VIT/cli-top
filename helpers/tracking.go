package helpers

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "cli-top/debug"
)

type RegisterData struct {
    UUID string `json:"uuid"`
}

func RegisterUUID(uuid string) error {
    data := RegisterData{
        UUID: uuid,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        if debug.Debug {
            fmt.Println("Error marshaling registration data:", err)
        }
        return err
    }

    req, err := http.NewRequest("POST", "https://cli-calendar.acmvit.in/track", bytes.NewBuffer(jsonData))
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
            fmt.Println("Error sending registration request:", err)
        }
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusCreated {
        if debug.Debug {
            fmt.Println("UUID registered successfully.")
        }
    } else if resp.StatusCode == http.StatusConflict {
        if debug.Debug {
            fmt.Println("UUID already registered.")
        }
    } else {
        if debug.Debug {
            fmt.Println("Unexpected response status during registration:", resp.Status)
        }
    }

    return nil
}
