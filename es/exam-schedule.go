package es

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vtop-cli/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	"golang.org/x/net/html"
)

func fetchReq(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
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

//payload := []byte("_csrf=154a792d-e0d1-42fb-8300-c4211db46510&semesterSubId=VL20232405&authorizedID=22BCI0272&x=" + time.Now().UTC().Format(time.RFC1123))

func GetSemDetails(cookies types.Cookies, regNo string) types.SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"

	//fmt.Println(regNo, cookies)
	bodyText, err := fetchReq(regNo, cookies, url, "")
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(string(bodyText))

	// Create a slice to store the extracted data
	var SemNames []string
	var SemIds []string

	var tempIds []string
	// Find and save the semester IDs
	FindAndSaveSemIds(doc, "form-select", &tempIds)
	//fmt.Printf("%q", tempIds)
	SemIds = RemoveEmptyStrings(tempIds)
	//fmt.Printf("%q", SemIds)

	// fmt.Printf("%q\n", SemIds)

	for _, semId := range SemIds {
		optionText := FindOptionWithTagValue(doc, semId)

		if optionText != "" {
			SemNames = append(SemNames, optionText)

		} else {
			// Handle the case where no <option> tag is found with the specified SemId
			SemNames = append(SemNames, "Unknown")
		}
	}

	// Reverse SemIds and SemNames
	reverseSemIds := make([]string, len(SemIds))
	reverseSemNames := make([]string, len(SemNames))
	for i := 0; i < len(SemIds); i++ {
		reverseIndex := len(SemIds) - 1 - i
		reverseSemIds[i] = SemIds[reverseIndex]
		reverseSemNames[i] = SemNames[reverseIndex]
	}

	// Save the reversed values back to SemDetails
	SemIds = reverseSemIds
	SemNames = reverseSemNames

	//fmt.Println("\nget wroking")
	// Return the encapsulated struct
	return types.SemesterDetails{
		SemNames: SemNames,
		SemIds:   SemIds,
	}
}

func PrintSemDetails(regNo string, cookies types.Cookies) {
	semDetails := GetSemDetails(cookies, regNo)
	//fmt.Println(len(semDetails.SemNames))
	//fmt.Println(len(semDetails.SemIds))
	if len(semDetails.SemIds) == 0 {
		fmt.Println("Error fetching semester details or no semesters available.")
		return
	}

	// Generate Markdown table text
	markdownTable := GenerateSemDetailsMarkdownTable(semDetails)

	// Render Markdown using glamour
	rendered, err := glamour.Render(markdownTable, "dark")
	if err != nil {
		fmt.Println("Error rendering Markdown:", err)
		return
	}

	// Print the rendered Markdown
	fmt.Println(rendered)
}

func GenerateSemDetailsMarkdownTable(semDetails types.SemesterDetails) string {
	var buf bytes.Buffer

	// Table header
	buf.WriteString("| Index | SemId          | SemName                   |\n")
	buf.WriteString("|-------|----------------|---------------------------|\n")

	// Iterate through SemIds and SemNames using a for loop
	for i := 0; i < len(semDetails.SemIds); i++ {
		index := fmt.Sprintf("%d", i+1)
		semId := semDetails.SemIds[i]
		semName := semDetails.SemNames[i]

		// Table row
		buf.WriteString(fmt.Sprintf("| %-5s | %-14s | %-25s |\n", index, semId, semName))
	}

	return buf.String()
}

func FindAndSaveSemIds(doc *goquery.Document, targetClass string, result *[]string) {
	// Find all <option> elements within <select> tags with the specified class
	//fmt.Println(targetClass, doc)
	//selection := doc.Find("select.form-select option")
	//fmt.Println("Number of elements found:", selection.Length())

	doc.Find("select." + targetClass + " option").Each(func(i int, s *goquery.Selection) {

		value, exists := s.Attr("value")
		//fmt.Println(value)
		if exists {
			*result = append(*result, value)
		}
	})
}

