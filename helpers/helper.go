package helpers

import (
	"cli-top/debug"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"os"

	"github.com/PuerkitoBio/goquery"
	//"golang.org/x/net/html"
)

// func GetTextContent(n *html.Node) string {
// 	var textContent string
// 	for c := n.FirstChild; c != nil; c = c.NextSibling {
// 		if c.Type == html.TextNode {
// 			textContent += c.Data
// 		} else if c.Type == html.ElementNode {
// 			textContent += GetTextContent(c)
// 		}
// 	}
// 	return textContent
// }

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

type UploadResponse struct {
	URL string `json:"url"`
}

func GenerateCalendarImportLinks(icsURL string, calendarName string) {
	fmt.Println("Import into your calendar using the links below:")
	fmt.Println()
	blueColor := "\033[34m"
	resetColor := "\033[0m"
	googleLink := GenerateGoogleCalendarLink(icsURL)
	googleLinkText := blueColor + "Add to Google Calendar" + resetColor
	fmt.Println(MakeANSILink(googleLinkText, googleLink))
	outlookLink := GenerateOutlookCalendarImportLink(icsURL, calendarName)
	outlookLinkText := blueColor + "Add to Outlook Calendar" + resetColor
	fmt.Println(MakeANSILink(outlookLinkText, outlookLink))
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

func GenerateGoogleCalendarLink(icsURL string) string {
	baseURL := "https://calendar.google.com/calendar/r?cid="
	return fmt.Sprintf("%s%s", baseURL, icsURL)
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

func ReverseSlice[T any](slice []T) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Reset  = "\033[0m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
)

func FormatDate(dateStr string) string {
	parsedTime, err := time.Parse("02-Jan-2006 15:04", dateStr)
	if err != nil {
		parsedTime, err = time.Parse("02-Jan-2006", dateStr)
		if err != nil {
			return dateStr
		}
	}
	return parsedTime.Format("02/01/06")
}

func ColorStatus(status string) string {
	upperStatus := strings.ToUpper(status)
	if strings.Contains(upperStatus, "PENDING") {
		return Red + status + Reset
	} else if strings.Contains(upperStatus, "APPROVED") {
		return Green + status + Reset
	}
	return status
}

func TruncateWithEllipses(text string, maxLength int) string {
	if len(text) > maxLength {
		return text[:maxLength-3] + "..."
	}
	return text
}

func AddLeftPadding(text string, padding int) string {
	paddingString := strings.Repeat(" ", padding)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = paddingString + line
	}
	return strings.Join(lines, "\n")
}

func EscapeString(str string) string {
	str = strings.ReplaceAll(str, "\\", "\\\\")
	str = strings.ReplaceAll(str, ";", "\\;")
	str = strings.ReplaceAll(str, ",", "\\,")
	str = strings.ReplaceAll(str, "\n", "\\n")
	return str
}

func FormatDateTime(dateStr string) string {
	formats := []string{
		"02-Jan-2006 03:04 PM", 
		"02-Jan-2006 15:04",   
		"02-Jan-2006",          
		"02/01/2006",          
		"02/01/06",             
	}

	var parsedTime time.Time
	var err error

	for _, format := range formats {
		parsedTime, err = time.Parse(format, dateStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return dateStr
	}

	return parsedTime.Format("02/01/06 15:04")
}

func SanitizeFilename(name string) string {
    replacer := strings.NewReplacer(
        "/", "_",
        "\\", "_",
        ":", "",
        "*", "_",
        "?", "",
        "\"", "_",
        "<", "_",
        ">", "_",
        "|", "_",
        "\u2013", "-",
        "\u2014", "-",
        "\u2018", "'",
        "\u2019", "'",
        "\u201C", "\"",
        "\u201D", "\"",
    )
    return replacer.Replace(name)
}

func SaveFile(data []byte, filePath string) error {
    return os.WriteFile(filePath, data, 0644)
}