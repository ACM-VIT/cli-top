package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	calendarPreviewURL      = "https://vtop.vit.ac.in/vtop/academics/common/CalendarPreview"
	getDateForSemPreviewURL = "https://vtop.vit.ac.in/vtop/getDateForSemesterPreview"
	processViewCalendarURL  = "https://vtop.vit.ac.in/vtop/processViewCalendar"
	classGroupID            = "COMB"
	saturdayIndex           = 6
)

var workingSatDayOrderRegex = regexp.MustCompile(`(?i)(monday|tuesday|wednesday|thursday|friday)\s*day order`)

type WorkingSaturday struct {
	Date     time.Time
	DayOrder string
}

func fetchWorkingSaturdays(regNo string, cookies types.Cookies, semSubID, classGroupID string) []WorkingSaturday {
	var result []WorkingSaturday

	client := helpers.GetHTTPClient()
	locIndia := time.FixedZone("IST", 5*3600+1800)
	now := time.Now().In(locIndia)

	payload0 := map[string]string{
		"verifyMenu":   "true",
		"authorizedID": regNo,
		"_csrf":        cookies.CSRF,
		"nocache":      fmt.Sprintf("%d", time.Now().Unix()),
	}
	form0 := helpers.FormatBodyData(payload0)
	_ = form0
	body0, _, err0 := helpers.FetchReqClient(client, regNo, cookies, calendarPreviewURL, "https://vtop.vit.ac.in/vtop/content", []byte(form0), "POST", "application/x-www-form-urlencoded")
	_ = body0
	_ = err0

	payloadVerify := map[string]string{
		"menuCode":     "CalendarPreview",
		"authorizedID": regNo,
		"_csrf":        cookies.CSRF,
		"nocache":      fmt.Sprintf("%d", time.Now().Unix()),
	}
	formVerify := helpers.FormatBodyData(payloadVerify)
	_ = formVerify
	bodyVerify, _, errVerify := helpers.FetchReqClient(client, regNo, cookies, calendarPreviewURL, "https://vtop.vit.ac.in/vtop/content", []byte(formVerify), "POST", "application/x-www-form-urlencoded")
	_ = bodyVerify
	_ = errVerify

	payload1 := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getDateForSemesterPreview",
		"semSubId":      semSubID,
		"authorizedID":  regNo,
		"x":             fmt.Sprintf("%d", time.Now().Unix()),
	}
	form1 := helpers.FormatBodyData(payload1)
	_ = form1
	body1, _, err1 := helpers.FetchReqClient(client, regNo, cookies, getDateForSemPreviewURL, "", []byte(form1), "POST", "application/x-www-form-urlencoded")
	_ = body1
	_ = err1

	monthAbr := strings.ToUpper(now.Format("Jan"))[:3]
	calDate := fmt.Sprintf("01-%s-%04d", monthAbr, now.Year())
	payload2 := map[string]string{
		"_csrf":        cookies.CSRF,
		"calDate":      calDate,
		"semSubId":     semSubID,
		"classGroupId": classGroupID,
		"authorizedID": regNo,
		"x":            fmt.Sprintf("%d", time.Now().Unix()),
	}
	form2 := helpers.FormatBodyData(payload2)
	body3, _, err3 := helpers.FetchReqClient(client, regNo, cookies, processViewCalendarURL, "", []byte(form2), "POST", "application/x-www-form-urlencoded")
	if err3 != nil {
		return result
	}

	return parseWorkingSaturdaysFromCalendar(body3, now, locIndia)
}

func parseWorkingSaturdaysFromCalendar(body []byte, now time.Time, loc *time.Location) []WorkingSaturday {
	var result []WorkingSaturday
	z := html.NewTokenizer(bytes.NewReader(body))

	var (
		inTable    bool
		tableDepth int
		inRow      bool
		inTD       bool
		tdIndex    int
		cellText   strings.Builder
	)

	for {
		switch z.Next() {
		case html.ErrorToken:
			return result
		case html.StartTagToken, html.SelfClosingTagToken:
			tagName, hasAttr := z.TagName()
			switch string(tagName) {
			case "table":
				if !inTable {
					classAttr := ""
					if hasAttr {
						for {
							key, val, more := z.TagAttr()
							if string(key) == "class" {
								classAttr = string(val)
							}
							if !more {
								break
							}
						}
					}
					if strings.Contains(classAttr, "calendar-table") {
						inTable = true
						tableDepth = 1
					}
				} else {
					tableDepth++
				}
			case "tr":
				if inTable {
					inRow = true
					tdIndex = -1
				}
			case "td":
				if inRow {
					inTD = true
					tdIndex++
					if tdIndex == saturdayIndex {
						cellText.Reset()
					}
				}
			}
		case html.TextToken:
			if inTD && tdIndex == saturdayIndex {
				cellText.Write(z.Text())
			}
		case html.EndTagToken:
			tagName, _ := z.TagName()
			switch string(tagName) {
			case "td":
				if inTD && tdIndex == saturdayIndex {
					text := cleanTimeTableText(cellText.String())
					if ws, ok := parseWorkingSaturdayCell(text, now, loc); ok {
						result = append(result, ws)
					}
				}
				inTD = false
			case "tr":
				inRow = false
			case "table":
				if inTable {
					tableDepth--
					if tableDepth <= 0 {
						inTable = false
					}
				}
			}
		}
	}
}

