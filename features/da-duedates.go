package features

import (
    "bytes"
    "cli-top/debug"
    "cli-top/helpers"
    "cli-top/types"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/PuerkitoBio/goquery"
)

type DAEvent struct {
    Name    string
    Title   string
    DueDate time.Time
}

func PrintDAdates(regNo string, cookies types.Cookies) {
    listOfSubjects := getAllSubs(regNo, cookies)
    var table_data [][]string
    var daEvents []DAEvent
    table_data = append(table_data, []string{"Name", "Title", "Date", "Days Left"})
    currentDate := time.Now()
    for _, detail := range listOfSubjects {
        code := detail[2]
        name := detail[0]
        doc := getOneSub(regNo, cookies, code)
        lastest_work := lastestda(doc)
        if len(lastest_work) > 0 {
            title := lastest_work[0]
            datestr := lastest_work[1]
            date, err := time.Parse("02-Jan-2006", datestr)
            if err != nil {
                fmt.Println("Error parsing date:", err)
                continue
            }
            daysLeft := int(date.Sub(currentDate).Hours()/24) + 1
            daysString := strconv.Itoa(daysLeft)
            color := "\033[32m"
            if daysLeft < 3 {
                color = "\033[31m"
            } else if daysLeft < 7 {
                color = "\033[33m"
            }
            reset := "\033[0m"
            daysString = color + daysString + reset
            table_data = append(table_data, []string{name, title, datestr, daysString})

            daEvent := DAEvent{
                Name:    name,
                Title:   title,
                DueDate: date,
            }
            daEvents = append(daEvents, daEvent)
        }
    }
    if len(table_data) == 1 {
        fmt.Println("YAYYYY!! No DA's due")
        return
    }
    helpers.PrintTable(table_data)
    fmt.Println()

    err := GenerateICSFile(daEvents)
    if err != nil {
        fmt.Println("Error generating ICS file:", err)
    } else {
        fmt.Println("Calendar file 'DAs.ics' generated with the upcoming DA deadlines.")

        serverURL := "https://syllabi.examcooker.in" // temporary hopefully :P
        uploadedFileURL, err := UploadICSFile("DAs.ics", serverURL)
        if err != nil {
            fmt.Println("Error uploading ICS file:", err)
            fmt.Println("Please import the 'DAs.ics' file manually.")
        } else {
            fmt.Println("ICS file uploaded successfully.")
            fmt.Printf("File URL: %s\n", uploadedFileURL)

            GenerateCalendarImportLinks(uploadedFileURL)
        }
    }
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

func GenerateCalendarImportLinks(icsURL string) {
    fmt.Println("\nImport all DAs into your calendar using the links below:")

    googleLink := GenerateGoogleCalendarImportLink(icsURL)
    fmt.Printf("Google Calendar Import Link: %s\n", makeANSILink("Click here", googleLink))

    outlookLink := GenerateOutlookCalendarImportLink(icsURL)
    fmt.Printf("Outlook Calendar Import Link: %s\n", makeANSILink("Click here", outlookLink))
}

func GenerateGoogleCalendarImportLink(icsURL string) string {
    baseURL := "https://calendar.google.com/calendar/r?cid="
    encodedURL := url.QueryEscape(icsURL)
    return fmt.Sprintf("%s%s", baseURL, encodedURL)
}

func GenerateOutlookCalendarImportLink(icsURL string) string {
    baseURL := "https://outlook.live.com/owa/?path=/calendar/action/subscribe&url=%s&name=%s"
    encodedURL := url.QueryEscape(icsURL)
    calendarName := url.QueryEscape("DAs")
    return fmt.Sprintf(baseURL, encodedURL, calendarName)
}

func makeANSILink(text, url string) string {
    return fmt.Sprintf("\u001B]8;;%s\a%s\u001B]8;;\a", url, text)
}

func GenerateICSFile(events []DAEvent) error {
    file, err := os.Create("DAs.ics")
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.WriteString("BEGIN:VCALENDAR\r\n")
    if err != nil {
        return err
    }
    _, err = file.WriteString("VERSION:2.0\r\n")
    if err != nil {
        return err
    }
    _, err = file.WriteString("PRODID:-//CLI-TOP//EN\r\n")
    if err != nil {
        return err
    }

    for _, event := range events {
        uid := fmt.Sprintf("%d-%s", time.Now().UnixNano(), event.Name)
        dtstamp := time.Now().UTC().Format("20060102T150405Z")
        startDate := event.DueDate.Format("20060102")
        endDate := event.DueDate.AddDate(0, 0, 1).Format("20060102")

        _, err = file.WriteString("BEGIN:VEVENT\r\n")
        if err != nil {
            return err
        }
        _, err = file.WriteString(fmt.Sprintf("UID:%s\r\n", uid))
        if err != nil {
            return err
        }
        _, err = file.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", dtstamp))
        if err != nil {
            return err
        }
        _, err = file.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", startDate))
        if err != nil {
            return err
        }
        _, err = file.WriteString(fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", endDate))
        if err != nil {
            return err
        }
        summary := fmt.Sprintf("%s - %s", event.Name, event.Title)
        _, err = file.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeString(summary)))
        if err != nil {
            return err
        }
        description := fmt.Sprintf("DA due for %s: %s", event.Name, event.Title)
        _, err = file.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeString(description)))
        if err != nil {
            return err
        }
        _, err = file.WriteString("END:VEVENT\r\n")
        if err != nil {
            return err
        }
    }

    _, err = file.WriteString("END:VCALENDAR\r\n")
    if err != nil {
        return err
    }

    return nil
}

