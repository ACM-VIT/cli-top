package features

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"vtop-cli/helpers"
	"vtop-cli/types"

	//"math"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	//"golang.org/x/net/html"
)

func fetchReq4(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request

	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())
	//fmt.Println(payload)
	// Create a new request with POST method and payload
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, err
	}

	// Set headers or cookies if needed
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

func fetchReq2(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request

	payload := fmt.Sprintf("authorizedID=%s&_csrf=%s&semesterSubId=%s&x=%s", regNo, cookies.CSRF, semID, time.Now().UTC().Format(time.RFC1123)) //fmt.Println(payload)
	//fmt.Println(payload)
	// Create a new request with POST method and payload
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, err
	}

	// Set headers or cookies if needed
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

// func pageLogic(c int,tot int, semDetails SemesterDetails){
// 	var buf bytes.Buffer

// 	fmt.Printf("\n--- PAGE %d/%d ---\n\n", c, tot)

// 	if 10*(c-1)+9 < len(semDetails.SemIds){

// 		buf.WriteString("| Index | SemId          | SemName                             |\n")
// 		buf.WriteString("|-------|----------------|-------------------------------------|\n")
// 		for i := 10*(c-1); i < 10*(c-1)+10; i++ {
// 			index := fmt.Sprintf("%d", i+1)
// 			semId := semDetails.SemIds[i]
// 			semName := semDetails.SemNames[i]

// 			// Table row
// 			buf.WriteString(fmt.Sprintf("| %-5s | %-14s | %-35s |\n", index, semId, semName))
// 		}

// 	}else{
// 		buf.WriteString("| Index | SemId          | SemName                             |\n")
// 		buf.WriteString("|-------|----------------|-------------------------------------|\n")
// 		for i := 10*(c-1); i < len(semDetails.SemIds); i++ {
// 			 index := fmt.Sprintf("%d", i+1)
// 			 semId := semDetails.SemIds[i]
// 			 semName := semDetails.SemNames[i]

// 			 // Table row

// 			 buf.WriteString(fmt.Sprintf("| %-5s | %-14s | %-35s |\n", index, semId, semName))
// 		}

// 	}
// 	fmt.Println(buf.String())
// 	fmt.Printf("--- END OF PAGE %d/%d ---",c,tot)
// 	fmt.Println(" ")
// }

func GetGrade(regNo string, cookies types.Cookies, semId string) {

	url := "https://vtop.vit.ac.in/vtop/examinations/examGradeView/doStudentGradeView"

	sel_id := GradeView(regNo, cookies, 0)
	//fmt.Println(sel_id)
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

func GradeView(regNo string, cookies types.Cookies, sem_choice int) string {

	selectedSemId := ""
	selectedSemName := ""

	var choice int
	fmt.Print("\nEnter the index of the semester to view grade: ")
	fmt.Scanln(&choice)
	semDet := helpers.GetSemDetails(cookies, regNo)

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
		helpers.PrintTable("Header", nil, &markdownTable)

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
				helpers.PrintFormattedRow(row, &markdownTable, count)
			}
		})
	} else {
		fmt.Println("Data not found")
	}

	fmt.Println(markdownTable.String())
	doc.Find("span[style='font-size: 18px; font-weight: bold;']").Each(func(i int, s *goquery.Selection) {
		gpa := s.Text()
		fmt.Println("\x1b[32;1mGreen\x1b[0m : Course not included in GPA/CGPA\n")
		fmt.Println(gpa)
	})
}
