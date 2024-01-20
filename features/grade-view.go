package features

import (
	types "cli-top/types"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

func GetGrade(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/examinations/examGradeView/doStudentGradeView"

	sel_id := Grade(regNo, cookies, sem_choice)
	bodyText, err := fetchReq2(regNo, cookies, url, sel_id)
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}

	findAndSaveGrade(doc)

}

func Grade(regNo string, cookies types.Cookies, sem_choice int) string {
	selectedSemId := ""
	selectedSemName := ""

	var choice int
	if sem_choice == 0 {
		PrintSemDetails(regNo, cookies)
		fmt.Print("\nEnter the index of the semester to view grade: ")
		fmt.Scanln(&choice)
	} else {
		choice = sem_choice
	}
	semDet := GetSemDetailsAtten(cookies, regNo)

	if choice < 1 || choice > len(semDet.SemIds) {
		fmt.Println("Invalid choice.")
	} else {
		for i, id := range semDet.SemIds {
			if i+1 == choice {
				selectedSemId = id
				selectedSemName = semDet.SemNames[i]
			}
		}
	}

	// Format the string with glamour
	formattedSelection := fmt.Sprintf("\n# You selected SemId: %s, SemName: %s\n", selectedSemId, selectedSemName)

	// Render and print the formatted string
	renderer, err := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(150))
	if err != nil {
		log.Fatal("Error creating glamour renderer:", err)
	}

	output, err := renderer.Render(formattedSelection)
	if err != nil {
		log.Fatal("Error rendering formatted string:", err)
	}

	fmt.Print(output)

	fmt.Println()

	return selectedSemId

}

func findAndSaveGrade(doc *goquery.Document) {
	var markdownTable strings.Builder
	var count int
	targetClass := "table.table-hover.table-bordered"
	table := doc.Find(targetClass)

	if table.Length() > 0 {
		printTableGrade("Header", nil, &markdownTable)

		// Find rows within tbody
		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
			style, exists := rowSelection.Attr("style")
			if exists && style == "background-color: #C0D8C0;" {
				count = i

			}
			//fmt.Println(count)
			row := []string{} // Initialize a new slice for each row

			// Find cells (td) within each row
			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) { //tr=11, td=11 , th=9+4
				text := strings.TrimSpace(cell.Text())
				row = append(row, text)
			})

			if len(row) > 2 {
				printFormattedRowGrade(row, &markdownTable, count)
			}
		})
	} else {
		fmt.Println("Data not found")
	}

	fmt.Println(markdownTable.String())
	doc.Find("span[style='font-size: 18px; font-weight: bold;']").Each(func(i int, s *goquery.Selection) {
		gpa := s.Text()
		fmt.Println("\x1b[32;1mCourse not included in GPA/CGPA\x1b[0m\n")
		fmt.Println(gpa)
	})
}

func printTableGrade(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-10s | %-50s | %-25s | %-21s | %-5s | %-6s |\n",
		"S.No.", "Course Code", "Course Title", "Course Type", "Credits", " Total", "Grade"))
	builder.WriteString("|       |             |                                                    |                           |-----------------------|        |        |\n")
	//builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")

	builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-7s| %-6s |\n",
		"", "", "", "", "L", "P", "J", "C", "", ""))
	builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")
}

func printFormattedRowGrade(row []string, builder *strings.Builder, count int) {
	if strToInt(row[0]) == count-1 {
		builder.WriteString(fmt.Sprintf("| \x1b[32m%-5s\x1b[0m | \x1b[32m%-11s\x1b[0m | \x1b[32m%-50s\x1b[0m | \x1b[32m%-25s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-6s\x1b[0m | \x1b[32m%-6s\x1b[0m |\n",
			row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[9], row[10]))
	} else {
		builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-6s | %-6s |\n",
			row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[9], row[10]))
	}
}