func RemoveEmptyStrings(data []string) []string {
	var cleanedData []string
	for _, item := range data {
		if item != "" {
			cleanedData = append(cleanedData, item)
		}
	}
	return cleanedData
}

func FindOptionWithTagValue(doc *goquery.Document, targetValue string) string {
	return doc.Find("option[value='" + targetValue + "']").Text()
}

func getTextContent(n *html.Node) string {
	var textContent string

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			textContent += c.Data
		} else if c.Type == html.ElementNode {
			textContent += getTextContent(c)
		}
	}

	return textContent
}

func GetExamSchedule(regNo string, cookies types.Cookies, semId string) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doSearchExamScheduleForStudent"
	selID := Marks(regNo, cookies, 0)

	bodyText, err := fetchReq2(regNo, cookies, url, selID)
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
	examSchedule, err := findAndSaveAtten(doc)
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

func findAndSaveAtten(doc *goquery.Document) ([][]string, error) {
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

			// Check if the length of the row is sufficient
			if len(row) >= 13 {
				// Access elements in the row slice

				//fmt.Printf("Row[%d]: %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s\n", i, row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8], row[9], row[10], row[11], row[12])
			} else {
				// Handle the case where the row length is insufficient
				//fmt.Printf("Insufficient data in row[%d]: %v\n", i, row)
			}

			// Render the row based on conditions
			// if i == 1 {
			// 	// This is the subheading row
			// 	fmt.Println("\nSubheading Row:")
			// 	for _, col := range row {
			// 		fmt.Printf("%-15s", col)
			// 	}
			// 	fmt.Println()
			// } else {
			// 	// This is an actual class and timing row
			// 	fmt.Println("\nClass and Timing Row:")
			// 	for _, col := range row {
			// 		fmt.Printf("%-15s", col)
			// 	}
			// 	fmt.Println()
			// }

			// Append the row to examSchedule
			examSchedule = append(examSchedule, row)
		})
	} else {
		return nil, fmt.Errorf("Table with ID '%s' not found", targetID)
	}

	return examSchedule, nil
}

func generateExamScheduleMarkdownTable(examSchedule [][]string) string {
	var buf bytes.Buffer

	// Table header
	//buf.WriteString("| S.No | Course Code | Course Title | Course Type | Class ID | Slot | Exam Date | Exam Session | Reporting Time | Exam Time | Venue | Seat Location | Seat No |\n")
	//buf.WriteString("|------|--------------|---------------|-------------|----------|------|------------|---------------|-----------------|-----------|-------|----------------|---------|\n")

	// Iterate through exam schedule and generate rows
	for _, row := range examSchedule {
		if len(row) >= 13 {
			// Table row
			fmt.Printf("%-6s | %-15s | %-45s | %-14s | %-18s | %-11s | %-15s | %-18s | %-18s | %-19s | %-9s | %-20s | %-9s\n", row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8], row[9], row[10], row[11], row[12])
		}
	}

	return buf.String()
}

func subjectDetails(doc *goquery.Document) []string {
	var details []string

	// Use CSS selectors to find and extract data
	doc.Find("tr.tableContent").Each(func(i int, s *goquery.Selection) {
		// Skip every other iteration
		if i%2 != 0 {
			return
		}

		// Extract data from each column
		td := s.Find("td")
		sNo := td.Eq(0).Text()
		courseCode := td.Eq(1).Text()
		courseTitle := td.Eq(2).Text()
		courseType := td.Eq(3).Text()
		classID := td.Eq(4).Text()
		slot := td.Eq(5).Text()
		examDate := td.Eq(6).Text()
		examSession := td.Eq(7).Text()
		reportingTime := td.Eq(8).Text()
		examTime := td.Eq(9).Text()
		venue := td.Eq(10).Text()
		seatLocation := td.Eq(11).Text()
		seatNo := td.Eq(12).Text()

		// Print or use the extracted data
		detail := fmt.Sprintf("## S.No.: %s, Course Code: %s, Course Title: %s, Course Type: %s, Class ID: %s, Slot: %s, Exam Date: %s, Exam Session: %s, Reporting Time: %s, Exam Time: %s, Venue: %s, Seat Location: %s, Seat No: %s\n",
			sNo, courseCode, courseTitle, courseType, classID, slot, examDate, examSession, reportingTime, examTime, venue, seatLocation, seatNo)

		details = append(details, detail)
	})
	return details
}

