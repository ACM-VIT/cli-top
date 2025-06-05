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

func PrintCal(regNo string, cookies types.Cookies, sem_choice int, classGrpFlag int) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	semester, err := helpers.SelectSemester(regNo, cookies, sem_choice)
	if err != nil {
		if err.Error() == "selection canceled by user" {
			fmt.Println("Selection canceled")
			return
		}
		if debug.Debug {
			fmt.Println(err)
		}
		fmt.Println("Error selecting semester:", err)
		return
	}

	grp_list := getClassGroups(regNo, cookies, semester)
	if len(grp_list) == 0 {
		fmt.Println("No class groups found")
		return
	}

	grp_list = append([][]string{{"CLASS GROUP"}}, grp_list...)
	result := helpers.TableSelector("class group", grp_list, strconv.Itoa(classGrpFlag))
	if result.ExitRequest || !result.Selected {
		fmt.Println("Selection canceled")
		return
	}
	grp := result.Index

	datelist := getDateList(regNo, cookies, semester, grp_list[grp][1])
	if len(datelist) == 0 {
		fmt.Println("No months found")
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
		fmt.Println(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
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
		fmt.Println(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	return readmonths(doc)
}

func processDates(regNo string, cookies types.Cookies, semester types.Semester, grp string, datelist []string, flag int) ([][]int, int, int) {
	var months []string
	year := datelist[0][7:]
	var color_list [][]int
	if flag == 1 {
		fmt.Println("\033[31mRed-Exam Day\033[0m\n\033[34mBlue-Holiday\033[0m\n\033[32mGreen-Instructional Day\033[0m\n\033[33mYellow-Today\033[0m")
	}
	isLeapYear := func(year int) bool {
		if year%4 == 0 {
			if year%100 == 0 {
				return year%400 == 0
			}
			return true
		}
		return false
	}
	// Convert year string to integer
	yearInt, err := strconv.Atoi(year)
	if err != nil {
		fmt.Println("Invalid year:", year)
		return nil, -1, -1
	}

	daysInMonth := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if isLeapYear(yearInt + 1) {
		daysInMonth[1] = 29 // February has 29 days in a leap year

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

	// Find the starting month from datelist
	startMonthStr := datelist[0][3:6]
	startMonth, ok := monthMap[startMonthStr]
	if !ok {
		fmt.Println("Invalid month:", startMonthStr)
		return nil, -1, -1
	}
	// Create the nested array with each sublist having the number of days of the month
	return_list := [][]int{}
	for i := 0; i < len(datelist); i++ {
		monthSize := daysInMonth[(startMonth+i)%12]
		monthelement := make([]int, monthSize)
		return_list = append(return_list, monthelement)
	}

	for i, date := range datelist {
		if year != date[7:] {
			addPadding(&color_list)
			if flag == 1 {
				renderMonths(months, year, color_list)
			}
			months = []string{}
			color_list = [][]int{}
			year = ""
		}
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
		if err != nil && debug.Debug {
			fmt.Println(err)
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
		if err != nil && debug.Debug {
			fmt.Println(err)
		}
		typeOfDay := extractTypeOfDay(doc, return_list[i])
		color_list = append(color_list, typeOfDay)
		months = append(months, date[3:6])
		year = date[7:]
	}
	addPadding(&color_list)
	if flag == 1 {
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
			if len(span.Text()) > 0 {
				if _, err := strconv.Atoi(strings.TrimSpace(span.Text())); err != nil {
				} else {
					count++
				}
				if strings.Contains(span.Text(), "Holiday") {
					t = -1
					k = 1
				} else if strings.Contains(span.Text(), "No Instructional Day") {
					t = -1
					k = 1
				} else if strings.Contains(span.Text(), "Exam") {
					t = -1
					k = 2
				} else if strings.Contains(span.Text(), "Instructional Day") {
					t = 0
					k = 3
				} else if strings.Contains(span.Text(), "Day Order") {
					k = 3
					if strings.Contains(span.Text(), "Monday") {
						t = 1
					} else if strings.Contains(span.Text(), "Tuesday") {
						t = 2
					} else if strings.Contains(span.Text(), "Wednesday") {
						t = 3
					} else if strings.Contains(span.Text(), "Thursday") {
						t = 4
					} else if strings.Contains(span.Text(), "Friday") {
						t = 5
					}
				} else {
					t = -1
					k = 1
				}
			} else {
				return
			}
		})
		typeOfDay = append(typeOfDay, k)
		if count != -1 {
			arr[count] = t
		}
	})
	return typeOfDay
}

func renderMonths(months []string, year string, nestedColour [][]int) {
	calendars := make([][]string, len(months))
	for i, month := range months {
		calendars[i] = generateCalendarLines(month, nestedColour[i])
	}
	spaceSize := (len(months) * 24 / 2) - 2
	yearHeader := fmt.Sprintf(" %s%s", strings.Repeat(" ", spaceSize), year)
	maxLength := 20
	yearHeader = fmt.Sprintf("%-*s", (maxLength*len(months)+len(yearHeader))/2, yearHeader)
	fmt.Println(yearHeader)
	fmt.Println()
	for row := 0; row < len(calendars[0]); row++ {
		for i := 0; i < len(months); i++ {
			fmt.Print(calendars[i][row])
			if i < len(months)-1 {
				fmt.Print("    ")
			}
		}
		fmt.Println()
	}
}

func generateCalendarLines(month string, colour []int) []string {
	var lines []string
	now := time.Now()
	todayMonth := strings.ToUpper(now.Month().String()[:3])
	todayDay := now.Day()
	monthHeader := month
	maxLength := 20
	padding := (maxLength - len(monthHeader)/2) / 2
	monthHeader = fmt.Sprintf("%s%s%s", strings.Repeat(" ", padding), monthHeader, strings.Repeat(" ", padding))
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
	for i := range lines {
		if len(lines[i]) < maxLength {
			lines[i] = fmt.Sprintf("%-*s", maxLength, lines[i])
		}
	}
	if len(lines) == 8 {
		lines = append(lines, strings.Repeat(" ", 21))
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
