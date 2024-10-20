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

const (
	green  = "\033[32m"
	red    = "\033[31m"
	reset  = "\033[0m"
	yellow = "\033[33m"
	blue   = "\033[34m"
)

func PrintCal(regNo string, cookies types.Cookies, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/getDateForSemesterPreview"
	semesterID := helpers.SelectSemester(regNo, cookies, sem_choice)
	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getDateForSemesterPreview",
		"semSubId":      semesterID,
		"authorizedID":  regNo,
		"x":             fmt.Sprintf("%d", time.Now().Unix()),
	}
	formData := helpers.FormatBodyData(payloadMap)
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semesterID, formData, "POST", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	grp_list := extractclassgrp(doc)

	PrintMarkdownList(grp_list)
	var choice int
	fmt.Print("Choose a class group: ")
	_, err = fmt.Scan(&choice)
	if err != nil {
		fmt.Println("Invalid input. Please enter a valid number.")
		return
	}
	if choice < 1 || choice > len(grp_list) {
		fmt.Println("Invalid choice.")
		return
	}
	selected_grp_code := grp_list[choice-1][0]
	fmt.Printf("\n    \033[1;44mYou selected Group: %s\033[0m\n\n\n", grp_list[choice-1][1])
	url = "https://vtop.vit.ac.in/vtop/getListForSemester"
	payloadMap = map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getListForSemester",
		"semSubId":      semesterID,
		"classGroupId":  selected_grp_code,
		"authorizedID":  regNo,
		"x":             fmt.Sprintf("%d", time.Now().Unix()),
	}
	formData = helpers.FormatBodyData(payloadMap)
	bodyText, err = helpers.FetchReq(regNo, cookies, url, semesterID, formData, "POST", "")
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
			"semSubId":     semesterID,
			"classGroupId": selected_grp_code,
			"authorizedID": regNo,
			"x":            fmt.Sprintf("%d", time.Now().Unix()),
		}
		formData = helpers.FormatBodyData(payloadMap)

		bodyText, err = helpers.FetchReq(regNo, cookies, url, semesterID, formData, "POST", "")
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
			color := reset
			switch day {
			case 1:
				color = blue
			case 2:
				color = red
			case 3:
				color = green
			}

			// Use consistent spacing for all numbers
			currentLine.WriteString(fmt.Sprintf("%s%2d%s ", color, dayOfMonth, reset))
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
			text := option.Text()
			grp_list = append(grp_list, []string{value, text})
		})
	})
	grp_list = grp_list[1:]
	return grp_list
}

func PrintMarkdownList(nestedList [][]string) {
	// Determine the maximum width for each column
	maxWidths := []int{len("INDEX"), len("CLASS GROUP")}
	for i, row := range nestedList {
		serial := fmt.Sprintf("%d", i+1)
		if len(serial) > maxWidths[0] {
			maxWidths[0] = len(serial)
		}
		if len(row) > 1 && len(row[1]) > maxWidths[1] {
			maxWidths[1] = len(row[1])
		}
	}

	// Function to center-align text
	center := func(s string, width int) string {
		if len(s) >= width {
			return s
		}
		pad := width - len(s)
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	}

	// Create format strings based on the maximum widths
	formatHeader := fmt.Sprintf("    %%-%ds │ %%-%ds\n", maxWidths[0], maxWidths[1])
	formatData := fmt.Sprintf("    %%%ds │ %%-%ds\n", maxWidths[0], maxWidths[1]) // Right-align the index
	separator := fmt.Sprintf("    %s─┼─%s\n",
		strings.Repeat("─", maxWidths[0]),
		strings.Repeat("─", maxWidths[1]))

	var builder strings.Builder
	// Print the table header with centered headers
	builder.WriteString(fmt.Sprintf(formatHeader,
		center("INDEX", maxWidths[0]),
		center("CLASS GROUP", maxWidths[1])))

	// Print separator line
	builder.WriteString(separator)

	// Print the table rows
	for i, row := range nestedList {
		serial := fmt.Sprintf("%d", i+1)
		semname := ""
		if len(row) > 1 {
			semname = strings.TrimSpace(row[1]) // Strip starting whitespace
		}
		builder.WriteString(fmt.Sprintf(formatData, serial, semname)) // Right-aligned index
	}

	fmt.Println(builder.String())
}
