package helpers

import (
	"archive/zip"
	"bytes"
	"cli-top/debug"
	types "cli-top/types"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/h2non/filetype"

	"github.com/PuerkitoBio/goquery"
)

func StrToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil && debug.Debug {
		fmt.Println("Error converting string to integer:", err)
	}
	return num
}

func GenerateCalendarImportLinks(icsURL string, calendarName string) {
	StopHeadlineForOutput()
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
		":", "_",
		"*", "_",
		"?", "_",
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
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)
	sanitized := replacer.Replace(name)

	var result strings.Builder
	for _, r := range sanitized {
		if r < 32 || r == 127 {
			continue
		}
		result.WriteRune(r)
	}

	trimmed := strings.TrimRight(result.String(), " .")

	if trimmed == "" {
		return "unnamed"
	}

	if len(trimmed) > 200 {
		trimmed = trimmed[:200]
	}

	return trimmed
}

func SaveFile(data []byte, filePath string) error {
	return os.WriteFile(filePath, data, 0644)
}
func FetchReqClient(client *http.Client, cookies types.Cookies, url string, referer string, formData string, method string, contentType string) ([]byte, http.Header, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	return FetchReqClientWithContext(ctx, client, cookies, url, referer, formData, method, contentType)
}

func FetchReqClientWithContext(ctx context.Context, client *http.Client, cookies types.Cookies, url string, referer string, formData string, method string, contentType string) ([]byte, http.Header, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = GetHTTPClient()
	}
	if err := ValidateVtopURL(url); err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(formData))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Apply default headers for VTOP requests
	SetVtopHeaders(req)

	req.Header.Set("Content-Type", contentType)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	if cookies.JSESSIONID != "" {
		req.AddCookie(&http.Cookie{
			Name:  "JSESSIONID",
			Value: cookies.JSESSIONID,
			Path:  "/",
		})
	}
	if cookies.SERVERID != "" {
		req.AddCookie(&http.Cookie{
			Name:  "SERVERID",
			Value: cookies.SERVERID,
			Path:  "/",
		})
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return body, resp.Header, fmt.Errorf("VTOP request failed with status %s", resp.Status)
	}

	return body, resp.Header, nil
}

func GetFileExtension(filename string, body []byte, headers http.Header) string {
	if ext := filepath.Ext(filename); ext != "" {
		return ext
	}

	if contentDisposition := headers.Get("Content-Disposition"); contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if ext := filepath.Ext(params["filename"]); ext != "" {
				return ext
			}
		}
	}

	if ext := ooxmlExtension(body); ext != "" {
		return ext
	}

	kind, err := filetypeMatch(body)
	if err == nil && kind != "unknown" {
		switch kind {
		case "doc", "xls", "ppt", "docx", "xlsx", "pptx", "pdf", "zip":
			return "." + kind
		}
	}

	switch http.DetectContentType(body) {
	case "application/msword":
		return ".doc"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.ms-powerpoint":
		return ".ppt"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	}

	if len(body) >= 8 && bytes.Equal(body[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
		if bytes.Contains(body, []byte("WordDocument")) {
			return ".doc"
		}
		if bytes.Contains(body, []byte("Workbook")) || bytes.Contains(body, []byte("Book")) {
			return ".xls"
		}
		if bytes.Contains(body, []byte("PowerPoint Document")) {
			return ".ppt"
		}
		return ".bin"
	}

	if len(body) >= 4 && string(body[:4]) == "PK\x03\x04" {
		return ".zip"
	}

	if debug.Debug {
		fmt.Println("Unable to determine file extension; using .bin")
	}
	return ".bin"
}

func ooxmlExtension(body []byte) string {
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return ""
	}
	for _, f := range zipReader.File {
		switch {
		case strings.HasPrefix(f.Name, "ppt/"):
			return ".pptx"
		case strings.HasPrefix(f.Name, "word/"):
			return ".docx"
		case strings.HasPrefix(f.Name, "xl/"):
			return ".xlsx"
		}
	}
	return ""
}

func RemoveDuplicates(ints []int) []int {
	uniqueMap := make(map[int]struct{})
	var unique []int
	for _, i := range ints {
		if _, exists := uniqueMap[i]; !exists {
			uniqueMap[i] = struct{}{}
			unique = append(unique, i)
		}
	}
	return unique
}

func filetypeMatch(body []byte) (string, error) {
	kind, err := filetype.Match(body)
	if err != nil {
		return "unknown", err
	}
	if kind == filetype.Unknown {
		return "unknown", nil
	}
	return kind.Extension, nil
}

func SanitizeString(input string) string {
	trimmed := strings.TrimSpace(input)

	fields := strings.Fields(trimmed)
	singleSpaced := strings.Join(fields, " ")

	var sanitizedBuilder strings.Builder
	for _, r := range singleSpaced {
		if unicode.IsPrint(r) {
			sanitizedBuilder.WriteRune(r)
		}
	}

	return sanitizedBuilder.String()
}

func ValidateCookies(cookies types.Cookies) bool {
	return cookies.CSRF != "" && cookies.JSESSIONID != "" && cookies.SERVERID != ""
}

func ExtractRowData(rowSelection *goquery.Selection) []string {
	row := []string{}
	rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
		text := StripAnsiCodes(strings.TrimSpace(cell.Text()))
		row = append(row, text)
	})
	return row
}

func HandleError(context string, err error) {
	if debug.Debug {
		fmt.Printf("Error %s: %v\n", context, err)
	} else {
		fmt.Println("An error occurred. Please try again.")
	}
}

// GetOrCreateDownloadDir creates and returns the path to a subdirectory in the CLI-TOP Downloads directory
func GetOrCreateDownloadDir(subDir string) (string, error) {
	baseDir := filepath.Join(GetDownloadsDir(), "CLI-TOP Downloads")

	// Create the base CLI-TOP Downloads directory if it doesn't exist
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create base directory: %w", err)
	}

	// If no subdirectory specified, return the base directory
	if subDir == "" {
		return baseDir, nil
	}

	// Create the specified subdirectory
	fullPath := filepath.Join(baseDir, subDir)
	if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create subdirectory %s: %w", subDir, err)
	}

	return fullPath, nil
}

// OpenFolder opens the folder containing the specified path using the system's default file manager
func OpenFolder(path string) {
	if ShouldMuteUI() {
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		if os.Getenv("WSL_DISTRO_NAME") != "" {
			if _, err := exec.LookPath("wslview"); err == nil {
				cmd = exec.Command("wslview", path)
			}
		}
		if cmd == nil {
			if _, err := exec.LookPath("xdg-open"); err == nil {
				cmd = exec.Command("xdg-open", path)
			} else if _, err := exec.LookPath("gio"); err == nil {
				cmd = exec.Command("gio", "open", path)
			} else {
				fmt.Println("Please open the folder manually:", path)
				return
			}
		}
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error opening folder: %v\n", err)
	}
}