func parseWorkingSaturdayCell(text string, now time.Time, loc *time.Location) (WorkingSaturday, bool) {
	if text == "" {
		return WorkingSaturday{}, false
	}
	textLower := strings.ToLower(text)
	if strings.Contains(textLower, "fresher") {
		return WorkingSaturday{}, false
	}

	dayInt := firstInt(textLower)
	if dayInt == 0 {
		return WorkingSaturday{}, false
	}

	match := workingSatDayOrderRegex.FindStringSubmatch(textLower)
	if len(match) < 2 {
		return WorkingSaturday{}, false
	}

	dayOrder := strings.Title(match[1])
	wsDate := time.Date(now.Year(), now.Month(), dayInt, 0, 0, 0, 0, loc)
	return WorkingSaturday{Date: wsDate, DayOrder: dayOrder}, true
}

func firstInt(text string) int {
	n := 0
	inNumber := false
	for _, r := range text {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
			inNumber = true
		} else if inNumber {
			break
		}
	}
	if !inNumber {
		return 0
	}
	return n
}

const (
	TimeTableTableSelector = "table.table"
	TimeTableRowsSelector  = "tbody tr"
	TimeTableCellSelector  = "td"
	CourseCellIndex        = 2
	SlotCellIndex          = 7
)

var schedule = map[string]map[string][]string{
	"A1": {
		"Monday":    []string{"08:00", "08:50"},
		"Wednesday": []string{"09:00", "09:50"},
	},
	"B1": {
		"Tuesday":  []string{"08:00", "08:50"},
		"Thursday": []string{"09:00", "09:50"},
	},
	"C1": {
		"Wednesday": []string{"08:00", "08:50"},
		"Friday":    []string{"09:00", "09:50"},
	},
	"D1": {
		"Monday":   []string{"10:00", "10:50"},
		"Thursday": []string{"08:00", "08:50"},
	},
	"E1": {
		"Tuesday": []string{"10:00", "10:50"},
		"Friday":  []string{"08:00", "08:50"},
	},
	"F1": {
		"Monday":    []string{"09:00", "09:50"},
		"Wednesday": []string{"10:00", "10:50"},
	},
	"G1": {
		"Tuesday":  []string{"09:00", "09:50"},
		"Thursday": []string{"10:00", "10:50"},
	},
	"TA1": {
		"Friday": []string{"10:00", "10:50"},
	},
	"TB1": {
		"Monday": []string{"11:00", "11:50"},
	},
	"TC1": {
		"Tuesday": []string{"11:00", "11:50"},
	},
	"TD1": {
		"Friday": []string{"12:00", "12:50"},
	},
	"TE1": {
		"Thursday": []string{"11:00", "11:50"},
	},
	"TF1": {
		"Friday": []string{"11:00", "11:50"},
	},
	"TG1": {
		"Monday": []string{"12:00", "12:50"},
	},
	"TAA1": {
		"Tuesday": []string{"12:00", "12:50"},
	},
	"TCC1": {
		"Thursday": []string{"12:00", "12:50"},
	},
	"A2": {
		"Monday":    []string{"14:00", "14:50"},
		"Wednesday": []string{"15:00", "15:50"},
	},
	"B2": {
		"Tuesday":  []string{"14:00", "14:50"},
		"Thursday": []string{"15:00", "15:50"},
	},
	"C2": {
		"Wednesday": []string{"14:00", "14:50"},
		"Friday":    []string{"15:00", "15:50"},
	},
	"D2": {
		"Monday":   []string{"16:00", "16:50"},
		"Thursday": []string{"14:00", "14:50"},
	},
	"E2": {
		"Tuesday": []string{"16:00", "16:50"},
		"Friday":  []string{"14:00", "14:50"},
	},
	"F2": {
		"Monday":    []string{"15:00", "15:50"},
		"Wednesday": []string{"16:00", "16:50"},
	},
	"G2": {
		"Tuesday":  []string{"15:00", "15:50"},
		"Thursday": []string{"16:00", "16:50"},
	},
	"TA2": {
		"Friday": []string{"16:00", "16:50"},
	},
	"TB2": {
		"Monday": []string{"17:00", "17:50"},
	},
	"TC2": {
		"Tuesday": []string{"17:00", "17:50"},
	},
	"TD2": {
		"Wednesday": []string{"17:00", "17:50"},
	},
	"TE2": {
		"Thursday": []string{"17:00", "17:50"},
	},
	"TF2": {
		"Friday": []string{"17:00", "17:50"},
	},
	"TG2": {
		"Monday": []string{"18:00", "18:50"},
	},
	"TAA2": {
		"Tuesday": []string{"18:00", "18:50"},
	},
	"TBB2": {
		"Wednesday": []string{"18:00", "18:50"},
	},
	"TCC2": {
		"Thursday": []string{"18:00", "18:50"},
	},
	"TDD2": {
		"Friday": []string{"18:00", "18:50"},
	},
	"L1+L2": {
		"Monday": []string{"08:00", "09:40"},
	},
	"L3+L4": {
		"Monday": []string{"09:50", "11:30"},
	},
	"L5+L6": {
		"Monday": []string{"11:40", "13:20"},
	},
	"L7+L8": {
		"Tuesday": []string{"08:00", "09:40"},
	},
	"L9+L10": {
		"Tuesday": []string{"09:50", "11:30"},
	},
	"L11+L12": {
		"Tuesday": []string{"11:40", "13:20"},
	},
	"L13+L14": {
		"Wednesday": []string{"08:00", "09:40"},
	},
	"L15+L16": {
		"Wednesday": []string{"09:50", "11:30"},
	},
	"L17+L18": {
		"Wednesday": []string{"11:40", "13:20"},
	},
	"L19+L20": {
		"Thursday": []string{"08:00", "09:40"},
	},
	"L21+L22": {
		"Thursday": []string{"09:50", "11:30"},
	},
	"L23+L24": {
		"Thursday": []string{"11:40", "13:20"},
	},
	"L25+L26": {
		"Friday": []string{"08:00", "09:40"},
	},
	"L27+L28": {
		"Friday": []string{"09:50", "11:30"},
	},
	"L29+L30": {
		"Friday": []string{"11:40", "13:20"},
	},
	"L31+L32": {
		"Monday": []string{"14:00", "15:40"},
	},
	"L33+L34": {
		"Monday": []string{"15:50", "17:30"},
	},
	"L35+L36": {
		"Monday": []string{"17:40", "19:20"},
	},
	"L37+L38": {
		"Tuesday": []string{"14:00", "15:40"},
	},
	"L39+L40": {
		"Tuesday": []string{"15:50", "17:30"},
	},
	"L41+L42": {
		"Tuesday": []string{"17:40", "19:20"},
	},
	"L43+L44": {
		"Wednesday": []string{"14:00", "15:40"},
	},
	"L45+L46": {
		"Wednesday": []string{"15:50", "17:30"},
	},
	"L47+L48": {
		"Wednesday": []string{"17:40", "19:20"},
	},
	"L49+L50": {
		"Thursday": []string{"14:00", "15:40"},
	},
	"L51+L52": {
		"Thursday": []string{"15:50", "17:30"},
	},
	"L53+L54": {
		"Thursday": []string{"17:40", "19:20"},
	},
	"L55+L56": {
		"Friday": []string{"14:00", "15:40"},
	},
	"L57+L58": {
		"Friday": []string{"15:50", "17:30"},
	},
	"L59+L60": {
		"Friday": []string{"17:40", "19:20"},
	},
	"V1": {
		"Wednesday": []string{"11:00", "11:50"},
	},
	"V2": {
		"Wednesday": []string{"12:00", "12:50"},
	},
	"V3": {
		"Monday": []string{"19:00", "19:50"},
	},
	"V4": {
		"Tuesday": []string{"19:00", "19:50"},
	},
	"V5": {
		"Wednesday": []string{"19:00", "19:50"},
	},
	"V6": {
		"Thursday": []string{"19:00", "19:50"},
	},
	"V7": {
		"Friday": []string{"19:00", "19:50"},
	},
	"V8": {
		"Saturday": []string{"08:00", "08:50"},
	},
	"X11": {
		"Saturday": []string{"09:00", "09:50"},
		"Sunday":   []string{"11:00", "11:50"},
	},
	"X12": {
		"Saturday": []string{"10:00", "10:50"},
		"Sunday":   []string{"12:00", "12:50"},
	},
	"Y11": {
		"Saturday": []string{"11:00", "11:50"},
		"Sunday":   []string{"09:00", "09:50"},
	},
	"Y12": {
		"Saturday": []string{"12:00", "12:50"},
		"Sunday":   []string{"10:00", "10:50"},
	},
	"X21": {
		"Saturday": []string{"14:00", "14:50"},
		"Sunday":   []string{"16:00", "16:50"},
	},
	"Z21": {
		"Saturday": []string{"15:00", "15:50"},
		"Sunday":   []string{"15:00", "15:50"},
	},
	"Y21": {
		"Saturday": []string{"16:00", "16:50"},
		"Sunday":   []string{"14:00", "14:50"},
	},
	"W21": {
		"Saturday": []string{"17:00", "17:50"},
		"Sunday":   []string{"17:00", "17:50"},
	},
	"W22": {
		"Saturday": []string{"18:00", "18:50"},
		"Sunday":   []string{"18:00", "18:50"},
	},
	"V9": {
		"Saturday": []string{"19:00", "19:50"},
	},
	"V10": {
		"Sunday": []string{"08:00", "08:50"},
	},
	"V11": {
		"Sunday": []string{"19:00", "19:50"},
	},
	"L71+L72": {
		"Saturday": []string{"08:00", "09:40"},
	},
	"L73+L74": {
		"Saturday": []string{"09:50", "11:30"},
	},
	"L75+L76": {
		"Saturday": []string{"11:40", "13:20"},
	},
	"L77+L78": {
		"Saturday": []string{"14:00", "15:40"},
	},
	"L79+L80": {
		"Saturday": []string{"15:50", "17:30"},
	},
	"L81+L82": {
		"Saturday": []string{"17:40", "19:20"},
	},
	"L83+L84": {
		"Sunday": []string{"08:00", "09:40"},
	},
	"L85+L86": {
		"Sunday": []string{"09:50", "11:30"},
	},
	"L87+L88": {
		"Sunday": []string{"11:40", "13:20"},
	},
	"L89+L90": {
		"Sunday": []string{"14:00", "15:40"},
	},
	"L91+L92": {
		"Sunday": []string{"15:50", "17:30"},
	},
	"L93+L94": {
		"Sunday": []string{"17:40", "19:20"},
	},
}

