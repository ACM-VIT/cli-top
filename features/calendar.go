package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"time"

	"strings"

	"github.com/PuerkitoBio/goquery"
)

func PrintCal(regNo string, cookies types.Cookies, sem_choice int, classGrpFlag int) {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		fmt.Println("Please login using the cli-top login command.")
		return
	}
	url := "https://vtop.vit.ac.in/vtop/getDateForSemesterPreview"
	semester, err := helpers.SelectSemester(regNo, cookies, sem_choice)
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}
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
	grp_list := extractclassgrp(doc)
	if len(grp_list) == 0 {
		fmt.Println("No calendars available")
		return
	}
	grp_list = append([][]string{{"CLASS GROUP"}}, grp_list...)
	grp := helpers.TableSelector("class group", grp_list, classGrpFlag)
	url = "https://vtop.vit.ac.in/vtop/getListForSemester"
	payloadMap = map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getListForSemester",
		"semSubId":      semester.SemID,
		"classGroupId":  grp_list[grp][1],
		"authorizedID":  regNo,
		"x":             fmt.Sprintf("%d", time.Now().Unix()),
	}
	formData = helpers.FormatBodyData(payloadMap)
	bodyText, err = helpers.FetchReq(regNo, cookies, url, semester.SemID, formData, "POST", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	doc, err = goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	datelist := readmonths(doc)
	if len(datelist) == 0 {
		fmt.Println("No months found")
		return
	}
	var months []string
	year := datelist[0][7:]
	var color_list [][]int

	for _, date := range datelist {
		if year != date[7:] {
			for i := range color_list {
				padding := (7 - (len(color_list[i]) % 7)) % 7
				for j := 0; j < padding; j++ {
					color_list[i] = append(color_list[i], 0)
				}
			}
			renderMonths(months, year, color_list)
			months = []string{}
			color_list = [][]int{}
			year = ""
		}
		url = "https://vtop.vit.ac.in/vtop/processViewCalendar"
		payloadMap = map[string]string{
			"_csrf":        cookies.CSRF,
			"calDate":      date,
			"semSubId":     semester.SemID,
			"classGroupId": grp_list[grp][1],
			"authorizedID": regNo,
			"x":            fmt.Sprintf("%d", time.Now().Unix()),
		}
		formData = helpers.FormatBodyData(payloadMap)

		bodyText, err = helpers.FetchReq(regNo, cookies, url, semester.SemID, formData, "POST", "")
		if err != nil && debug.Debug {
			fmt.Println(err)
		}
		doc, err = goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
		if err != nil && debug.Debug {
			fmt.Println(err)
		}
		var typeOfDay []int // Reset typeOfDay for each month
		doc.Find("td").Each(func(i int, s *goquery.Selection) {
			k := 0
			s.Find("span").Each(func(j int, span *goquery.Selection) {
				if len(span.Text()) > 0 {
					if strings.Contains(span.Text(), "Holiday") {
						k = 1
					} else if strings.Contains(span.Text(), "No Instructional Day") {
						k = 1
					} else if strings.Contains(span.Text(), "Exam") {
						k = 2
					} else if strings.Contains(span.Text(), "Instructional Day") {
						k = 3
                    } else if strings.Contains(span.Text(), "Day Order") {
                        k = 3
					} else {
						k = 1
					}
				} else {
					return
				}
			})
			typeOfDay = append(typeOfDay, k)
		})
		color_list = append(color_list, typeOfDay)
		months = append(months, date[3:6])
		year = date[7:]
	}
	for i := range color_list {
		padding := (7 - (len(color_list[i]) % 7)) % 7
		for j := 0; j < padding; j++ {
			color_list[i] = append(color_list[i], 0)
		}
	}
	renderMonths(months, year, color_list)
}

func renderMonths(months []string, year string, nestedColour [][]int) {
	// Store calendar lines for each month
	calendars := make([][]string, len(months))

	// Generate calendar strings for each month
	for i, month := range months {
		calendars[i] = generateCalendarLines(month, nestedColour[i])
	}

	// Print the year centered
	spaceSize := (len(months) * 24 / 2) - 2
	// Print the year centered
	yearHeader := fmt.Sprintf(" %s%s", strings.Repeat(" ", spaceSize), year)
	maxLength := 20 // Increased to accommodate the full calendar width
	fmt.Println("\033[31mRed-Exam Day\033[0m\n\033[34mBlue-Holiday\033[0m\n\033[32mGreen-Instructional Day\033[0m")
	//fmt.Println("\033[41mRed-Exam Day\033[0m\n\033[44mBlue-Holiday\033[0m\n\033[42mGreen-Instructional Day\033[0m")
	yearHeader = fmt.Sprintf("%-*s", (maxLength*len(months)+len(yearHeader))/2, yearHeader)
	fmt.Println(yearHeader)
	fmt.Println()

	// Print calendars side by side
	for row := 0; row < len(calendars[0]); row++ {
		for i := 0; i < len(months); i++ {
			fmt.Print(calendars[i][row])
			if i < len(months)-1 {
				fmt.Print("    ") // Increased spacing between calendars
			}
		}
		fmt.Println()
	}
}

func generateCalendarLines(month string, colour []int) []string {
	var lines []string
	// Header
	monthHeader := fmt.Sprintf("%s", month)

	// Center-align the month header
	maxLength := 20
	padding := (maxLength - len(monthHeader)/2) / 2
	monthHeader = fmt.Sprintf("%s%s%s", strings.Repeat(" ", padding), monthHeader, strings.Repeat(" ", padding))

	lines = append(lines, monthHeader)
	lines = append(lines, "Su Mo Tu We Th Fr Sa ")

	// Build the days
	var currentLine strings.Builder
	dayOfMonth := 1
	for i, day := range colour {
		if i%7 == 0 && i != 0 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}

		if day == 0 {
			currentLine.WriteString("   ") // Three spaces for blank days
		} else {
			color := helpers.Reset
			switch day {
			case 1:
				color = helpers.Blue
			case 2:
				color = helpers.Red
			case 3:
				color = helpers.Green
			}

			// Use consistent spacing for all numbers
			currentLine.WriteString(fmt.Sprintf("%s%2d%s ", color, dayOfMonth, helpers.Reset))
			dayOfMonth++
		}
	}

	// Add the last line if it's not empty
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	// Pad each line to a consistent width
	for i := range lines {
		if len(lines[i]) < maxLength {
			lines[i] = fmt.Sprintf("%-*s", maxLength, lines[i])
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
