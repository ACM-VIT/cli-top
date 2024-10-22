package helpers

import (
	"bytes"
	"cli-top/debug"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func GetTextContent(n *html.Node) string {
	var textContent string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			textContent += c.Data
		} else if c.Type == html.ElementNode {
			textContent += GetTextContent(c)
		}
	}
	return textContent
}

func StrToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil && debug.Debug {
		fmt.Println("Error converting string to integer:", err)
	}
	return num
}

func FindOptionWithTagValue(doc *goquery.Document, targetValue string) string {
	return doc.Find("option[value='" + targetValue + "']").Text()
}

func RemoveEmptyStrings(data []string) []string {
	var cleanedData []string
	for _, item := range data {
		if item != "" {
			cleanedData = append(cleanedData, item)
		}
	}
	return cleanedData
}

func GenerateCalendarImportLinks(icsURL string, calendarName string) {
	fmt.Println("Import events into your calendar using the links below:")
	fmt.Println()
	blueColor := "\033[34m"
	resetColor := "\033[0m"
	googleLink := GenerateGoogleCalendarImportLink(icsURL)
	googleLinkText := blueColor + "Add to Google Calendar" + resetColor
	fmt.Println(MakeANSILink(googleLinkText, googleLink))
	outlookLink := GenerateOutlookCalendarImportLink(icsURL, calendarName)
	outlookLinkText := blueColor + "Add to Outlook Calendar" + resetColor
	fmt.Println(MakeANSILink(outlookLinkText, outlookLink))
}

func GenerateGoogleCalendarImportLink(icsURL string) string {
	baseURL := "https://calendar.google.com/calendar/r?cid="
	encodedURL := url.QueryEscape(icsURL)
	return fmt.Sprintf("%s%s", baseURL, encodedURL)
}

func GenerateOutlookCalendarImportLink(icsURL string, calendarName string) string {
	baseURL := "https://outlook.live.com/owa/?path=/calendar/action/subscribe&url=%s&name=%s"
	encodedURL := url.QueryEscape(icsURL)
	calendarNameEncoded := url.QueryEscape(calendarName)
	return fmt.Sprintf(baseURL, encodedURL, calendarNameEncoded)
}

func MakeANSILink(text, url string) string {
	return fmt.Sprintf("\u001B]8;;%s\a%s\u001B]8;;\a", url, text)
}

func UploadICSFile(filename, serverURL string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", serverURL+"/upload", &requestBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned non-OK status: %s", resp.Status)
	}
	var respData struct {
		URL string `json:"url"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return "", err
	}
	return respData.URL, nil
}

func EscapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func TruncateWithEllipsis(s string, maxLength int) string {
	runes := []rune(s)
	if len(runes) <= maxLength {
		return s
	}
	if maxLength <= 3 {
		return string(runes[:maxLength])
	}
	return string(runes[:maxLength-3]) + "..."
}

func StripAnsiCodes(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

func GenerateUID(data string) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
