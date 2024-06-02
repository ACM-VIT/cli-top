package features

import (
	"bytes"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

//payload := []byte("_csrf=154a792d-e0d1-42fb-8300-c4211db46510&semesterSubId=VL20232405&authorizedID=22BCI0272&x=" + time.Now().UTC().Format(time.RFC1123))

func GetExamSchedule(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doSearchExamScheduleForStudent"

	semesterID := helpers.SelectSemester(regNo, cookies, sem_choice)

	bodyText, err := helpers.FetchReq(regNo, cookies, url, semesterID, "UTC", "POST")
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(string(bodyText))
	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}

	// Find and save the exam schedule
	examSchedule, err := findAndSaveExamSchedule(doc)
	if err != nil {
		log.Fatal(err)
	}

	// Generate Markdown table text
	markdownTable := generateExamScheduleMarkdownTable(examSchedule)
	//findAndSaveAtten(doc)
	// Render Markdown using glamour
	rendered, err := glamour.Render(markdownTable, "dark")
	if err != nil {
		fmt.Println("Error rendering Markdown:", err)
		return
	}

	// // Print the rendered Markdown
	fmt.Println(rendered)
}

func findAndSaveExamSchedule(doc *goquery.Document) ([][]string, error) {
	var examSchedule [][]string

	targetID := "customTable"
	//fmt.Println(targetID)
	table := doc.Find("table")

	if table.Length() > 0 {
		table.Find("tr").Each(func(i int, rowSelection *goquery.Selection) {
			if i == 0 {
				// Skip the first row (header row)
				return
			}

			var row []string
			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				row = append(row, text)
			})
			examSchedule = append(examSchedule, row)
		})
	} else {
		return nil, fmt.Errorf("table with ID '%s' not found", targetID)
	}

	return examSchedule, nil
}

func generateExamScheduleMarkdownTable(examSchedule [][]string) string {
	var buf bytes.Buffer
	// Iterate through exam schedule and generate rows
	for i, row := range examSchedule {
		if i != 0 {
			if len(row) >= 13 {
				if row[0] == "1" {
					title := examSchedule[0]
					fmt.Printf("| %-5s | %-11s | %-45s | %-11s | %-11s | %-14s | %-19s | %-7s | %-7s | %-8s |\n", title[0], title[1][7:], title[2], title[5], title[6], title[8], title[9], title[10], title[11][:4], title[12])
					fmt.Printf("|%-7s|%-13s|%-47s|%-13s|%-13s|%-16s|%-21s|%-9s|%-9s|%-10s|\n", strings.Repeat("-", 7),
						strings.Repeat("-", 13), strings.Repeat("-", 47), strings.Repeat("-", 13), strings.Repeat("-", 13), strings.Repeat("-", 16), strings.Repeat("-", 21),
						strings.Repeat("-", 9), strings.Repeat("-", 9), strings.Repeat("-", 10))
				}
				// Table row
				fmt.Printf("| %-5s | %-11s | %-45s | %-11s | %-11s | %-14s | %-19s | %-7s | %-7s | %-8s |\n", row[0], row[1], row[2], row[5], row[6], row[8], row[9], row[10], row[11], row[12])
			} else {
				fmt.Println()
				fmt.Println(row)
				fmt.Println()
			}
		}
	}

	return buf.String()
}
