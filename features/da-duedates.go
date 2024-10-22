package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type DAEvent struct {
	SubjectName string
	Title       string
	DueDate     time.Time
	DaysLeft    int
}

type SubjectDAs struct {
	SubjectName string
	DAs         []DAEvent
}

func PrintDAdates(regNo string, cookies types.Cookies) {
	listOfSubjects := getAllSubs(regNo, cookies)
	if len(listOfSubjects) == 0 {
		fmt.Println("No subjects found.")
		return
	}
	var subjectsWithDAs []SubjectDAs
	var allDAs []DAEvent
	for _, detail := range listOfSubjects {
		code := detail[2]
		subjectName := detail[0]
		doc := getOneSub(regNo, cookies, code)
		pendingAssignments := pendingDAs(doc, subjectName)
		if len(pendingAssignments) > 0 {
			sortDAsByDueDateAsc(pendingAssignments)
			allDAs = append(allDAs, pendingAssignments...)
			subjectDAs := SubjectDAs{
				SubjectName: subjectName,
				DAs:         pendingAssignments,
			}
			subjectsWithDAs = append(subjectsWithDAs, subjectDAs)
		}
	}
	if len(allDAs) == 0 {
		fmt.Println("YAYYYY!! No DA's due")
		return
	}
	icsFileName := "DAs_All.ics"
	err := GenerateDAICSFile(allDAs, icsFileName)
	if err != nil {
		fmt.Println("Error generating ICS file:", err)
	} else {
		serverURL := "https://cli-calendar.acmvit.in"
		uploadedFileURL, err := helpers.UploadICSFile(icsFileName, serverURL)
		if err != nil {
			fmt.Println("Error uploading ICS file:", err)
			fmt.Println("Please import the 'DAs_All.ics' file manually.")
		} else {
			var tableData [][]string
			tableData = append(tableData, []string{"Course", "Title", "Date", "Days Left"})
			for _, subjectDAs := range subjectsWithDAs {
				latestDA := subjectDAs.DAs[0]
				daysLeftStr := strconv.Itoa(latestDA.DaysLeft)
				color := "\033[32m"
				if latestDA.DaysLeft < 3 {
					color = "\033[31m"
				} else if latestDA.DaysLeft < 7 {
					color = "\033[33m"
				}
				reset := "\033[0m"
				daysLeftColored := color + daysLeftStr + reset
				title := helpers.TruncateWithEllipsis(latestDA.Title, 50)
				tableData = append(tableData, []string{
					subjectDAs.SubjectName,
					title,
					latestDA.DueDate.Format("02-Jan-2006"),
					daysLeftColored,
				})
			}
			helpers.PrintTable(tableData)
			fmt.Println()
			fmt.Println("ICS file generated and saved successfully.")
			helpers.GenerateCalendarImportLinks(uploadedFileURL, "DAs")
		}
	}
	fmt.Println()
	for {
		fmt.Print("Enter the INDEX number of the course to view all DAs (or 'q' to quit): ")
		var input string
		fmt.Scan(&input)
		if strings.ToLower(input) == "q" {
			break
		}
		index, err := strconv.Atoi(input)
		if err != nil || index < 1 || index > len(subjectsWithDAs) {
			fmt.Println("Invalid input. Please enter a valid index number.")
			continue
		}
		selectedCourse := subjectsWithDAs[index-1]
		fmt.Printf("\nAll DAs for %s:\n", selectedCourse.SubjectName)
		fmt.Println()
		var courseTable [][]string
		courseTable = append(courseTable, []string{"Title", "Date", "Days Left"})
		for _, da := range selectedCourse.DAs {
			daysLeftStr := strconv.Itoa(da.DaysLeft)
			color := "\033[32m"
			if da.DaysLeft < 3 {
				color = "\033[31m"
			} else if da.DaysLeft < 7 {
				color = "\033[33m"
			}
			reset := "\033[0m"
			daysLeftColored := color + daysLeftStr + reset
			title := helpers.TruncateWithEllipsis(da.Title, 50)
			courseTable = append(courseTable, []string{
				title,
				da.DueDate.Format("02-Jan-2006"),
				daysLeftColored,
			})
		}
		helpers.PrintTable(courseTable)
		fmt.Println()
	}
	fmt.Println("Program terminated.")
	return
}

func sortDAsByDueDateAsc(das []DAEvent) {
	sort.Slice(das, func(i, j int) bool {
		return das[i].DueDate.Before(das[j].DueDate)
	})
}

func GenerateDAICSFile(events []DAEvent, filePath string) error {
	file, err := os.Create(filePath)
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
		data := fmt.Sprintf("%s-%s-%s", event.SubjectName, event.Title, event.DueDate.Format("20060102"))
		uid := helpers.GenerateUID(data)
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
		summary := fmt.Sprintf("%s - %s", event.SubjectName, event.Title)
		_, err = file.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", helpers.EscapeString(summary)))
		if err != nil {
			return err
		}
		description := fmt.Sprintf("DA due for %s: %s", event.SubjectName, event.Title)
		_, err = file.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", helpers.EscapeString(description)))
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

func pendingDAs(doc *goquery.Document, subjectName string) []DAEvent {
	var events []DAEvent
	currentDate := time.Now()
	doc.Find("tr.fixedContent.tableContent").Each(func(i int, s *goquery.Selection) {
		td := s.Find("td")
		if td.Length() < 5 {
			return
		}
		title := strings.TrimSpace(td.Eq(1).Text())
		span := td.Eq(4).Find("span")
		dateStr := strings.TrimSpace(span.Text())
		style, exists := span.Attr("style")
		if exists && strings.Contains(style, "color: green;") {
			date, err := time.Parse("02-Jan-2006", dateStr)
			if err != nil {
				fmt.Printf("Error parsing date '%s': %v\n", dateStr, err)
				return
			}
			daysLeft := int(date.Sub(currentDate).Hours()/24) + 1
			daEvent := DAEvent{
				SubjectName: subjectName,
				Title:       title,
				DueDate:     date,
				DaysLeft:    daysLeft,
			}
			events = append(events, daEvent)
		}
	})
	return events
}

func getAllSubs(regNo string, cookies types.Cookies) [][]string {
	url := "https://vtop.vit.ac.in/vtop/examinations/doDigitalAssignment"
	semDetails := helpers.GetSemDetails(cookies, regNo)
	if len(semDetails.SemIds) == 0 {
		fmt.Println("No semesters found.")
		return nil
	}
	semID := semDetails.SemIds[0]
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
	if err != nil && debug.Debug {
		fmt.Printf("Error fetching subjects: %v\n", err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Printf("Error parsing subjects document: %v\n", err)
	}
	return allSubDetails(doc)
}

func allSubDetails(doc *goquery.Document) [][]string {
	var details [][]string
	doc.Find("tr.tableContent").Each(func(i int, s *goquery.Selection) {
		td := s.Find("td")
		code := strings.TrimSpace(td.Eq(1).Text())
		subject := strings.TrimSpace(td.Eq(2).Text())
		name := strings.TrimSpace(td.Eq(3).Text())
		detail := []string{name, subject, code}
		details = append(details, detail)
	})
	fmt.Println()
	return details
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
		fmt.Printf("Error fetching subject details for code %s: %v\n", code, err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(subBody)))
	if err != nil && debug.Debug {
		fmt.Printf("Error parsing subject details document for code %s: %v\n", code, err)
	}
	return doc
}