func printTable(title string, data [][]string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-5s | %-8s | %-18s | %-5s | %-10s | %-10s | %-10s |%-5s |%-14s |%-14s |%-5s |%-5s |%-5s |\n",
		"S.No", "Course Code", "Course Title", "Course Type", "	Class ID", "Slot", "Exam Date", "Exam Session", "Reporting Time", "Exam Time", "Venue", "Seat Location", "Seat No"))
	builder.WriteString("|-------|----------------|---------------|-------------|------|-----------|--------------|-------------------|-----------|---------|---------------|----------|\n")
}

func strToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		fmt.Println("Error converting string to integer:", err)

	}

	return num
}

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

func findElementsByClass(doc *goquery.Document, class string) []*goquery.Selection {
	var result []*goquery.Selection

	doc.Find("." + class).Each(func(_ int, selection *goquery.Selection) {
		result = append(result, selection)
	})

	return result
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			classes := strings.Fields(attr.Val)
			for _, c := range classes {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func Marks(regNo string, cookies types.Cookies, sem_choice int) string {

	selectedSemId := ""
	selectedSemName := ""

	PrintSemDetails(regNo, cookies)

	var choice int
	fmt.Print("\nEnter the index of the semester to view examschedule: ")
	fmt.Scanln(&choice)
	semDet := GetSemDetails(cookies, regNo)

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
	//glamour.WithWordWrap(150)

	fmt.Print(output)
	//fmt.Print(helpers.ExtractBodyText(*http.Response))
	fmt.Println()

	// Marks(regNo,cookies,sem_choice)
	// GetExamSchedule(regNo, cookies, selectedSemId)
	// fmt.Println("hii")

	return selectedSemId

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
// 			printFormattedRow(row, &markdownTable)
// 		}
// 	})

// 	return markdownTable.String(), nil
// }

/*
		// Convert HTML elements to Markdown tables
		for i, element := range elements {
			// Convert each element to Markdown
			markdownTable, err := convertHTMLElementToMarkdown(element)
			if err != nil {
				log.Fatal(err)
			}

			renderer, e := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(150))
			if e != nil {
				fmt.Println("Error rendering markdown:", err)
				return
			}
			markdown, err := renderer.Render(markdownTable)
			if err != nil {
				fmt.Println("Error rendering Table:", err)
				return
			}

			subjectDetail, e1 := renderer.Render(subjectDetails[i])
			if e1 != nil {
				fmt.Println("Error rendering SubjectDetails:", err)
				return
			}

			fmt.Println(subjectDetail)
			fmt.Println(markdown)
		}
	}
*/
/*
func convertHTMLElementToMarkdown(element *goquery.Selection) (string, error) {
	var markdownTable strings.Builder
	var tableStarted bool

	// Find and print data rows excluding rows with class "tableHeader-level1"
	element.Find("tbody tr").Each(func(_ int, rowSelection *goquery.Selection) {
		// Check if the row has the specified class
		if !rowSelection.HasClass("tableHeader-level1") {
			if !tableStarted {
				printTable("Header", nil, &markdownTable) // Print header only once
				tableStarted = true
			}

			row := []string{}
			rowSelection.Find("td").Each(func(_ int, cellSelection *goquery.Selection) {
				// Extract and append cell text to the markdownTable string
				text := strings.TrimSpace(cellSelection.Text())
				row = append(row, text)
			})
			printFormattedRow(row, &markdownTable)
		}
	})

	return markdownTable.String(), nil
}
*/