// func updateTimetableWithWorkingSaturdays(timetable map[string][]types.Class, workingSaturdays []WorkingSaturday) {
// 	for _, ws := range workingSaturdays {
// 		classes, ok := timetable[ws.DayOrder]
// 		if !ok {
// 			continue
// 		}
// 		for _, c := range classes {
// 			timetable["Saturday"] = append(timetable["Saturday"], types.Class{
// 				Subject:   c.Subject,
// 				Slot:      c.Slot,
// 				Venue:     c.Venue,
// 				StartTime: c.StartTime,
// 				EndTime:   c.EndTime,
// 				DayOrder:  ws.DayOrder,
// 			})
// 		}
// 	}
// 	if _, exists := timetable["Saturday"]; exists {
// 		sort.Slice(timetable["Saturday"], func(i, j int) bool {
// 			return timetable["Saturday"][i].StartTime < timetable["Saturday"][j].StartTime
// 		})
// 	}
// }

func GetTimeTable(regNo string, cookies types.Cookies, sem_choice int) {
	if !helpers.ValidateLogin(cookies) {
		return
	}
	// Fetch all semesters to determine if user is a fresher (only one semester)
	allSems, _ := helpers.GetSemDetails(cookies, regNo)
	semester, err := helpers.SelectSemester(regNo, cookies, sem_choice)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return
	}

	grp_list := getClassGroups(regNo, cookies, semester)
	wsChan := make(chan []WorkingSaturday, 1)
	go func() {
		// Only fetch working Saturdays for the selected semester
		wsChan <- fetchWorkingSaturdays(regNo, cookies, semester.SemID, classGroupID)
	}()

	url := "https://vtop.vit.ac.in/vtop/processViewTimeTable"
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, "UTC", "POST", "")
	if err != nil && debug.Debug {
		helpers.Println(err)
	}

	grp_list = getClassGroups(regNo, cookies, semester)
	datelist := getDateList(regNo, cookies, semester, grp_list[1][1])
	semSec, month, year := processDates(regNo, cookies, semester, grp_list[1][1], datelist, 0)
	courseMap := getCourseNameFromHTML(bodyText)
	timetable := makeTT(schedule, courseMap)

	if _, exists := timetable["Saturday"]; !exists {
		timetable["Saturday"] = []types.Class{}
	}

	workingSats := <-wsChan
	// If user is a fresher (only one semester), only show working Saturdays for that semester
	if len(allSems) == 1 {
		// Already filtered by selected semester, nothing to do
		updateTimetableWithWorkingSaturdays(timetable, workingSats)
		printTT(timetable, workingSats)
	} else {
		// For non-freshers, also only show working Saturdays for the selected semester
		updateTimetableWithWorkingSaturdays(timetable, workingSats)
		printTT(timetable, workingSats)
	}

	if os.Getenv("CLI_TOP_PROXY_MODE") == "1" {
		return
	}

	icsDir, err := helpers.GetOrCreateDownloadDir(filepath.Join("Other Downloads", "ICS File"))
	if err != nil {
		helpers.Println("Error creating ICS file directory:", err)
		return
	}
	icsPath := filepath.Join(icsDir, "CLI-TOP_Timetable.ics")
	icsContent := makeISC(timetable, semSec, month, year, workingSats)
	if err := writetoFile(icsPath, icsContent); err != nil {
		helpers.Println("Error generating ICS file:", err)
	} else {
		if link, err := helpers.UploadICSFile(icsPath, helpers.CalendarServerURL); err == nil {
			helpers.Println("\nICS file generated and saved successfully.")
			helpers.GenerateCalendarImportLinks(link, "Timetable")
		} else {
			helpers.Println("Error uploading ICS file; please import manually.")
		}
	}
}

