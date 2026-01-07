package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	GradeTableSelector     = "table.table-hover.table-bordered"
	GradeHeaderRowSelector = "thead tr"
	GradeRowsSelector      = "tbody tr"
	GradeCellSelector      = "td"
	GradeHeaderSelector    = "th"
	GradeSummarySelector   = "div.panel-body"
)

func GetGrades(regNo string, cookies types.Cookies, semId string, semChoice int) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	url := "https://vtop.vit.ac.in/vtop/examinations/examGradeView/doStudentGradeView"

	semester, err := helpers.SelectSemester(regNo, cookies, semChoice)
	if err != nil {
		helpers.HandleError("fetching semesters", err)
		helpers.Println()
		return
	}

	bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, "UTC", "POST", "")
	if err != nil {
		helpers.HandleError("fetching grades", err)
		if debug.Debug {
			helpers.Println(err)
		}
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		helpers.HandleError("parsing HTML", err)
		if debug.Debug {
			helpers.Println(err)
		}
	}

	findAndSaveGrade(doc)
}

func findAndSaveGrade(doc *goquery.Document) {
	var gradesData [][]string
	table := doc.Find(GradeTableSelector)

	if table.Length() == 0 {
		helpers.Println("Data not found")
		return
	}

	headers := []string{
		"Course Code", "Course Title", "Course Type",
		"Credits", "Total", "Grading", "Grade",
	}
	gradesData = append(gradesData, headers)

	table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		row := helpers.ExtractRowData(rowSelection)

		if len(row) < 12 {
			for len(row) < 12 {
				row = append(row, "")
			}
		}

		if containsGPA(row) {
			return
		}

		selectedRow := []string{
			row[1],  // Course Code
			row[2],  // Course Title
			row[3],  // Course Type
			row[7],  // Credits
			row[9],  // Total
			row[8],  // Grading
			row[10], // Grade
		}

		grading := strings.ToUpper(strings.TrimSpace(selectedRow[5]))
		switch grading {
		case "AG":
			selectedRow[5] = "Absolute"
		case "RG":
			selectedRow[5] = "Relative"
		}

		courseType := strings.ToUpper(strings.TrimSpace(selectedRow[2]))
		grade := strings.ToUpper(strings.TrimSpace(selectedRow[6]))

		if courseType == "ONLINE COURSE" || courseType == "PROJECT" || courseType == "EXTRA CURRICULAR ACTIVITY" {
			for idx := range selectedRow {
				if selectedRow[0][0:4] != "CFOC" {
					selectedRow[idx] = fmt.Sprintf("\x1b[32m%s\x1b[0m", selectedRow[idx]) // Green
				}
			}
		} else {
			if grade == "F" || grade == "N" {
				for idx := range selectedRow {
					selectedRow[idx] = fmt.Sprintf("\x1b[31m%s\x1b[0m", selectedRow[idx]) // Red
				}
			}
		}

		gradesData = append(gradesData, selectedRow)
	})

	gradesData = filterEmptyRows(gradesData, len(headers))

	helpers.PrintTable(gradesData, 1)
	helpers.Println()

	doc.Find("span[style='font-size: 18px; font-weight: bold;']").Each(func(i int, s *goquery.Selection) {
		gpa := s.Text()
		helpers.Println("\x1b[32;1m**Course not included in GPA/CGPA\x1b[0m")
		helpers.Println(gpa)
	})
}

func containsGPA(row []string) bool {
	for _, cell := range row {
		if strings.Contains(strings.ToUpper(cell), "GPA") {
			return true
		}
	}
	return false
}

func filterEmptyRows(data [][]string, expectedLength int) [][]string {
	filtered := [][]string{}
	for _, row := range data {
		if len(row) != expectedLength {
			continue
		}
		empty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				empty = false
				break
			}
		}
		if !empty {
			filtered = append(filtered, row)
		}
	}
	return filtered
}
