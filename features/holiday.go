package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type CalendarDayEntry struct {
	Date     time.Time
	Status   string
	Reason   string
	DayOrder string
}

type UpcomingHoliday struct {
	Date     time.Time
	Status   string
	Reason   string
	Schedule string
}

func GetHolidayList(regNo string, cookies types.Cookies, semChoice int, classGroupChoice int) {
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
	calendarEntries := fetchCalendarEntries(regNo, cookies, semester, classGroupID, dateList, locIndia)

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
	upcoming := FilterUpcomingClassImpactingHolidays(calendarEntries, timetable, now)
	if len(upcoming) == 0 {
		helpers.Println("No upcoming class-impacting holidays found for this semester.")
		return
	}

	tableData := [][]string{{"In", "Date", "Day", "Type", "Reason"}}
	for _, holiday := range upcoming {
		reason := holiday.Reason
		if reason == "" {
			reason = "-"
		}
		tableData = append(tableData, []string{
			formatDaysUntil(holiday.Date, now),
			holiday.Date.Format("02 Jan 2006"),
			holiday.Date.Weekday().String(),
			holiday.Status,
			reason,
		})
	}

	helpers.Println()
	helpers.PrintTable(tableData, 0)
}

func resolveHolidaySemester(regNo string, cookies types.Cookies, semChoice int) (types.Semester, error) {
	if semChoice > 0 {
		return helpers.SelectSemester(regNo, cookies, semChoice)
	}

	semesters, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error fetching semester details:", err)
		}
		semesters, err = helpers.GetSemDetailsBackup(cookies, regNo)
		if err != nil {
			return types.Semester{}, err
		}
	}

	if len(semesters) == 0 {
		return types.Semester{}, fmt.Errorf("error fetching semester details or no semesters available. Try logging out and logging back in")
	}

	return LatestSemester(semesters), nil
}

func LatestSemester(semesters []types.Semester) types.Semester {
	return semesters[len(semesters)-1]
}

func resolveHolidayClassGroupID(classGroups [][]string, classGroupChoice int) (string, error) {
	if classGroupChoice > 0 {
		if classGroupChoice > len(classGroups) {
			return "", fmt.Errorf("invalid class group selection")
		}
		return classGroups[classGroupChoice-1][1], nil
	}
	return classGroups[0][1], nil
}

func fetchCalendarEntries(regNo string, cookies types.Cookies, semester types.Semester, classGroupID string, dateList []string, loc *time.Location) []CalendarDayEntry {
	var entries []CalendarDayEntry

	for _, calendarDate := range dateList {
		month, year, err := parseCalendarMonthYear(calendarDate)
		if err != nil {
			if debug.Debug {
				helpers.Println(err)
			}
			continue
		}

		url := "https://vtop.vit.ac.in/vtop/processViewCalendar"
		payloadMap := map[string]string{
			"_csrf":        cookies.CSRF,
			"calDate":      calendarDate,
			"semSubId":     semester.SemID,
			"classGroupId": classGroupID,
			"authorizedID": regNo,
			"x":            fmt.Sprintf("%d", time.Now().Unix()),
		}
		formData := helpers.FormatBodyData(payloadMap)
		bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, formData, "POST", "")
		if err != nil {
			if debug.Debug {
				helpers.Println(err)
			}
			continue
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
		if err != nil {
			if debug.Debug {
				helpers.Println(err)
			}
			continue
		}

		entries = append(entries, extractCalendarEntriesFromDoc(doc, month, year, loc)...)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})
	return entries
}

func parseCalendarMonthYear(calendarDate string) (time.Month, int, error) {
	parts := strings.Split(calendarDate, "-")
	if len(parts) != 3 {
		return time.January, 0, fmt.Errorf("invalid calendar date: %s", calendarDate)
	}

	monthMap := map[string]time.Month{
		"JAN": time.January,
		"FEB": time.February,
		"MAR": time.March,
		"APR": time.April,
		"MAY": time.May,
		"JUN": time.June,
		"JUL": time.July,
		"AUG": time.August,
		"SEP": time.September,
		"OCT": time.October,
		"NOV": time.November,
		"DEC": time.December,
	}

	month, ok := monthMap[strings.ToUpper(parts[1])]
	if !ok {
		return time.January, 0, fmt.Errorf("invalid month in calendar date: %s", calendarDate)
	}

	year, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.January, 0, fmt.Errorf("invalid year in calendar date: %s", calendarDate)
	}

	return month, year, nil
}

