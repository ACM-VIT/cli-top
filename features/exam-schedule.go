package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func GetExamSchedule(regNo string, cookies types.Cookies, sem_choice int) {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		fmt.Println("Please login using the cli-top login command.")
		return
	}

	url := "https://vtop.vit.ac.in/vtop/examinations/doSearchExamScheduleForStudent"

	allSems, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error retrieving semester details:", err)
		}
		fmt.Println("Failed to retrieve semester details.")
		return
	}

	if len(allSems) == 0 {
		fmt.Println("No semester details found.")
		return
	}

	var semID string
	var examSchedule []types.ExamEvent

	for i := len(allSems) - 1; i >= 0; i-- {
		semID = allSems[i].SemID
		bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
		if err != nil {
			if debug.Debug {
				fmt.Printf("Error fetching exam schedule for Semester %s: %v\n", allSems[i].SemName, err)
			}
			continue
		}

		if debug.Debug {
			fmt.Printf("HTML Response for Semester %s:\n%s\n", allSems[i].SemName, string(bodyText))
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
		if err != nil {
			if debug.Debug {
				fmt.Printf("Error parsing HTML document for Semester %s: %v\n", allSems[i].SemName, err)
			}
			continue
		}

		examSchedule, err = parseExamSchedule(doc)
		if err != nil {
			if debug.Debug {
				fmt.Printf("Error parsing exam schedule for Semester %s: %v\n", allSems[i].SemName, err)
			}
			continue
		}

		if len(examSchedule) > 0 {
			if debug.Debug {
				fmt.Printf("Selected Semester: %s (%s)\n", allSems[i].SemName, semID)
			}
			break
		} else {
			if debug.Debug {
				fmt.Printf("No exams found for Semester: %s (%s). Trying previous semester.\n", allSems[i].SemName, semID)
			}
		}
	}

	if len(examSchedule) == 0 {
		fmt.Println("No exams scheduled in any semester.")
		return
	}

	sortExamsByDateAsc(examSchedule)

	upcomingExams := []types.ExamEvent{}
	for _, exam := range examSchedule {
		if exam.DaysLeft >= 0 {
			upcomingExams = append(upcomingExams, exam)
		}
	}

	if len(upcomingExams) == 0 {
		fmt.Println("No upcoming exams scheduled!")
		return
	}

	displayExamScheduleTable(upcomingExams)

	var icsEvents []types.ICSWithLocation
	for _, exam := range upcomingExams {

		timeRange := strings.Split(exam.ExamTime, " - ")
		if len(timeRange) != 2 {
			fmt.Println("Invalid time range format")
			continue
		}

		inputTimeLayout := "3:04 PM"
		outputTimeLayout := "20060102T150405"

		startTime, err1 := time.Parse(inputTimeLayout, timeRange[0])
		endTime, err2 := time.Parse(inputTimeLayout, timeRange[1])

		if err1 != nil || err2 != nil {
			fmt.Println("Error parsing time range:", err1, err2)
			continue
		}

		examDate := exam.ExamDate
		location := examDate.Location()

		startDateTime := time.Date(
			examDate.Year(), examDate.Month(), examDate.Day(),
			startTime.Hour(), startTime.Minute(), startTime.Second(), 0, location,
		)
		endDateTime := time.Date(
			examDate.Year(), examDate.Month(), examDate.Day(),
			endTime.Hour(), endTime.Minute(), endTime.Second(), 0, location,
		)

		startFormatted := fmt.Sprintf("TZID=Asia/Kolkata:%s", startDateTime.Format(outputTimeLayout))
		endFormatted := fmt.Sprintf("TZID=Asia/Kolkata:%s", endDateTime.Format(outputTimeLayout))

		eventwithoutlocation := types.ICSEvent{
			UID:     helpers.GenerateUID("Exam"),
			DtStamp: time.Now().UTC().Format(outputTimeLayout + "Z"), // DtStamp in UTC
			DtStart: startFormatted,
			DtEnd:   endFormatted,
			Summary: fmt.Sprintf("Exam: %s - %s", exam.Slot, exam.CourseTitle),
			Description: fmt.Sprintf("Exam for %s (%s) scheduled on %s at %s. Seat Number: %s.",
				exam.CourseTitle, exam.CourseCode, exam.ExamDate.Format("02-Jan-2006"), exam.Venue, exam.SeatNo),
		}

		event := types.ICSWithLocation{
			Event: eventwithoutlocation,
			Time:  fmt.Sprintf("%s - %s", startFormatted, endFormatted), // Include both start and end
		}

		fmt.Println(event)

		icsEvents = append(icsEvents, event)
	}

	icsFileName := "Exam_Schedule.ics"
	icsFilePath := filepath.Join(helpers.GetDownloadsDir(), icsFileName)

	err = helpers.VenueAdd(icsEvents, icsFilePath, "CLI-TOP Exams")
	if err != nil {
		fmt.Println("Error generating ICS file:", err)
	} else {
		serverURL := "https://cli-calendar.acmvit.in"
		uploadedFileURL, err := helpers.UploadICSFile(icsFilePath, serverURL)
		if err != nil {
			fmt.Println("Error uploading ICS file:", err)
			fmt.Println("Please import the 'Exam_Schedule.ics' file manually from your Downloads folder.")
		} else {
			fmt.Println()
			fmt.Println("ICS file generated and saved successfully.")
			helpers.GenerateCalendarImportLinks(uploadedFileURL, "Exams")
		}
	}
}

