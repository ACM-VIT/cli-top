package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var tomorrowSubjectNoiseRegex = regexp.MustCompile(`\([^)]*\)|[^a-z0-9]+`)

type TomorrowClassAdvice struct {
	Class         types.Class
	Attendance    string
	Leverage      string
	Skip          string
	HasAttendance bool
}

func ResolvePlanningDate(now time.Time, dayOffset int) time.Time {
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return startOfDay.AddDate(0, 0, dayOffset)
}

func GetToday(regNo string, cookies types.Cookies, semChoice int, classGroupChoice int) {
	getPlanningDay(regNo, cookies, semChoice, classGroupChoice, 0, "Today")
}

func GetTomorrow(regNo string, cookies types.Cookies, semChoice int, classGroupChoice int) {
	getPlanningDay(regNo, cookies, semChoice, classGroupChoice, 1, "Tomorrow")
}

func GetDayAfter(regNo string, cookies types.Cookies, semChoice int, classGroupChoice int) {
	getPlanningDay(regNo, cookies, semChoice, classGroupChoice, 2, "Day After Tomorrow")
}

func getPlanningDay(regNo string, cookies types.Cookies, semChoice int, classGroupChoice int, dayOffset int, label string) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	semester, err := resolveHolidaySemester(regNo, cookies, semChoice)
	if err != nil {
		if err.Error() == "selection canceled by user" {
			helpers.Println("Selection canceled")
			return
		}
		if debug.Debug {
			helpers.Println(err)
		}
		helpers.Println("Error selecting semester:", err)
		return
	}

	classGroups := getClassGroups(regNo, cookies, semester)
	if len(classGroups) == 0 {
		helpers.Println("No class groups found")
		return
	}

	classGroupID, err := resolveHolidayClassGroupID(classGroups, classGroupChoice)
	if err != nil {
		helpers.Println(err)
		return
	}

	dateList := getDateList(regNo, cookies, semester, classGroupID)
	if len(dateList) == 0 {
		helpers.Println("No months found")
		return
	}

	locIndia := time.FixedZone("IST", 5*3600+1800)
	now := time.Now().In(locIndia)
	targetDate := ResolvePlanningDate(now, dayOffset)
	labelLower := strings.ToLower(label)

	calendarEntries := fetchCalendarEntries(regNo, cookies, semester, classGroupID, dateList, locIndia)
	entry, hasEntry := findCalendarEntryForDate(calendarEntries, targetDate)
	exams, err := fetchExamEventsForSemester(regNo, cookies, semester.SemID)
	if err != nil && debug.Debug {
		helpers.Printf("Error fetching exams for %s: %v\n", labelLower, err)
	}
	todaysExams := FilterExamsForDate(exams, targetDate)

	helpers.Printf("%s: %s\n", label, targetDate.Format("Monday, 02 Jan 2006"))

	if len(todaysExams) > 0 {
		examTable := [][]string{{"Course", "Category", "Time", "Venue"}}
		for _, exam := range todaysExams {
			examTable = append(examTable, []string{
				exam.CourseTitle,
				exam.Category,
				exam.ExamTime,
				exam.Venue,
			})
		}
		helpers.Println()
		helpers.Println("Exams")
		helpers.Println()
		helpers.PrintTable(examTable, 0)
	}

	if hasEntry && (entry.Status == "Holiday" || entry.Status == "No Instructional Day") {
		helpers.Printf("No classes %s.\n", labelLower)
		if entry.Reason != "" {
			helpers.Printf("Calendar: %s (%s)\n", entry.Status, entry.Reason)
		} else {
			helpers.Printf("Calendar: %s\n", entry.Status)
		}
		return
	}

	url := "https://vtop.vit.ac.in/vtop/processViewTimeTable"
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, "UTC", "POST", "")
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		helpers.Println("Error fetching timetable:", err)
		return
	}

	timetable := makeTT(schedule, getCourseNameFromHTML(bodyText))
	scheduleDay := targetDate.Weekday().String()
	if hasEntry && entry.DayOrder != "" {
		scheduleDay = entry.DayOrder
	}

	if scheduleDay != targetDate.Weekday().String() {
		helpers.Printf("%s follows %s schedule.\n", label, scheduleDay)
	}
	if hasEntry && entry.Status == "Instructional Day" && entry.Reason != "" {
		helpers.Printf("Calendar: %s\n", entry.Reason)
	}

	classes := append([]types.Class(nil), timetable[scheduleDay]...)
	sort.Slice(classes, func(i, j int) bool {
		return classes[i].StartTime < classes[j].StartTime
	})

	if len(classes) == 0 {
		helpers.Printf("No classes %s based on your timetable.\n", labelLower)
		return
	}

	attendanceRecords, err := fetchAttendanceRecordsForSemester(regNo, cookies, semester.SemID)
	if err != nil && debug.Debug {
		helpers.Printf("Error fetching attendance for %s: %v\n", labelLower, err)
	}

	advices := BuildTomorrowClassAdvices(classes, attendanceRecords)
	tableData := [][]string{{"Time", "Subject", "Venue", "%", "Leverage", "Skip?"}}
	skippable := 0
	unknown := 0

	for _, advice := range advices {
		if advice.Skip == "Yes" {
			skippable++
		}
		if !advice.HasAttendance {
			unknown++
		}

		leverage := advice.Leverage
		if leverage == "" {
			leverage = "Unknown"
		}

		tableData = append(tableData, []string{
			fmt.Sprintf("%s-%s", advice.Class.StartTime, advice.Class.EndTime),
			advice.Class.Subject,
			advice.Class.Venue,
			advice.Attendance,
			leverage,
			advice.Skip,
		})
	}

	helpers.Println()
	helpers.PrintTable(tableData, 0)
	helpers.Printf("\nSkippable %s: %d/%d classes.\n", labelLower, skippable, len(advices))
	if unknown > 0 {
		helpers.Printf("Attendance leverage unavailable for %d class(es).\n", unknown)
	}
}

