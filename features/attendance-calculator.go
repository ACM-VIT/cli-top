package features

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
	"vtop-cli/helpers"
	"vtop-cli/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

func fetchReq3(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
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

func fetchReq5(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
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

//payload := []byte("_csrf=154a792d-e0d1-42fb-8300-c4211db46510&semesterSubId=VL20232405&authorizedID=22BCI0272&x=" + time.Now().UTC().Format(time.RFC1123))

func GetAttendance(regNo string, cookies types.Cookies, semId string) {

	url := "https://vtop.vit.ac.in/vtop/processViewStudentAttendance"

	sel_id := AttendanceCalculator(regNo, cookies, 0)
	//fmt.Println(sel_id)
	bodyText, err := fetchReq2(regNo, cookies, url, sel_id)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println("bodyText" , bodyText)
	//bodyString := string(bodyText)
	//fmt.Println(bodyString)

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(doc)

	//var temp[] string
	//var Atten[] string
	findAndSaveAtten(doc)
	//fmt.Printf("%q", tempIds)
	// 	Atten = removeEmptyStrings(temp)
	// 	fmt.Printf("%q", Atten)

	// 	subjectDetails := subjectDetails(doc)
	// 	fmt.Println("hi")
	// 	fmt.Println(subjectDetails)

}

func findAndSaveAtten(doc *goquery.Document) {
	var markdownTable strings.Builder
	targetID := "AttendanceDetailDataTable"
	table := doc.Find("table#" + targetID)
	if table.Length() > 0 {
		helpers.PrintTable("Header", nil, &markdownTable)
		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
			row := []string{} // Initialize a new slice for each row
			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				row = append(row, text)
			})
			//fmt.Println("hi table")
			//fmt.Println(row)
			helpers.PrintFormattedRow(row, &markdownTable, 1) //broken
		})
	} else {
		fmt.Println("Table with ID 'AttendanceDetailDataTable' not found")
	}
	fmt.Println(markdownTable.String())
}

// func convertHTMLElementToMarkdown(element *goquery.Selection) (string, error) {
// 	var markdownTable strings.Builder
// 	var tableStarted bool

// 	// Use goquery for easier HTML manipulation
// 	// doc := goquery.NewDocumentFromNode(element)

// 	// Find and print data rows excluding rows with class "tableHeader-level1"
// 	element.Find("tbody tr").Each(func(_ int, rowSelection *goquery.Selection) {
// 		// Check if the row has the specified class
// 		if !rowSelection.HasClass("tableHeader-level1") {
// 			if !tableStarted {
// 				printTable("Header", nil, &markdownTable) // Print header only once
// 				tableStarted = true
// 			}

// 			row := []string{}
// 			rowSelection.Find("td").Each(func(_ int, cellSelection *goquery.Selection) {
// 				// Extract and append cell text to the markdownTable string
// 				text := strings.TrimSpace(cellSelection.Text())
// 				row = append(row, text)
// 			})
// 			helpers.PrintFormattedRow(row, &markdownTable)
// 		}
// 	})

// 	return markdownTable.String(), nil
// }

func Cal75(att int, tot int, perc int) string {
	var ret string
	if perc == 75 {
		ret = "\033[31m" + "Can skip 0 class" + "\033[0m" + "\t"
	} else if perc < 75 {
		for i := 1; i < att; i++ {
			if math.Ceil((float64(att+i)/float64(tot+i))*100) <= 75 {
				ret = fmt.Sprintf("\033[31m"+"Attend %d class\033[0m"+"\033[0m"+"\t", i)

			}
		}
	} else {
		//fmt.Println("hello")
		//  fmt.Println(att)
		//  fmt.Println(tot)
		//  fmt.Println(perc)
		//fmt.Println((float64(att) / float64(tot)) * 100)
		//for j := 0; j<15 ; j++{
		for i := 1; i < att; i++ {
			//fmt.Println(att / (tot+i))
			//fmt.Println("hello1")
			//fmt.Println(math.Ceil((float64(att) / float64(tot+i)) * 100) )
			if math.Ceil((float64(att)/float64(tot+i))*100) >= 75 {

				//fmt.Println("hello2")
				ret = fmt.Sprintf("\033[32m"+"Can skip %d class"+"\033[0m"+"\t", i)

			}
		}
		//}

	}
	return ret
}

func AttendanceCalculator(regNo string, cookies types.Cookies, sem_choice int) string {

	selectedSemId := ""
	selectedSemName := ""

	var choice int
	fmt.Print("\nEnter the index of the semester to view attendance: ")
	fmt.Scanln(&choice)
	semDet := helpers.GetSemDetails(cookies, regNo)

	if choice < 1 || choice > len(semDet.SemIds) {
		fmt.Println("Invalid choice.")
	} else {
		for i, id := range semDet.SemIds {
			if i+1 == choice {
				// fmt.Println("Selected sem id : ",id)
				// fmt.Println("Selected sem name : ",semDet.SemNames[i])
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

	// AttendanceCalculator(regNo,cookies,sem_choice)
	// GetAttendance(regNo, cookies, selectedSemId)
	// fmt.Println("hii")

	return selectedSemId

}
