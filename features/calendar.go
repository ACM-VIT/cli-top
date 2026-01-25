package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const calColWidth = 21 // "Su Mo Tu We Th Fr Sa " is 21 chars

func PrintCal(regNo string, cookies types.Cookies, sem_choice int, classGrpFlag int) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	semester, err := helpers.SelectSemester(regNo, cookies, sem_choice)
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

	grp_list := getClassGroups(regNo, cookies, semester)
	if len(grp_list) == 0 {
		helpers.Println("No class groups found")
		return
	}

	grp_list = append([][]string{{"CLASS GROUP"}}, grp_list...)
	result := helpers.TableSelector("class group", grp_list, strconv.Itoa(classGrpFlag))
	if result.ExitRequest || !result.Selected {
		helpers.Println("Selection canceled")
		return
	}
	grp := result.Index

	datelist := getDateList(regNo, cookies, semester, grp_list[grp][1])
	if len(datelist) == 0 {
		helpers.Println("No months found")
		return
	}
	processDates(regNo, cookies, semester, grp_list[grp][1], datelist, 1)
}

func getClassGroups(regNo string, cookies types.Cookies, semester types.Semester) [][]string {
	url := "https://vtop.vit.ac.in/vtop/getDateForSemesterPreview"
	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getDateForSemesterPreview",
		"semSubId":      semester.SemID,
		"authorizedID":  regNo,
		"x":             fmt.Sprintf("%d", time.Now().Unix()),
	}
	formData := helpers.FormatBodyData(payloadMap)
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, formData, "POST", "")
	if err != nil && debug.Debug {
		helpers.Println(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		helpers.Println(err)
	}
	return extractclassgrp(doc)
}

func getDateList(regNo string, cookies types.Cookies, semester types.Semester, grp string) []string {
	url := "https://vtop.vit.ac.in/vtop/getListForSemester"
	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getListForSemester",
		"semSubId":      semester.SemID,
		"classGroupId":  grp,
		"authorizedID":  regNo,
		"x":             fmt.Sprintf("%d", time.Now().Unix()),
	}
	formData := helpers.FormatBodyData(payloadMap)
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, formData, "POST", "")
	if err != nil && debug.Debug {
		helpers.Println(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		helpers.Println(err)
	}
	return readmonths(doc)
}