func makeTT(schedule map[string]map[string][]string, courseMap map[string]types.SubjectTime) map[string][]types.Class {
	timetable := make(map[string][]types.Class)
	for subj, st := range courseMap {
		for i := 0; i < len(st.Slot); i++ {
			slot := st.Slot[i]
			for dayName, times := range schedule[slot] {
				timetable[dayName] = append(timetable[dayName], types.Class{
					Subject:   subj,
					Slot:      slot,
					Venue:     st.Venue,
					StartTime: times[0],
					EndTime:   times[1],
					DayOrder:  "",
				})
			}
		}
	}
	return timetable
}

func makeISC(timetable map[string][]types.Class, semSection [][]int, startMonth int, startYear int, workingSaturdays []WorkingSaturday) string {
	icsContent := "BEGIN:VCALENDAR\nVERSION:2.0\nCALSCALE:GREGORIAN\nX-WR-CALNAME:CLI-TOP Timetable\n"
	startMonth++
	startDate := time.Date(startYear, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	wsMap := make(map[string]string)
	for _, ws := range workingSaturdays {
		wsMap[ws.Date.Format("2006-01-02")] = ws.DayOrder
	}
	for months := 0; months < len(semSection); months++ {
		for dayofMonth := 0; dayofMonth < len(semSection[months]); dayofMonth++ {
			if semSection[months][dayofMonth] == -1 {
				startDate = startDate.AddDate(0, 0, 1)
				continue
			}
			dateKey := startDate.Format("2006-01-02")
			var day string
			if wd, isWS := wsMap[dateKey]; isWS {
				day = wd
				for _, class := range timetable[day] {
					startDateTime := fmt.Sprintf("%sT%s00", startDate.Format("20060102"), strings.ReplaceAll(class.StartTime, ":", ""))
					endDateTime := fmt.Sprintf("%sT%s00", startDate.Format("20060102"), strings.ReplaceAll(class.EndTime, ":", ""))
					icsContent += fmt.Sprintf("BEGIN:VEVENT\n"+
						"SUMMARY:%s (Working Saturday - %s schedule)\n"+
						"DTSTART;TZID=Asia/Kolkata:%s\n"+
						"DTEND;TZID=Asia/Kolkata:%s\n"+
						"LOCATION:%s\n"+
						"DESCRIPTION:Slot: %s\\nWorking Saturday following %s schedule\n"+
						"BEGIN:VALARM\n"+
						"TRIGGER:-PT5M\n"+
						"ACTION:DISPLAY\n"+
						"END:VALARM\n"+
						"END:VEVENT\n",
						class.Subject, day, startDateTime, endDateTime, class.Venue, class.Slot, day)
				}
			} else if semSection[months][dayofMonth] == 0 {
				day = startDate.Format("Monday")
				for _, class := range timetable[day] {
					startDateTime := fmt.Sprintf("%sT%s00", startDate.Format("20060102"), strings.ReplaceAll(class.StartTime, ":", ""))
					endDateTime := fmt.Sprintf("%sT%s00", startDate.Format("20060102"), strings.ReplaceAll(class.EndTime, ":", ""))
					icsContent += fmt.Sprintf("BEGIN:VEVENT\n"+
						"SUMMARY:%s\n"+
						"DTSTART;TZID=Asia/Kolkata:%s\n"+
						"DTEND;TZID=Asia/Kolkata:%s\n"+
						"LOCATION:%s\n"+
						"DESCRIPTION:Slot: %s\n"+
						"BEGIN:VALARM\n"+
						"TRIGGER:-PT5M\n"+
						"ACTION:DISPLAY\n"+
						"END:VALARM\n"+
						"END:VEVENT\n",
						class.Subject, startDateTime, endDateTime, class.Venue, class.Slot)
				}
			} else {
				dayInt := semSection[months][dayofMonth]
				day = getDayName(time.Weekday(dayInt))
				for _, class := range timetable[day] {
					startDateTime := fmt.Sprintf("%sT%s00", startDate.Format("20060102"), strings.ReplaceAll(class.StartTime, ":", ""))
					endDateTime := fmt.Sprintf("%sT%s00", startDate.Format("20060102"), strings.ReplaceAll(class.EndTime, ":", ""))
					icsContent += fmt.Sprintf("BEGIN:VEVENT\n"+
						"SUMMARY:%s\n"+
						"DTSTART;TZID=Asia/Kolkata:%s\n"+
						"DTEND;TZID=Asia/Kolkata:%s\n"+
						"LOCATION:%s\n"+
						"DESCRIPTION:Slot: %s\n"+
						"BEGIN:VALARM\n"+
						"TRIGGER:-PT5M\n"+
						"ACTION:DISPLAY\n"+
						"END:VALARM\n"+
						"END:VEVENT\n",
						class.Subject, startDateTime, endDateTime, class.Venue, class.Slot)
				}
			}
			startDate = startDate.AddDate(0, 0, 1)
		}
	}
	icsContent += "END:VCALENDAR"
	return icsContent
}

func getDayName(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "Monday"
	case time.Tuesday:
		return "Tuesday"
	case time.Wednesday:
		return "Wednesday"
	case time.Thursday:
		return "Thursday"
	case time.Friday:
		return "Friday"
	case time.Saturday:
		return "Saturday"
	case time.Sunday:
		return "Sunday"
	default:
		return ""
	}
}

