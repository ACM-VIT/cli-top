package examschedule

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	"golang.org/x/net/html"
)

type SemesterDetails struct {
	SemNames []string
	SemIds   []string
}

func fetchReq(target_url string, semID string) ([]byte, error) {
	client := &http.Client{}

	//variables that need to be set with every request
	authID := "22BCI0001"
	csrf := "27a145c6-69dd-4f15-9f02-65a5d8f648d0"
	jsessionID := "0A14B27E06F45745AB0283D9DABFA449"

	var data = strings.NewReader(fmt.Sprintf("------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"authorizedID\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"semesterSubId\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"_csrf\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J--\r\n", authID, semID, csrf))
	req, err := http.NewRequest("POST", target_url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("authority", "vtop.vit.ac.in")
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.6")
	req.Header.Set("content-type", "multipart/form-data; boundary=----WebKitFormBoundary9yjNZXu7BBjgQK7J")
	req.Header.Set("cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=s2", jsessionID))
	req.Header.Set("origin", "https://vtop.vit.ac.in")
	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/content?")
	req.Header.Set("sec-ch-ua", `"Brave";v="119", "Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("sec-ch-ua-platform", `"Android"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-gpc", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Mobile Safari/537.36")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bodyText, nil
}

func GetSemDetails() SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentTimeTable"
	bodyText, err := fetchReq(url, "")
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}

	// Create a slice to store the extracted data
	var SemNames []string
	var SemIds []string

	var tempIds []string
	// Find and save the semester IDs
	findAndSaveSemIds(doc, "form-select", &tempIds)

	SemIds = removeEmptyStrings(tempIds)

	// fmt.Printf("%q\n", SemIds)

	for _, semId := range SemIds {
		optionText := findOptionWithTagValue(doc, semId)

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

	// Return the encapsulated struct
	return SemesterDetails{
		SemNames: SemNames,
		SemIds:   SemIds,
	}

}

func PrintSemDetails() {
	semDetails := GetSemDetails()

	if len(semDetails.SemIds) == 0 {
		fmt.Println("Error fetching semester details or no semesters available.")
		return
	}

	// Generate Markdown table text
	markdownTable := generateSemDetailsMarkdownTable(semDetails)

	// Render Markdown using glamour
	rendered, err := glamour.Render(markdownTable, "dark")
	if err != nil {
		fmt.Println("Error rendering Markdown:", err)
		return
	}

	// Print the rendered Markdown
	fmt.Println(rendered)
}

func generateSemDetailsMarkdownTable(semDetails SemesterDetails) string {
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

func findAndSaveSemIds(doc *goquery.Document, targetClass string, result *[]string) {
	doc.Find("select." + targetClass + " option").Each(func(i int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists {
			*result = append(*result, value)
		}
	})
}

func removeEmptyStrings(data []string) []string {
	var cleanedData []string
	for _, item := range data {
		if item != "" {
			cleanedData = append(cleanedData, item)
		}
	}
	return cleanedData
}

func findOptionWithTagValue(doc *goquery.Document, targetValue string) string {
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

func GetMarks(semID string) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doSearchExamScheduleForStudent"
	bodyText, err := fetchReq(url, semID)
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}

	subjectDetails := subjectDetails(doc)

	// Find all elements with the specified class
	class := "customTable-level1"
	elements := findElementsByClass(doc, class)

	// Check if elements were found
	if len(elements) == 0 {
		fmt.Println()
		in := `# No Data Found`

		out, _ := glamour.Render(in, "dark")
		fmt.Print(out)
		return
	}

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

		// Check if there are enough elements in the row
		if td.Length() >= 13 {
			sNo := getCellValue(td, 0)
			courseCode := getCellValue(td, 1)
			courseTitle := getCellValue(td, 2)
			courseType := getCellValue(td, 3)
			classID := getCellValue(td, 4)
			slot := getCellValue(td, 5)
			examDate := getCellValue(td, 6)
			examSession := getCellValue(td, 7)
			reportingTime := getCellValue(td, 8)
			examTime := getCellValue(td, 9)
			venue := getCellValue(td, 10)
			seatLocation := getCellValue(td, 11)
			seatNo := getCellValue(td, 12)

			// Add additional data fields here
			// Example:
			// additionalField1 := getCellValue(td, 13)
			// additionalField2 := getCellValue(td, 14)

			// Print or use the extracted data
			detail := fmt.Sprintf("## S.No.: %s, Course Code: %s, Course Title: %s, Course Type: %s, Class ID: %s, Slot: %s, Exam Date: %s, Exam Session: %s, Reporting Time: %s, Exam Time: %s, Venue: %s, Seat Location: %s, Seat No: %s\n",
				sNo, courseCode, courseTitle, courseType, classID, slot, examDate, examSession, reportingTime, examTime, venue, seatLocation, seatNo)

			// Add additional data fields to the detail string
			// Example:
			// detail += fmt.Sprintf("Additional Field 1: %s, Additional Field 2: %s\n", additionalField1, additionalField2)

			details = append(details, detail)
		} else {
			// Handle the case where there are not enough elements
			fmt.Printf("Warning: Insufficient elements in row %d (expected at least 13, found %d)\n", i, td.Length())

			// Optionally, you can add a default value or handle this case in another way
			// For now, let's add an empty detail for rows with insufficient elements
			details = append(details, "## Insufficient Data for this row\n")
		}
	})
	return details
}