func extractCalendarEntriesFromDoc(doc *goquery.Document, month time.Month, year int, loc *time.Location) []CalendarDayEntry {
	var entries []CalendarDayEntry

	cells := doc.Find("table.calendar-table td")
	if cells.Length() == 0 {
		cells = doc.Find("td")
	}

	cells.Each(func(i int, cell *goquery.Selection) {
		entry, ok := ParseCalendarEntry(cell, month, year, loc)
		if ok {
			entries = append(entries, entry)
		}
	})

	return entries
}

func ParseCalendarEntry(cell *goquery.Selection, month time.Month, year int, loc *time.Location) (CalendarDayEntry, bool) {
	var (
		day        int
		status     string
		reasonBits []string
		dayOrder   string
	)

	cell.Find("span").Each(func(i int, span *goquery.Selection) {
		text := strings.TrimSpace(span.Text())
		if text == "" {
			return
		}

		lowerText := strings.ToLower(text)
		if n, err := strconv.Atoi(text); err == nil {
			if day == 0 {
				day = n
			}
			return
		}

		if parsedDayOrder := extractCalendarDayOrder(lowerText); parsedDayOrder != "" {
			dayOrder = parsedDayOrder
			return
		}

		switch {
		case strings.Contains(lowerText, "no instructional day"):
			status = "No Instructional Day"
		case strings.Contains(lowerText, "holiday"):
			status = "Holiday"
		case strings.Contains(lowerText, "exam"):
			status = "Exam Day"
		case strings.Contains(lowerText, "instructional day"):
			status = "Instructional Day"
		default:
			if note := trimCalendarNote(text); note != "" {
				reasonBits = append(reasonBits, note)
			}
		}
	})

	if day == 0 {
		return CalendarDayEntry{}, false
	}

	return CalendarDayEntry{
		Date:     time.Date(year, month, day, 0, 0, 0, 0, loc),
		Status:   status,
		Reason:   strings.Join(reasonBits, "; "),
		DayOrder: dayOrder,
	}, true
}

func extractCalendarDayOrder(text string) string {
	match := workingSatDayOrderRegex.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return normalizeDayName(match[1])
}

func normalizeDayName(day string) string {
	if day == "" {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(day))
	return strings.ToUpper(lower[:1]) + lower[1:]
}

func trimCalendarNote(text string) string {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimPrefix(trimmed, "(")
	trimmed = strings.TrimSuffix(trimmed, ")")
	return strings.TrimSpace(trimmed)
}

func FilterUpcomingClassImpactingHolidays(entries []CalendarDayEntry, timetable map[string][]types.Class, now time.Time) []UpcomingHoliday {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var holidays []UpcomingHoliday

	for _, entry := range entries {
		if entry.Date.Before(today) {
			continue
		}
		if entry.Status != "Holiday" && entry.Status != "No Instructional Day" {
			continue
		}

		scheduleDay := entry.DayOrder
		if scheduleDay == "" {
			scheduleDay = entry.Date.Weekday().String()
		}
		if len(timetable[scheduleDay]) == 0 {
			continue
		}

		holidays = append(holidays, UpcomingHoliday{
			Date:     entry.Date,
			Status:   entry.Status,
			Reason:   entry.Reason,
			Schedule: scheduleDay,
		})
	}

	sort.Slice(holidays, func(i, j int) bool {
		return holidays[i].Date.Before(holidays[j].Date)
	})
	return holidays
}

func formatDaysUntil(date time.Time, now time.Time) string {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	target := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	days := int(target.Sub(start).Hours() / 24)

	switch days {
	case 0:
		return "Today"
	case 1:
		return "1 day"
	default:
		return fmt.Sprintf("%d days", days)
	}
}