func applyColor(text string, colorCode string) string {
	return colorCode + text + helpers.Reset
}

func applyStyle(text string, style string) string {
	return style + text + helpers.Reset
}

const (
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Underline = "\033[4m"
	Blink     = "\033[5m"
	Reverse   = "\033[7m"
	Hidden    = "\033[8m"

	Cyan    = "\033[36m"
	White   = "\033[37m"
	BgBlack = "\033[40m"
)

func writetoFile(filepath string, content string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create ICS file: %v", err)
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write ICS content: %v", err)
	}
	return nil
}

func getCourseNameFromHTML(body []byte) map[string]types.SubjectTime {
	courseMap := make(map[string]types.SubjectTime)

	z := html.NewTokenizer(bytes.NewReader(body))
	var (
		inTable    bool
		tableDepth int
		inRow      bool
		inTD       bool
		tdIndex    int
		cellText   strings.Builder
		rowCells   []string
	)

	for {
		switch z.Next() {
		case html.ErrorToken:
			if len(courseMap) == 0 {
				helpers.Println("Table with class 'table' not found")
			}
			return courseMap
		case html.StartTagToken, html.SelfClosingTagToken:
			tagName, hasAttr := z.TagName()
			switch string(tagName) {
			case "table":
				if !inTable {
					classAttr := ""
					if hasAttr {
						for {
							key, val, more := z.TagAttr()
							if string(key) == "class" {
								classAttr = string(val)
							}
							if !more {
								break
							}
						}
					}
					if strings.Contains(classAttr, "table") {
						inTable = true
						tableDepth = 1
					}
				} else {
					tableDepth++
				}
			case "tr":
				if inTable {
					inRow = true
					tdIndex = -1
					rowCells = rowCells[:0]
				}
			case "td":
				if inRow {
					inTD = true
					tdIndex++
					cellText.Reset()
				}
			}
		case html.TextToken:
			if inTD {
				cellText.Write(z.Text())
			}
		case html.EndTagToken:
			tagName, _ := z.TagName()
			switch string(tagName) {
			case "td":
				if inTD {
					inTD = false
					rowCells = append(rowCells, cleanTimeTableText(cellText.String()))
				}
			case "tr":
				if inRow {
					inRow = false
					if len(rowCells) <= SlotCellIndex || len(rowCells) <= CourseCellIndex {
						break
					}
					cellText := strings.TrimSpace(rowCells[CourseCellIndex])
					parts := strings.SplitN(cellText, " - ", 2)
					var courseName string
					if len(parts) == 2 {
						courseName = strings.TrimSpace(parts[1])
						if idxStart := strings.Index(courseName, "("); idxStart != -1 {
							idxEnd := strings.Index(courseName, ")")
							if idxEnd != -1 && idxEnd > idxStart {
								parenthetical := courseName[idxStart+1 : idxEnd]
								if strings.Contains(strings.ToLower(parenthetical), "embedded") {
									courseName = strings.TrimSpace(courseName[:idxStart]) + strings.TrimSpace(courseName[idxStart-1:])
								} else {
									courseName = strings.TrimSpace(courseName[:idxStart])
								}
							}
						}
					}
					slotText := strings.TrimSpace(rowCells[SlotCellIndex])
					if slotText == "" {
						break
					}
					parts = strings.SplitN(slotText, " - ", 2)
					if len(parts) < 2 {
						break
					}
					newparts := strings.Split(parts[0], "+")
					if slotText[0] == 'L' {
						var joinedParts []string
						for i := 0; i < len(newparts); i += 2 {
							if i+1 < len(newparts) {
								joinedParts = append(joinedParts, strings.Join([]string{newparts[i], newparts[i+1]}, "+"))
							}
						}
						newparts = joinedParts
					}
					courseMap[courseName] = types.SubjectTime{
						Slot:  newparts,
						Venue: strings.TrimSpace(parts[1]),
					}
				}
			case "table":
				if inTable {
					tableDepth--
					if tableDepth <= 0 {
						inTable = false
					}
				}
			}
		}
	}
}