func BuildTomorrowClassAdvices(classes []types.Class, attendance []AttendanceRecord) []TomorrowClassAdvice {
	sortedClasses := append([]types.Class(nil), classes...)
	sort.Slice(sortedClasses, func(i, j int) bool {
		return sortedClasses[i].StartTime < sortedClasses[j].StartTime
	})

	remaining := make(map[string]int)
	advices := make([]TomorrowClassAdvice, 0, len(sortedClasses))

	for _, class := range sortedClasses {
		record, ok := findAttendanceRecordForClass(class, attendance)
		advice := TomorrowClassAdvice{
			Class:      class,
			Attendance: "Unknown",
			Skip:       "Unknown",
		}
		if !ok {
			advices = append(advices, advice)
			continue
		}

		advice.HasAttendance = true
		advice.Attendance = formatAttendancePercentage(record.Percentage)
		advice.Leverage = record.Alert

		key := attendanceBudgetKey(record)
		if _, exists := remaining[key]; !exists {
			remaining[key] = record.CanMiss
		}

		if remaining[key] > 0 {
			advice.Skip = "Yes"
			remaining[key]--
		} else {
			advice.Skip = "No"
		}

		advices = append(advices, advice)
	}

	return advices
}

func findCalendarEntryForDate(entries []CalendarDayEntry, target time.Time) (CalendarDayEntry, bool) {
	targetDate := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, target.Location())
	for _, entry := range entries {
		entryDate := time.Date(entry.Date.Year(), entry.Date.Month(), entry.Date.Day(), 0, 0, 0, 0, entry.Date.Location())
		if entryDate.Equal(targetDate) {
			return entry, true
		}
	}
	return CalendarDayEntry{}, false
}

func findAttendanceRecordForClass(class types.Class, attendance []AttendanceRecord) (AttendanceRecord, bool) {
	subjectKey := normalizeTomorrowSubject(class.Subject)
	wantLab := strings.HasPrefix(strings.ToUpper(strings.TrimSpace(class.Slot)), "L")

	var fallback *AttendanceRecord
	for i := range attendance {
		record := &attendance[i]
		if normalizeTomorrowSubject(record.Subject) != subjectKey {
			continue
		}
		if record.IsLab == wantLab {
			return *record, true
		}
		if fallback == nil {
			fallback = record
		}
	}

	if fallback != nil {
		return *fallback, true
	}
	return AttendanceRecord{}, false
}

func attendanceBudgetKey(record AttendanceRecord) string {
	typeKey := "theory"
	if record.IsLab {
		typeKey = "lab"
	}
	return normalizeTomorrowSubject(record.Subject) + "|" + typeKey
}

func normalizeTomorrowSubject(subject string) string {
	subject = strings.ToLower(subject)
	subject = tomorrowSubjectNoiseRegex.ReplaceAllString(subject, " ")
	return strings.Join(strings.Fields(subject), " ")
}

func FilterExamsForDate(exams []types.ExamEvent, targetDate time.Time) []types.ExamEvent {
	var filtered []types.ExamEvent
	for _, exam := range exams {
		if sameCalendarDate(exam.ExamDate, targetDate) {
			filtered = append(filtered, exam)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ExamTime == filtered[j].ExamTime {
			return filtered[i].CourseTitle < filtered[j].CourseTitle
		}
		return filtered[i].ExamTime < filtered[j].ExamTime
	})

	return filtered
}

func sameCalendarDate(first, second time.Time) bool {
	return first.Year() == second.Year() &&
		first.Month() == second.Month() &&
		first.Day() == second.Day()
}

func formatAttendancePercentage(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "Unknown"
	}
	if strings.Contains(trimmed, "%") {
		return trimmed
	}
	return trimmed + "%"
}