func processDates(regNo string, cookies types.Cookies, semester types.Semester, grp string, datelist []string, flag int) ([][]int, int, int) {
	var months []string
	year := datelist[0][7:]
	var color_list [][]int

	if flag == 1 {
		helpers.Println("\033[31mRed-Exam Day\033[0m\n\033[34mBlue-Holiday\033[0m\n\033[32mGreen-Instructional Day\033[0m\n\033[33mYellow-Today\033[0m")
	}

	// Map to convert month abbreviations to integers
	monthMap := map[string]int{
		"JAN": 0,
		"FEB": 1,
		"MAR": 2,
		"APR": 3,
		"MAY": 4,
		"JUN": 5,
		"JUL": 6,
		"AUG": 7,
		"SEP": 8,
		"OCT": 9,
		"NOV": 10,
		"DEC": 11,
	}

	isLeapYear := func(y int) bool {
		if y%4 == 0 {
			if y%100 == 0 {
				return y%400 == 0
			}
			return true
		}
		return false
	}

	daysInMonthFor := func(mon int, y int) int {
		switch mon {
		case 0: // Jan
			return 31
		case 1: // Feb
			if isLeapYear(y) {
				return 29
			}
			return 28
		case 2:
			return 31
		case 3:
			return 30
		case 4:
			return 31
		case 5:
			return 30
		case 6:
			return 31
		case 7:
			return 31
		case 8:
			return 30
		case 9:
			return 31
		case 10:
			return 30
		case 11:
			return 31
		default:
			return 30
		}
	}

	// yearInt is still returned for compatibility
	yearInt, err := strconv.Atoi(year)
	if err != nil {
		helpers.Println("Invalid year:", year)
		return nil, -1, -1
	}

	startMonthStr := datelist[0][3:6]
	startMonth, ok := monthMap[startMonthStr]
	if !ok {
		helpers.Println("Invalid month:", startMonthStr)
		return nil, -1, -1
	}

	// Create the nested array with correct days per month/year from each date string
	return_list := [][]int{}
	for i := 0; i < len(datelist); i++ {
		monStr := datelist[i][3:6]
		yStr := datelist[i][7:]
		mon, ok := monthMap[monStr]
		if !ok {
			helpers.Println("Invalid month:", monStr)
			return nil, -1, -1
		}
		yInt, err := strconv.Atoi(yStr)
		if err != nil {
			helpers.Println("Invalid year:", yStr)
			return nil, -1, -1
		}
		monthSize := daysInMonthFor(mon, yInt)
		monthelement := make([]int, monthSize)
		return_list = append(return_list, monthelement)
	}

	type monthDoc struct {
		doc *goquery.Document
		err error
	}
	monthDocs := make([]monthDoc, len(datelist))
	parallel := helpers.NewParallelizer(helpers.DetermineParallelism(len(datelist)))

	for idx, date := range datelist {
		idx := idx
		date := date
		parallel.Go(func() {
			url := "https://vtop.vit.ac.in/vtop/processViewCalendar"
			payloadMap := map[string]string{
				"_csrf":        cookies.CSRF,
				"calDate":      date,
				"semSubId":     semester.SemID,
				"classGroupId": grp,
				"authorizedID": regNo,
				"x":            fmt.Sprintf("%d", time.Now().Unix()),
			}
			formData := helpers.FormatBodyData(payloadMap)
			bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, formData, "POST", "")
			if err != nil {
				if debug.Debug {
					helpers.Println(err)
				}
				monthDocs[idx].err = err
				return
			}
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
			if err != nil {
				if debug.Debug {
					helpers.Println(err)
				}
				monthDocs[idx].err = err
				return
			}
			monthDocs[idx].doc = doc
		})
	}
	parallel.Wait()

	for i, date := range datelist {
		if year != date[7:] {
			addPadding(&color_list)
			if flag == 1 && len(months) > 0 {
				renderMonths(months, year, color_list)
			}
			months = []string{}
			color_list = [][]int{}
			year = ""
		}
		if monthDocs[i].doc == nil {
			continue
		}
		typeOfDay := extractTypeOfDay(monthDocs[i].doc, return_list[i])
		color_list = append(color_list, typeOfDay)
		months = append(months, date[3:6])
		year = date[7:]
	}

	addPadding(&color_list)
	if flag == 1 && len(months) > 0 {
		renderMonths(months, year, color_list)
	}

	return return_list, startMonth, yearInt
}

func addPadding(color_list *[][]int) {
	for i := range *color_list {
		padding := (7 - (len((*color_list)[i]) % 7)) % 7
		for j := 0; j < padding; j++ {
			(*color_list)[i] = append((*color_list)[i], 0)
		}
	}
}

func extractTypeOfDay(doc *goquery.Document, arr []int) []int {
	var typeOfDay []int
	count := -1

	doc.Find("td").Each(func(i int, s *goquery.Selection) {
		k := 0
		t := 0
		s.Find("span").Each(func(j int, span *goquery.Selection) {
			txt := strings.TrimSpace(span.Text())
			if len(txt) == 0 {
				return
			}

			if _, err := strconv.Atoi(txt); err == nil {
				count++
			}

			if strings.Contains(txt, "Holiday") {
				t = -1
				k = 1
			} else if strings.Contains(txt, "No Instructional Day") {
				t = -1
				k = 1
			} else if strings.Contains(txt, "Exam") {
				t = -1
				k = 2
			} else if strings.Contains(txt, "Instructional Day") {
				t = 0
				k = 3
			} else if strings.Contains(txt, "Day Order") {
				k = 3
				switch {
				case strings.Contains(txt, "Monday"):
					t = 1
				case strings.Contains(txt, "Tuesday"):
					t = 2
				case strings.Contains(txt, "Wednesday"):
					t = 3
				case strings.Contains(txt, "Thursday"):
					t = 4
				case strings.Contains(txt, "Friday"):
					t = 5
				default:
					t = 0
				}
			} else {
				t = -1
				k = 1
			}
		})

		typeOfDay = append(typeOfDay, k)

		if count >= 0 && count < len(arr) {
			arr[count] = t
		}
	})

	return typeOfDay
}

