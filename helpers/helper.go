package helpers

import (
	"cli-top/debug"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"os"
	"bytes"
    "io/ioutil"
    "net/http"
    "cli-top/types" 
	"path/filepath"
	"archive/zip"
    "github.com/h2non/filetype"

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

func FetchReqClient(client *http.Client, regNo string, cookies types.Cookies, url string, referer string, formData string, method string, contentType string) ([]byte, error) {
    req, err := http.NewRequest(method, url, bytes.NewBufferString(formData))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", contentType)
    if referer != "" {
        req.Header.Set("Referer", referer)
    }
    req.Header.Set("Cookie", buildCookieHeader(cookies))
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; YourApp/1.0)") 

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
    }

    return body, nil
}

func buildCookieHeader(cookies types.Cookies) string {
    return fmt.Sprintf("JSESSIONID=%s; SERVERID=%s;", cookies.JSESSIONID, cookies.SERVERID)
}

func GetFileExtension(filename string, body []byte) string {
	// 1. Check if the filename already has an extension
	ext := filepath.Ext(filename)
	if ext != "" {
		fmt.Printf("Existing extension found: %s\n", ext)
		return ext
	}

	// 2. Use the filetype package to detect the file type
	kind, err := filetype.Match(body)
	if err == nil && kind != filetype.Unknown {
		fmt.Printf("Filetype package detected: %s\n", kind.Extension)
		switch kind.Extension {
		case "doc":
			return ".doc"
		case "xls":
			return ".xls"
		case "ppt":
			return ".ppt"
		case "docx":
			return ".docx"
		case "xlsx":
			return ".xlsx"
		case "pptx":
			return ".pptx"
		case "pdf":
			return ".pdf"
		// Add more cases as needed
		default:
			fmt.Printf("Filetype package detected unknown type: %s\n", kind.Extension)
		}
	} else {
		fmt.Println("Filetype package could not determine the file type.")
	}

	// 3. Fallback to MIME type detection
	mimeType := http.DetectContentType(body)
	fmt.Printf("MIME type detected: %s\n", mimeType)
	switch mimeType {
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
	// Add more MIME types as needed
	default:
		fmt.Printf("Unhandled MIME type: %s\n", mimeType)
	}

	// 4. Manual byte signature checks for older Office formats
	if len(body) >= 8 && bytes.Equal(body[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
		fmt.Println("OLE Compound Document detected.")
		// Use regex to find specific Office type identifiers
		if bytes.Contains(body, []byte("WordDocument")) {
			fmt.Println("Identified as .doc")
			return ".doc"
		}
		if bytes.Contains(body, []byte("Workbook")) || bytes.Contains(body, []byte("Book")) {
			fmt.Println("Identified as .xls")
			return ".xls"
		}
		if bytes.Contains(body, []byte("PowerPoint Document")) {
			fmt.Println("Identified as .ppt")
			return ".ppt"
		}
		fmt.Println("OLE Compound Document but specific type not identified. Assigning .ole")
		return ".ole" // Fallback for unidentified OLE types
	}

	// 5. Detect OOXML (Open XML) formats by inspecting ZIP structure
	if len(body) >= 4 && string(body[:4]) == "PK\x03\x04" {
		fmt.Println("ZIP archive detected. Inspecting internal structure for OOXML formats.")
		readerAt := bytes.NewReader(body)
		size := int64(len(body))
		zipReader, err := zip.NewReader(readerAt, size)
		if err == nil {
			for _, f := range zipReader.File {
				if strings.HasPrefix(f.Name, "ppt/") {
					fmt.Println("Identified as .pptx")
					return ".pptx"
				} else if strings.HasPrefix(f.Name, "word/") {
					fmt.Println("Identified as .docx")
					return ".docx"
				} else if strings.HasPrefix(f.Name, "xl/") {
					fmt.Println("Identified as .xlsx")
					return ".xlsx"
				}
			}
		} else {
			fmt.Printf("Error reading ZIP structure: %v\n", err)
		}
	}

	// 6. Default to empty string if no extension could be determined
	fmt.Println("Failed to determine file extension; returning empty string.")
	return ""
}

func isOLECompoundDocument(body []byte) bool {
    return len(body) >= 8 && bytes.Equal(body[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
}

func bytesContains(body []byte, substr string) bool {
    return bytes.Contains(body, []byte(substr))
}

func isOOXML(body []byte) bool {
    readerAt := bytes.NewReader(body)
    size := int64(len(body))
    zipReader, err := zip.NewReader(readerAt, size)
    if err != nil {
        return false
    }
    for _, f := range zipReader.File {
        if strings.HasPrefix(f.Name, "ppt/") || strings.HasPrefix(f.Name, "word/") || strings.HasPrefix(f.Name, "xl/") {
            return true
        }
    }
    return false
}

func getOOXMLExtension(body []byte) string {
    readerAt := bytes.NewReader(body)
    size := int64(len(body))
    zipReader, err := zip.NewReader(readerAt, size)
    if err != nil {
        return ""
    }
    for _, f := range zipReader.File {
        if strings.HasPrefix(f.Name, "ppt/") {
            return ".pptx"
        } else if strings.HasPrefix(f.Name, "word/") {
            return ".docx"
        } else if strings.HasPrefix(f.Name, "xl/") {
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