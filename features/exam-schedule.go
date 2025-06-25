package features

import (
	"bytes"
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

const (
	ExamTableSelector = "table.customTable"
	ExamRowsSelector  = "tbody tr"
	ExamCellSelector  = "td"
)

func GetExamSchedule(regNo string, cookies types.Cookies, sem_choice int) {
	if !helpers.ValidateLogin(cookies) {
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
	var allExams []types.ExamEvent

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

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
		if err != nil {
			if debug.Debug {
				fmt.Printf("Error parsing HTML document for Semester %s: %v\n", allSems[i].SemName, err)
			}
			continue
		}

		allExams, err = parseExamSchedule(doc)
		if err != nil {
			if debug.Debug {
				fmt.Printf("Error parsing exam schedule for Semester %s: %v\n", allSems[i].SemName, err)
			}
			continue
		}

		if len(allExams) > 0 {
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

	if len(allExams) == 0 {
		fmt.Println("No exams scheduled in this Semester.")
		return
	}

	// Filter and sort upcoming exams
	allExams = filterAndSortUpcomingExams(allExams)
	// Group exams by category
	var cat1Exams, cat2Exams, mtExams, fatExams []types.ExamEvent
	for _, exam := range allExams {
		switch exam.Category {
		case "CAT1":
			cat1Exams = append(cat1Exams, exam)
		case "CAT2":
			cat2Exams = append(cat2Exams, exam)
		case "MT":
			mtExams = append(mtExams, exam)
		case "FAT":
			fatExams = append(fatExams, exam)
		}
	}

	totalExams := len(cat1Exams) + len(cat2Exams) + len(mtExams) + len(fatExams)
	if totalExams == 0 {
		fmt.Println("No exams scheduled yet. Try checking VTOP.")
		return
	}

	// Display the grouped exams
	if len(cat1Exams) > 0 {
		fmt.Println("\nCAT1 EXAMS")
		fmt.Println()
		displayExamScheduleTable(cat1Exams)
	}
	if len(cat2Exams) > 0 {
		fmt.Println("\nCAT2 EXAMS")
		fmt.Println()
		displayExamScheduleTable(cat2Exams)
	}

	if len(mtExams) > 0 {
		fmt.Println("\nMID-TERM EXAMS")
		fmt.Println()
		displayExamScheduleTable(mtExams)
	}

	if len(fatExams) > 0 {
		fmt.Println("\nFAT EXAMS")
		fmt.Println()
		displayExamScheduleTable(fatExams)
	}

	// Generate ICS file with all upcoming exams
	if len(allExams) > 0 {
		generateICSFile(allExams)
	}
}

func safeGetCellText(cells *goquery.Selection, index int) string {
	if index < cells.Length() {
		return strings.TrimSpace(cells.Eq(index).Text())
	}
	return "-"
}

func parseExamSchedule(doc *goquery.Document) ([]types.ExamEvent, error) {
	var allExams []types.ExamEvent

	now := time.Now()
	todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	currentExamType := ""
	doc.Find(ExamTableSelector).Find(ExamRowsSelector).Each(func(i int, s *goquery.Selection) {
		// Check for section headers
		if s.Find("td.panelHead-secondary").Length() > 0 {
			headerText := strings.TrimSpace(s.Find("td.panelHead-secondary").Text())
			if debug.Debug {
				fmt.Printf("Found exam section header: %s\n", headerText)
			}
			currentExamType = headerText
			return
		}

		cells := s.Find(ExamCellSelector)
		if cells.Length() < 8 {
			if debug.Debug {
				fmt.Printf("Skipping row %d with insufficient cells (%d)\n", i+1, cells.Length())
			}
			return
		}

		serialNo := safeGetCellText(cells, 0)
		if _, err := strconv.Atoi(serialNo); err != nil {
			if debug.Debug {
				fmt.Printf("Skipping non-data row %d with serial '%s'\n", i+1, serialNo)
			}
			return
		}

		courseCode := safeGetCellText(cells, 1)
		courseTitle := safeGetCellText(cells, 2)
		slot := safeGetCellText(cells, 5)
		examDateStr := safeGetCellText(cells, 6)
		reportingTime := safeGetCellText(cells, 8)
		examTime := safeGetCellText(cells, 9)
		venue := safeGetCellText(cells, 10)
		seat := safeGetCellText(cells, 11)
		seatNo := safeGetCellText(cells, 12)

		if examDateStr == "" || strings.ToLower(examDateStr) == "exam date" || examDateStr == "-" {
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
			daysLeft++
		}
		if reportingTime != "" && reportingTime != "-" {
			examTime = fmt.Sprintf("%s (Report by: %s)", examTime, reportingTime)
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
			Category:    strings.ToUpper(currentExamType),
		}

		allExams = append(allExams, examEvent)
	})

	if debug.Debug {
		var cat1Count, cat2Count, mtCount, fatCount int
		for _, exam := range allExams {
			switch exam.Category {
			case "CAT1":
				cat1Count++
			case "CAT2":
				cat2Count++
			case "MT":
				mtCount++
			case "FAT":
				fatCount++
			}
		}
		fmt.Printf("Parsed exams: FAT=%d, CAT1=%d, CAT2=%d, MT=%d\n", fatCount, cat1Count, cat2Count, mtCount)
	}

	return allExams, nil
}

// Filter exams to only include upcoming ones and sort them by date
func filterAndSortUpcomingExams(exams []types.ExamEvent) []types.ExamEvent {
	var upcomingExams []types.ExamEvent

	// Filter for upcoming exams (date >= today)
	now := time.Now()
	todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, exam := range exams {
		if !exam.ExamDate.Before(todayDate) {
			upcomingExams = append(upcomingExams, exam)
		}
	}

	// Sort by date
	sort.Slice(upcomingExams, func(i, j int) bool {
		return upcomingExams[i].ExamDate.Before(upcomingExams[j].ExamDate)
	})

	return upcomingExams
}

func generateICSFile(exams []types.ExamEvent) {
	var icsEvents []types.ICSWithLocation
	for _, exam := range exams {
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

		// Use the Category field directly instead of trying to determine from course title
		examType := exam.Category

		eventwithoutlocation := types.ICSEvent{
			UID:     helpers.GenerateUID("Exam"),
			DtStamp: time.Now().UTC().Format(outputTimeLayout + "Z"),
			DtStart: startFormatted,
			DtEnd:   endFormatted,
			Summary: fmt.Sprintf("%s: %s - %s", examType, exam.Slot, exam.CourseTitle),
			Description: fmt.Sprintf("Exam for %s (%s) scheduled on %s at %s. Seat Number: %s.",
				exam.CourseTitle, exam.CourseCode, exam.ExamDate.Format("02-Jan-2006"), exam.Venue, exam.SeatNo),
		}

		event := types.ICSWithLocation{
			Event: eventwithoutlocation,
			Time:  fmt.Sprintf("%s - %s", startFormatted, endFormatted),
		}

		icsEvents = append(icsEvents, event)
	}

	// Create the Other Downloads/ICS File directory
	icsDir, err := helpers.GetOrCreateDownloadDir(filepath.Join("Other Downloads", "ICS File"))
	if err != nil {
		fmt.Println("Error creating ICS file directory:", err)
		return
	}

	icsFileName := "Exam_Schedule.ics"
	icsFilePath := filepath.Join(icsDir, icsFileName)

	err = helpers.VenueAdd(icsEvents, icsFilePath, "CLI-TOP Exams")
	if err != nil {
		fmt.Println("Error generating ICS file:", err)
	} else {
		uploadedFileURL, err := helpers.UploadICSFile(icsFilePath, helpers.CalendarServerURL)
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

func displayExamScheduleTable(exams []types.ExamEvent) {
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
