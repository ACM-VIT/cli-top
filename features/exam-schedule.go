// features/exam-schedule.go
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

// GetExamSchedule fetches, processes, and displays the exam schedule for a student,
// and generates an ICS file for calendar integration.
func GetExamSchedule(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doSearchExamScheduleForStudent"

	// Fetch semester details
	semDetails, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			fmt.Println(err)
		}
		fmt.Println("Failed to retrieve semester details.")
		return
	}

	if len(semDetails) == 0 {
		fmt.Println("No semester details found.")
		return
	}

	semesterID := semDetails[len(semDetails)-1].SemID

	// Fetch exam schedule data
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semesterID, "UTC", "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching exam schedule:", err)
		}
		fmt.Println("Failed to fetch exam schedule.")
		return
	}

	if debug.Debug {
		fmt.Println("HTML Response:\n", string(bodyText))
	}

	// Parse the HTML response
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML document:", err)
		}
		fmt.Println("Failed to parse exam schedule data.")
		return
	}

	// Parse exam schedule from the document
	examSchedule, err := parseExamSchedule(doc)
	if err != nil {
		fmt.Println("Error parsing exam schedule:", err)
		return
	}

	if len(examSchedule) == 0 {
		fmt.Println("No exams scheduled.")
		return
	}

	// Sort exams by date in ascending order
	sortExamsByDateAsc(examSchedule)

	// Filter upcoming exams (days left >= 0)
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

	// Display the exam schedule in a table
	displayExamScheduleTable(upcomingExams)

	// **Map types.ExamEvent to helpers.ICSEvent**
	var icsEvents []helpers.ICSEvent
	for _, exam := range upcomingExams {
		// Define start and end times for the entire exam day up to 23:59 PM
		startTime := exam.ExamDate.Format("20060102T000000") // Start of the day
		endTime := exam.ExamDate.Format("20060102T235900")   // End of the day

		// Create ICSEvent
		event := helpers.ICSEvent{
			UID:         helpers.GenerateUID("Exam"),
			DtStamp:     time.Now().UTC().Format("20060102T150405Z"),
			DtStart:     startTime,
			DtEnd:       endTime,
			Summary:     fmt.Sprintf("Exam: %s - %s", exam.Slot, exam.CourseTitle),
			Description: fmt.Sprintf("Exam for %s (%s) scheduled on %s at %s.",
				exam.CourseTitle, exam.CourseCode, exam.ExamDate.Format("02-Jan-2006"), exam.Venue),
		}

		icsEvents = append(icsEvents, event)
	}

	// Generate ICS file for the exam schedule
	icsFileName := "Exam_Schedule.ics"
	icsFilePath := filepath.Join(helpers.GetDownloadsDir(), icsFileName)

	err = helpers.GenerateICSFile(icsEvents, icsFilePath)
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

// parseExamSchedule parses the exam schedule from the HTML document.
func parseExamSchedule(doc *goquery.Document) ([]types.ExamEvent, error) {
	var exams []types.ExamEvent

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

			if examDateStr == "" || examDateStr == "Exam Date" {
				return
			}

			examDate, err := time.Parse("02-Jan-2006", examDateStr)
			if err != nil {
				if debug.Debug {
					fmt.Println("Error parsing exam date:", err)
				}
				return
			}

			daysLeft := int(examDate.Sub(time.Now()).Hours() / 24)

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

// sortExamsByDateAsc sorts the exams slice in ascending order based on ExamDate.
func sortExamsByDateAsc(exams []types.ExamEvent) {
	sort.Slice(exams, func(i, j int) bool {
		return exams[i].ExamDate.Before(exams[j].ExamDate)
	})
}

// displayExamScheduleTable displays the exam schedule in a formatted table.
func displayExamScheduleTable(exams []types.ExamEvent) {
	fmt.Println()

	var tableData [][]string
	// Table headers
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
