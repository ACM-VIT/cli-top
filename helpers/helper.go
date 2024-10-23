package helpers

import (
	"cli-top/debug"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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

func ReverseSlice[T any](slice []T) {
    for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
        slice[i], slice[j] = slice[j], slice[i]
    }
}

const (
	Red   = "\033[31m"
	Green = "\033[32m"
	Reset = "\033[0m"
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

func EscapeString(text string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		";", "\\;",
		",", "\\,",
		"\n", "\\n",
	)
	return replacer.Replace(text)
}