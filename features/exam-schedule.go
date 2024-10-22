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

type ExamEvent struct {
	CourseCode    string
	CourseTitle   string
	Slot          string
	ExamDate      time.Time
	ReportingTime string
	ExamTime      string
	Venue         string
	Seat          string
	SeatNo        string
	DaysLeft      int
}

func GetExamSchedule(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doSearchExamScheduleForStudent"
	semesterID := helpers.SelectSemester(regNo, cookies, sem_choice)
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semesterID, "UTC", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println("Error fetching exam schedule:", err)
	}
	if debug.Debug {
		fmt.Println("HTML Response:\n", string(bodyText))
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println("Error parsing HTML document:", err)
	}
	examSchedule, err := parseExamSchedule(doc)
	if err != nil {
		fmt.Println("Error parsing exam schedule:", err)
		return
	}
	if len(examSchedule) == 0 {
		fmt.Println("No exams scheduled.")
		return
	}
	sortExamsByDateAsc(examSchedule)

	upcomingExams := []ExamEvent{}
	for _, exam := range examSchedule {
		if exam.DaysLeft >= 0 {
			upcomingExams = append(upcomingExams, exam)
		}
	}

	if len(upcomingExams) == 0 {
		fmt.Println("No upcoming exams scheduled!")
		return
	}

	icsFileName := "Exam_Schedule.ics"
	err = GenerateExamICSFile(upcomingExams, icsFileName)
	if err != nil {
		fmt.Println("Error generating ICS file:", err)
	} else {
		serverURL := "https://syllabi.examcooker.in"
		uploadedFileURL, err := helpers.UploadICSFile(icsFileName, serverURL)
		if err != nil {
			fmt.Println("Error uploading ICS file:", err)
			fmt.Println("Please import the 'Exam_Schedule.ics' file manually.")
		} else {
			displayExamScheduleTable(upcomingExams)
			fmt.Println()
			fmt.Println("ICS file generated and saved successfully.")
			helpers.GenerateCalendarImportLinks(uploadedFileURL, "Exams")
		}
	}
}

func parseExamSchedule(doc *goquery.Document) ([]ExamEvent, error) {
	var exams []ExamEvent

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
			reportingTime := strings.TrimSpace(cells.Eq(8).Text())
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

			examEvent := ExamEvent{
				CourseCode:    courseCode,
				CourseTitle:   courseTitle,
				Slot:          slot,
				ExamDate:      examDate,
				ReportingTime: reportingTime,
				ExamTime:      examTime,
				Venue:         venue,
				Seat:          seat,
				SeatNo:        seatNo,
				DaysLeft:      daysLeft,
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

func sortExamsByDateAsc(exams []ExamEvent) {
	sort.Slice(exams, func(i, j int) bool {
		return exams[i].ExamDate.Before(exams[j].ExamDate)
	})
}

func displayExamScheduleTable(exams []ExamEvent) {
	var tableData [][]string
	tableData = append(tableData, []string{"Code", "Course Title", "Slot", "Exam Date", "Reporting Time", "Exam Time", "Venue", "Seat", "Seat No.", "Days Left"})

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
		color := "\033[32m" 
		if exam.DaysLeft < 3 {
			color = "\033[31m" 
		} else if exam.DaysLeft < 7 {
			color = "\033[33m" 
		}
		reset := "\033[0m"
		daysLeftColored := color + daysLeftStr + reset

		courseTitle := helpers.TruncateWithEllipsis(exam.CourseTitle, 40)

		tableData = append(tableData, []string{
			exam.CourseCode,
			courseTitle,
			exam.Slot,
			exam.ExamDate.Format("02-Jan-2006"),
			exam.ReportingTime,
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
		helpers.PrintTable(tableData)
	}
}

func GenerateExamICSFile(exams []ExamEvent, filePath string) error {
	if len(exams) == 0 {
		return nil 
	}

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
	_, err = file.WriteString("BEGIN:VTIMEZONE\r\nTZID:Asia/Kolkata\r\nBEGIN:STANDARD\r\nDTSTART:19700101T000000\r\nTZOFFSETFROM:+0530\r\nTZOFFSETTO:+0530\r\nTZNAME:IST\r\nEND:STANDARD\r\nEND:VTIMEZONE\r\n")
	if err != nil {
		return err
	}

	for _, exam := range exams {
		data := fmt.Sprintf("%s-%s-%s-%s", exam.CourseCode, exam.Slot, exam.ExamDate.Format("20060102"), exam.ExamTime)
		uid := helpers.GenerateUID(data)
		dtstamp := time.Now().UTC().Format("20060102T150405Z")
		startDateTime, err := parseExamDateTime(exam.ExamDate, exam.ExamTime, true)
		if err != nil {
			if debug.Debug {
				fmt.Println("Error parsing exam start time:", err)
			}
			continue
		}
		endDateTime, err := parseExamDateTime(exam.ExamDate, exam.ExamTime, false)
		if err != nil {
			if debug.Debug {
				fmt.Println("Error parsing exam end time:", err)
			}
			continue
		}

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
		_, err = file.WriteString(fmt.Sprintf("DTSTART;TZID=Asia/Kolkata:%s\r\n", startDateTime))
		if err != nil {
			return err
		}
		_, err = file.WriteString(fmt.Sprintf("DTEND;TZID=Asia/Kolkata:%s\r\n", endDateTime))
		if err != nil {
			return err
		}
		summary := fmt.Sprintf("%s - %s", exam.Slot, exam.CourseTitle)
		_, err = file.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", helpers.EscapeString(summary)))
		if err != nil {
			return err
		}
		description := fmt.Sprintf("Exam for %s (%s)", exam.CourseTitle, exam.CourseCode)
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


func parseExamDateTime(examDate time.Time, examTime string, isStart bool) (string, error) {
    timeParts := strings.Split(examTime, "-")
    if len(timeParts) != 2 {
        return "", fmt.Errorf("invalid exam time format: %s", examTime)
    }

    timeStr := strings.TrimSpace(timeParts[0])
    if !isStart {
        timeStr = strings.TrimSpace(timeParts[1])
    }

    if timeStr == "" || strings.ToLower(timeStr) == "exam time" {
        return "", fmt.Errorf("invalid exam time format: %s", examTime)
    }

    layout := "02-Jan-2006 03:04 PM"

    dateTimeStr := fmt.Sprintf("%s %s", examDate.Format("02-Jan-2006"), timeStr)

    istLocation, err := time.LoadLocation("Asia/Kolkata")
    if err != nil {
        return "", fmt.Errorf("failed to load IST location: %v", err)
    }

    localDateTime, err := time.ParseInLocation(layout, dateTimeStr, istLocation)
    if err != nil {
        if debug.Debug {
            fmt.Printf("Error parsing date time '%s': %v\n", dateTimeStr, err)
        }
        return "", err
    }

    return localDateTime.Format("20060102T150405"), nil
}
