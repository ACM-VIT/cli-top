package helpers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"bytes"
	"mime/multipart"
)

type ICSEvent struct {
	UID         string
	DtStamp     string
	DtStart     string
	DtEnd       string
	Summary     string
	Description string
}

func GenerateICSFile(events []ICSEvent, filePath string) error {
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
		"X-WR-CALNAME:CLI-TOP Events",
		"BEGIN:VTIMEZONE",
		"TZID:Asia/Kolkata",
		"BEGIN:STANDARD",
		"DTSTART:19700101T000000",
		"TZOFFSETFROM:+0530",
		"TZOFFSETTO:+0530",
		"TZNAME:IST",
		"END:STANDARD",
		"END:VTIMEZONE",
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
			fmt.Sprintf("DTSTART;TZID=Asia/Kolkata:%s", event.DtStart),
			fmt.Sprintf("DTEND;TZID=Asia/Kolkata:%s", event.DtEnd),
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
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(homeDir, "Downloads")
	default:
		return filepath.Join(homeDir, "Downloads")
	}
}

// UploadICSFile uploads the ICS file to the specified server and returns the uploaded file URL.
func UploadICSFile(filePath string, serverURL string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open ICS file: %v", err)
	}
	defer file.Close()

	// Prepare a buffer to hold the multipart form data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Create the file field in the multipart form data
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}

	// Copy the file content to the multipart field
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file content: %v", err)
	}

	// Close the multipart writer to finalize the form data
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %v", err)
	}

	// Create a new POST request with the multipart form data
	req, err := http.NewRequest("POST", serverURL+"/upload", &body)
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %v", err)
	}

	// Set the content type to multipart/form-data
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload ICS file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status: %s", resp.Status)
	}

	// Read the server's response (URL of the uploaded file)
	uploadedURL, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read upload response: %v", err)
	}

	return string(uploadedURL), nil
}