func parseExamSchedule(doc *goquery.Document) ([]types.ExamEvent, error) {
	var exams []types.ExamEvent

	now := time.Now()
	todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	doc.Find("table.customTable tbody tr").Each(func(i int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() >= 13 {
			serialNo := strings.TrimSpace(cells.Eq(0).Text())
			if _, err := strconv.Atoi(serialNo); err != nil {
				if debug.Debug {
					fmt.Printf("Skipping non-data row %d with serial '%s'.\n", i+1, serialNo)
				}
				return
			}

			courseCode := strings.TrimSpace(cells.Eq(1).Text())
			courseTitle := strings.TrimSpace(cells.Eq(2).Text())
			slot := strings.TrimSpace(cells.Eq(5).Text())
			examDateStr := strings.TrimSpace(cells.Eq(6).Text())
			examTime := strings.TrimSpace(cells.Eq(9).Text())
			venue := strings.TrimSpace(cells.Eq(10).Text())
			seat := strings.TrimSpace(cells.Eq(11).Text())
			seatNo := strings.TrimSpace(cells.Eq(12).Text())

			if examDateStr == "" || strings.ToLower(examDateStr) == "exam date" {
				return
			}

			examDate, err := time.ParseInLocation("02-Jan-2006", examDateStr, now.Location())
			if err != nil {
				if debug.Debug {
					fmt.Printf("Error parsing exam date '%s': %v\n", examDateStr, err)
				}
				return
			}

			daysLeft := int(examDate.Sub(todayDate).Hours() / 24)

			if examDate.After(todayDate) && examDate.Sub(todayDate).Hours()/24 > float64(daysLeft) {
				daysLeft += 1
			}

			examEvent := types.ExamEvent{
				CourseCode:  courseCode,
				CourseTitle: courseTitle,
				Slot:        slot,
				ExamDate:    examDate,
				ExamTime:    examTime,
				Venue:       venue,
				Seat:        seat,
				SeatNo:      seatNo,
				DaysLeft:    daysLeft,
			}
			exams = append(exams, examEvent)
		} else if debug.Debug {
			fmt.Printf("Unexpected number of cells (%d) in row %d. Expected at least 13.\n", cells.Length(), i+1)
		}
	})

	if debug.Debug && len(exams) == 0 {
		fmt.Println("Parsed exams:", len(exams))
	}

	return exams, nil
}

func sortExamsByDateAsc(exams []types.ExamEvent) {
	sort.Slice(exams, func(i, j int) bool {
		return exams[i].ExamDate.Before(exams[j].ExamDate)
	})
}

func displayExamScheduleTable(exams []types.ExamEvent) {
	fmt.Println()

	var tableData [][]string
	tableData = append(tableData, []string{
		"Code", "Course Title", "Slot", "Exam Date", "Exam Time", "Venue", "Seat", "Seat No.", "Days Left",
	})

	maxCourseTitleLength := 30

	for _, exam := range exams {
		venue := exam.Venue
		if venue == "-" {
			venue = "TBA"
		}

		seat := exam.Seat
		if seat == "-" {
			seat = "TBA"
		}

		seatNo := exam.SeatNo
		if seatNo == "-" {
			seatNo = "TBA"
		}

		daysLeftStr := strconv.Itoa(exam.DaysLeft)
		color := "\033[32m" // Green
		if exam.DaysLeft < 3 {
			color = "\033[31m" // Red
		} else if exam.DaysLeft < 7 {
			color = "\033[33m" // Yellow
		}
		reset := "\033[0m"
		daysLeftColored := color + daysLeftStr + reset

		courseTitle := helpers.TruncateWithEllipses(exam.CourseTitle, maxCourseTitleLength)

		tableData = append(tableData, []string{
			exam.CourseCode,
			courseTitle,
			exam.Slot,
			exam.ExamDate.Format("02-Jan-2006"),
			exam.ExamTime,
			venue,
			seat,
			seatNo,
			daysLeftColored,
		})
	}
	if len(tableData) == 1 {
		fmt.Println("No upcoming exams scheduled!")
	} else {
		helpers.PrintTable(tableData, 1)
	}
}