func renderMonths(months []string, year string, nestedColour [][]int) {
	calendars := make([][]string, len(months))
	maxRows := 0

	for i, month := range months {
		calendars[i] = generateCalendarLines(month, nestedColour[i])
		if len(calendars[i]) > maxRows {
			maxRows = len(calendars[i])
		}
	}

	totalWidth := len(months)*calColWidth + (len(months)-1)*4
	leftPad := 0
	if totalWidth > len(year) {
		leftPad = (totalWidth - len(year)) / 2
	}
	helpers.Println(strings.Repeat(" ", leftPad) + year)
	helpers.Println()

	blank := fmt.Sprintf("%-*s", calColWidth, "")

	for row := 0; row < maxRows; row++ {
		for i := 0; i < len(months); i++ {
			if row < len(calendars[i]) {
				helpers.Print(calendars[i][row])
			} else {
				helpers.Print(blank)
			}
			if i < len(months)-1 {
				helpers.Print("    ")
			}
		}
		helpers.Println()
	}
}

func generateCalendarLines(month string, colour []int) []string {
	var lines []string
	now := time.Now()
	todayMonth := strings.ToUpper(now.Month().String()[:3])
	todayDay := now.Day()

	if len(colour) < 42 {
		pad := make([]int, 42-len(colour))
		colour = append(colour, pad...)
	} else if len(colour) > 42 {
		colour = colour[:42]
	}

	monthHeader := month
	left := 0
	if calColWidth > len(monthHeader) {
		left = (calColWidth - len(monthHeader)) / 2
	}
	monthHeader = strings.Repeat(" ", left) + monthHeader
	if len(monthHeader) < calColWidth {
		monthHeader = fmt.Sprintf("%-*s", calColWidth, monthHeader)
	} else if len(monthHeader) > calColWidth {
		monthHeader = monthHeader[:calColWidth]
	}

	lines = append(lines, monthHeader)
	lines = append(lines, "Su Mo Tu We Th Fr Sa ")

	var currentLine strings.Builder
	dayOfMonth := 1

	for i, day := range colour {
		if i%7 == 0 && i != 0 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}
		if day == 0 {
			currentLine.WriteString("   ")
		} else {
			var color string
			if month == todayMonth && dayOfMonth == todayDay {
				color = helpers.Yellow
			} else {
				color = helpers.Reset
				switch day {
				case 1:
					color = helpers.Blue
				case 2:
					color = helpers.Red
				case 3:
					color = helpers.Green
				}
			}
			currentLine.WriteString(fmt.Sprintf("%s%2d%s ", color, dayOfMonth, helpers.Reset))
			dayOfMonth++
		}
	}
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	// Ensure all lines are exactly calColWidth chars
	for i := range lines {
		if len(lines[i]) < calColWidth {
			lines[i] = fmt.Sprintf("%-*s", calColWidth, lines[i])
		} else if len(lines[i]) > calColWidth {
			lines[i] = lines[i][:calColWidth]
		}
	}

	return lines
}

func readmonths(doc *goquery.Document) []string {
	var monthlist []string
	doc.Find("a.btn.btn-md.btn-primary").Each(func(i int, s *goquery.Selection) {
		onclick, exists := s.Attr("onclick")
		if exists {
			start := strings.Index(onclick, "'") + 1
			end := strings.Index(onclick[start:], "'") + start
			date := onclick[start:end]
			monthlist = append(monthlist, date)
		}
	})
	return monthlist
}

func extractclassgrp(doc *goquery.Document) [][]string {
	var grp_list [][]string
	doc.Find("select#classGroupId").Each(func(i int, s *goquery.Selection) {
		s.Find("option").Each(func(i int, option *goquery.Selection) {
			value, _ := option.Attr("value")
			text := strings.TrimSpace(option.Text())
			grp_list = append(grp_list, []string{text, value})
		})
	})
	grp_list = grp_list[1:]
	return grp_list
}