func cleanTimeTableText(s string) string {
	if s == "" {
		return ""
	}
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func updateTimetableWithWorkingSaturdays(timetable map[string][]types.Class, workingSaturdays []WorkingSaturday) {
	for _, ws := range workingSaturdays {
		classes, ok := timetable[ws.DayOrder]
		if !ok || len(classes) == 0 {
			// If no classes on the referenced DayOrder, skip this working Saturday
			continue
		}
		for _, c := range classes {
			duplicate := false
			for _, existing := range timetable["Saturday"] {
				if existing.Subject == c.Subject &&
					existing.Slot == c.Slot &&
					existing.StartTime == c.StartTime &&
					existing.EndTime == c.EndTime &&
					existing.Venue == c.Venue &&
					existing.DayOrder == ws.DayOrder {
					duplicate = true
					break
				}
			}
			if !duplicate {
				timetable["Saturday"] = append(timetable["Saturday"], types.Class{
					Subject:   c.Subject,
					Slot:      c.Slot,
					Venue:     c.Venue,
					StartTime: c.StartTime,
					EndTime:   c.EndTime,
					DayOrder:  ws.DayOrder,
				})
			}
		}
	}
	if _, exists := timetable["Saturday"]; exists {
		sort.Slice(timetable["Saturday"], func(i, j int) bool {
			return timetable["Saturday"][i].StartTime < timetable["Saturday"][j].StartTime
		})
	}
}

func printTT(timetable map[string][]types.Class, workingSaturdays []WorkingSaturday) {
	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	locIndia := time.FixedZone("IST", 5*3600+1800)
	now := time.Now().In(locIndia)
	currentDay := now.Weekday().String()

	currentDayIndex := -1
	for i, day := range daysOfWeek {
		if strings.EqualFold(day, currentDay) {
			currentDayIndex = i
			break
		}
	}

	highlightNextDay := false
	if currentDayIndex != -1 {
		classes, exists := timetable[currentDay]
		if exists && len(classes) > 0 {
			sort.Slice(classes, func(i, j int) bool {
				return classes[i].StartTime < classes[j].StartTime
			})
			lastClass := classes[len(classes)-1]
			lastClassTime, _ := time.Parse("15:04", lastClass.EndTime)
			lastClassToday := time.Date(now.Year(), now.Month(), now.Day(), lastClassTime.Hour(), lastClassTime.Minute(), 0, 0, now.Location())
			if now.After(lastClassToday) {
				highlightNextDay = true
			}
		} else {
			highlightNextDay = true
		}
	}

	dayToHighlight := daysOfWeek[currentDayIndex]
	if highlightNextDay {
		// Find the next day with classes (including Saturday/Sunday)
		for offset := 1; offset <= 7; offset++ {
			nextIdx := (currentDayIndex + offset) % len(daysOfWeek)
			nextDay := daysOfWeek[nextIdx]
			classes, exists := timetable[nextDay]
			if exists && len(classes) > 0 {
				dayToHighlight = nextDay
				break
			}
		}
	}

	var targetSaturdayDate time.Time
	currentWeekday := now.Weekday()
	daysAhead := (int(time.Saturday) - int(currentWeekday) + 7) % 7
	targetSaturdayDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, locIndia).AddDate(0, 0, daysAhead)

	matchingDayOrder := ""
	for _, ws := range workingSaturdays {
		if ws.Date.Year() == targetSaturdayDate.Year() &&
			ws.Date.Month() == targetSaturdayDate.Month() &&
			ws.Date.Day() == targetSaturdayDate.Day() {
			matchingDayOrder = ws.DayOrder
			break
		}
	}

	updateTimetableWithWorkingSaturdays(timetable, workingSaturdays)
	for _, day := range daysOfWeek {
		if day != "Saturday" {
			classes, exists := timetable[day]
			if !exists || len(classes) == 0 {
				continue
			}
			if day == dayToHighlight {
				helpers.Printf("%s\n\n", applyColor(day, Cyan+Bold))
			} else {
				helpers.Printf("%s\n\n", day)
			}
			sort.Slice(classes, func(i, j int) bool {
				return classes[i].StartTime < classes[j].StartTime
			})
			tableData := [][]string{{"Time", "Subject", "Slot", "Venue"}}
			for _, c := range classes {
				row := []string{fmt.Sprintf("%s-%s", c.StartTime, c.EndTime), c.Subject, c.Slot, c.Venue}
				if day == dayToHighlight {
					startT, _ := time.Parse("15:04", c.StartTime)
					endT, _ := time.Parse("15:04", c.EndTime)
					startTime := time.Date(now.Year(), now.Month(), now.Day(), startT.Hour(), startT.Minute(), 0, 0, now.Location())
					endTime := time.Date(now.Year(), now.Month(), now.Day(), endT.Hour(), endT.Minute(), 0, 0, now.Location())
					if now.After(startTime) && now.Before(endTime) {
						for i := range row {
							row[i] = applyColor(row[i], helpers.Green+Bold)
						}
					} else if now.After(endTime) {
						for i := range row {
							row[i] = applyColor(row[i], White+Dim)
						}
					} else {
						for i := range row {
							row[i] = applyStyle(row[i], Bold)
						}
					}
				}
				tableData = append(tableData, row)
			}
			helpers.PrintTable(tableData, 0)
			helpers.Println()
			continue
		}

		satClasses, exists := timetable["Saturday"]
		if !exists || len(satClasses) == 0 {
			continue
		}

		var subset []types.Class
		if matchingDayOrder != "" {
			for _, c := range satClasses {
				if c.DayOrder == matchingDayOrder {
					subset = append(subset, c)
				}
			}
			// Only show working Saturday if there are classes for the referenced DayOrder
			if len(subset) == 0 {
				continue
			}
		} else {
			for _, c := range satClasses {
				if c.DayOrder == "" {
					subset = append(subset, c)
				}
			}
			if len(subset) == 0 {
				continue
			}
		}

		if day == dayToHighlight {
			helpers.Printf("%s\n\n", applyColor(day, Cyan+Bold))
		} else {
			helpers.Printf("%s\n\n", day)
		}

		if matchingDayOrder != "" {
			helpers.Printf("%s %s\n\n", applyColor("Working Saturday", helpers.Yellow),
				applyColor(fmt.Sprintf("(Following %s schedule)", matchingDayOrder), helpers.Yellow))
		}

		sort.Slice(subset, func(i, j int) bool {
			return subset[i].StartTime < subset[j].StartTime
		})
		tableData := [][]string{{"Time", "Subject", "Slot", "Venue"}}
		for _, c := range subset {
			row := []string{fmt.Sprintf("%s-%s", c.StartTime, c.EndTime), c.Subject, c.Slot, c.Venue}
			if day == dayToHighlight {
				startT, _ := time.Parse("15:04", c.StartTime)
				endT, _ := time.Parse("15:04", c.EndTime)
				startTime := time.Date(now.Year(), now.Month(), now.Day(), startT.Hour(), startT.Minute(), 0, 0, now.Location())
				endTime := time.Date(now.Year(), now.Month(), now.Day(), endT.Hour(), endT.Minute(), 0, 0, now.Location())
				if now.After(startTime) && now.Before(endTime) {
					for i := range row {
						row[i] = applyColor(row[i], helpers.Green+Bold)
					}
				} else if now.After(endTime) {
					for i := range row {
						row[i] = applyColor(row[i], White+Dim)
					}
				} else {
					for i := range row {
						row[i] = applyStyle(row[i], Bold)
					}
				}
			}
			tableData = append(tableData, row)
		}
		helpers.PrintTable(tableData, 0)
		helpers.Println()
	}
}