func getCellValue(td *goquery.Selection, index int) string {
	if td.Length() > index {
		return td.Eq(index).Text()
	}
	return ""
}

func convertHTMLElementToMarkdown(element *goquery.Selection) (string, error) {
	var markdownTable strings.Builder
	var tableStarted bool

	// Use goquery for easier HTML manipulation
	// doc := goquery.NewDocumentFromNode(element)

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

func printTable(title string, data [][]string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-3s | %-8s | %-31s | %-3s | %-16s | %-10s | %-20s |%-5s |%-10s |%-20s |%-8s |%-8s |%-3s |\n",
		"S.No", "Course Code", "Course Title", "Course Type", "	Class ID", "Slot", "Exam Date", "Exam Session", "Reporting Time", "Exam Time", "Venue", "Seat Location", "Seat No"))
	builder.WriteString("|-------|----------------  |---------------|-------------|------|-----------|--------------|-------------------|-----------|---------|---------------|----------|\n")
}

func printFormattedRow(row []string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-3s | %-8s | %-31s | %-3s | %-16s | %-10s | %-20s |%-5s |%-10s |%-20s |%-8s |%-3s |\n",
		row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8], row[9], row[10], row[11]))
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

func Marks(sem_choice int) {
	selectedSemId := ""
	selectedSemName := ""

	if sem_choice != 0 {
		// Validate the provided choice
		choice := sem_choice
		semDetails := GetSemDetails()

		if choice < 1 || choice > len(semDetails.SemIds) {
			fmt.Println("Invalid choice. Please select a valid index.")
			return
		}

		// Display the selected semester details
		selectedIndex := choice - 1
		selectedSemId = semDetails.SemIds[selectedIndex]
		selectedSemName = semDetails.SemNames[selectedIndex]

	} else {
		in := `# Select the semester to view the marks for:`

		out, _ := glamour.Render(in, "dark")
		fmt.Print(out)

		PrintSemDetails()

		var choice int
		fmt.Print("\nEnter the index of the semester to view marks: ")
		fmt.Scanln(&choice)

		// Validate the choice
		semDetails := GetSemDetails()
		if choice < 1 || choice > len(semDetails.SemIds) {
			fmt.Println("Invalid choice. Please select a valid index.")
			return
		}

		// Display the selected semester details
		selectedIndex := choice - 1
		selectedSemId = semDetails.SemIds[selectedIndex]
		selectedSemName = semDetails.SemNames[selectedIndex]

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

	GetMarks(selectedSemId)
}
