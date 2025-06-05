package helpers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cli-top/types"
)

func GenerateICSFileDateOnly(events []types.ICSEvent, filePath string, calName string) error {
	if len(events) == 0 {
		return nil
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create ICS file: %v", err)
	}
	defer file.Close()

	icsHeaders := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//CLI-TOP//EN",
		fmt.Sprintf("X-WR-CALNAME:%s", EscapeString(calName)),
	}
	_, err = file.WriteString(strings.Join(icsHeaders, "\r\n") + "\r\n")
	if err != nil {
		return fmt.Errorf("failed to write ICS headers: %v", err)
	}

	for _, event := range events {
		vevent := []string{
			"BEGIN:VEVENT",
			fmt.Sprintf("UID:%s", event.UID),
			fmt.Sprintf("DTSTAMP:%s", event.DtStamp),
			fmt.Sprintf("DTSTART;VALUE=DATE:%s", event.DtStart),
			fmt.Sprintf("DTEND;VALUE=DATE:%s", event.DtEnd),
			fmt.Sprintf("SUMMARY:%s", EscapeString(event.Summary)),
			fmt.Sprintf("DESCRIPTION:%s", EscapeString(event.Description)),
			"END:VEVENT",
		}

		_, err = file.WriteString(strings.Join(vevent, "\r\n") + "\r\n")
		if err != nil {
			return fmt.Errorf("failed to write VEVENT: %v", err)
		}
	}

	_, err = file.WriteString("END:VCALENDAR\r\n")
	if err != nil {
		return fmt.Errorf("failed to write ICS footer: %v", err)
	}

	return nil
}

func GenerateUID(prefix string) string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(bytes))
}

func GetDownloadsDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	switch strings.ToLower(os.Getenv("GOOS")) {
	case "windows":
		return filepath.Join(homeDir, "Downloads")
	default:
		return filepath.Join(homeDir, "Downloads")
	}
}

func UploadICSFile(filePath string, serverURL string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open ICS file: %v", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	uploadURL := serverURL + "/upload"

	req, err := http.NewRequest("POST", uploadURL, &requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to create POST request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload ICS file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status: %s", resp.Status)
	}

	var uploadResponse struct {
		URL string `json:"url"`
	}
	err = json.NewDecoder(resp.Body).Decode(&uploadResponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse upload response: %v", err)
	}

	return uploadResponse.URL, nil
}
func VenueAdd(events []types.ICSWithLocation, filePath string, calName string) error {
	if len(events) == 0 {
		return nil
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create ICS file: %v", err)
	}
	defer file.Close()

	icsHeaders := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//CLI-TOP//EN",
		fmt.Sprintf("X-WR-CALNAME:%s", EscapeString(calName)),
	}
	_, err = file.WriteString(strings.Join(icsHeaders, "\r\n") + "\r\n")
	if err != nil {
		return fmt.Errorf("failed to write ICS headers: %v", err)
	}

	for _, event := range events {
		vevent := []string{
			"BEGIN:VEVENT",
			fmt.Sprintf("DTSTART;%s", event.Event.DtStart),
			fmt.Sprintf("DTEND;%s", event.Event.DtEnd),
			fmt.Sprintf("SUMMARY:%s", EscapeString(event.Event.Summary)),
			fmt.Sprintf("DESCRIPTION:%s", EscapeString(event.Event.Description)),
			fmt.Sprintf("DTSTAMP:%s", time.Now().UTC().Format("20060102T150405Z")),
			"END:VEVENT",
		}

		_, err = file.WriteString(strings.Join(vevent, "\r\n") + "\r\n")
		if err != nil {
			return fmt.Errorf("failed to write VEVENT: %v", err)
		}
	}

	_, err = file.WriteString("END:VCALENDAR\r\n")
	if err != nil {
		return fmt.Errorf("failed to write ICS footer: %v", err)
	}

	return nil
}