func escapeString(s string) string {
    s = strings.ReplaceAll(s, "\\", "\\\\")
    s = strings.ReplaceAll(s, ";", "\\;")
    s = strings.ReplaceAll(s, ",", "\\,")
    s = strings.ReplaceAll(s, "\n", "\\n")
    return s
}

func allSubDetails(doc *goquery.Document) [][]string {
    var details [][]string

    doc.Find("tr.tableContent").Each(func(i int, s *goquery.Selection) {
        td := s.Find("td")
        code := td.Eq(1).Text()
        subject := td.Eq(2).Text()
        name := td.Eq(3).Text()

        detail := []string{name, subject, code}
        details = append(details, detail)
    })
    fmt.Println()
    return details
}

func lastestda(doc *goquery.Document) []string {
    var details []string

    doc.Find("tr.fixedContent.tableContent").EachWithBreak(func(i int, s *goquery.Selection) bool {
        td := s.Find("td")
        name := td.Eq(1).Text()
        span := td.Eq(4).Find("span")
        date := span.Text()

        style, exists := span.Attr("style")
        if exists && strings.Contains(style, "color: green;") {
            details = append(details, name)
            details = append(details, date)
            return false
        }
        return true
    })
    return details
}

func getAllSubs(regNo string, cookies types.Cookies) [][]string {
    url := "https://vtop.vit.ac.in/vtop/examinations/doDigitalAssignment"
    semDetails := helpers.GetSemDetails(cookies, regNo)
    if len(semDetails.SemIds) == 0 {
        fmt.Println("No semesters found")
        return nil
    }
    semID := semDetails.SemIds[0]
    bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
    if err != nil && debug.Debug {
        fmt.Println(err)
    }
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
    if err != nil && debug.Debug {
        fmt.Println(err)
    }

    return allSubDetails(doc)
}

func getOneSub(regNo string, cookies types.Cookies, code string) *goquery.Document {
    url := "https://vtop.vit.ac.in/vtop/examinations/processDigitalAssignment"
    payloadMap := map[string]string{
        "_csrf":         cookies.CSRF,
        "paramReturnId": "getCourseForCoursePage",
        "classId":       code,
        "authorizedID":  regNo,
        "x":             fmt.Sprintf("%d", time.Now().Unix()),
    }
    formData := helpers.FormatBodyData(payloadMap)
    subBody, err := helpers.FetchReq(regNo, cookies, url, "", formData, "POST", "")
    if err != nil && debug.Debug {
        fmt.Println(err)
    }
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(subBody)))
    if err != nil && debug.Debug {
        fmt.Println(err)
    }
    return doc
}
