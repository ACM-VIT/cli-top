// features/marks.go

package features

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"vtop-cli/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	"golang.org/x/net/html"
)

type SemesterDetails struct {
	SemNames []string
	SemIds   []string
}

func GetSemDetails(cookies types.Cookies, regNo string) SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentTimeTable"

	bodyText, err := fetchReq(regNo, cookies, url, "")
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

func PrintSemDetails(regNo string, cookies types.Cookies) {
	semDetails := GetSemDetails(cookies, regNo)

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

func GetMarks(regNo string, cookies types.Cookies, semID string) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doStudentMarkView"

	bodyText, err := fetchReq(regNo, cookies, url, semID)
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
		code := td.Eq(1).Text()
		subject := td.Eq(2).Text()
		name := td.Eq(3).Text()
		ctype := td.Eq(4).Text()
		fac := td.Eq(6).Text()
		slot := td.Eq(7).Text()
		// Add more lines as needed for other columns

		// Print or use the extracted data
		detail := fmt.Sprintf("## CourseCode: %s, CourseTitle: %s,  CourseType: %s, Faculty: %s, Slot: %s, ClassNbr: %s\n", subject, name, ctype, fac, slot, code)
		// Print or use other extracted data as needed

		details = append(details, detail)
	})
	return details
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
	builder.WriteString(fmt.Sprintf("| %-5s | %-31s | %-7s | %-10s | %-7s | %-10s | %-13s |\n",
		"Index", "Title", "MaxMark", "Weightage%", "Status", "ScoredMark", "WeightageMark"))
	builder.WriteString("|-------|---------------------------------|---------|------------|---------|------------|---------------|\n")
}

func printFormattedRow(row []string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-5s | %-31s | %-7s | %-10s | %-7s | %-10s | %-13s |\n",
		row[0], row[1], row[2], row[3], row[4], row[5], row[6]))
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

func Marks(regNo string, cookies types.Cookies, sem_choice int) {
	selectedSemId := ""
	selectedSemName := ""

	if sem_choice != 0 {
		// Validate the provided choice
		choice := sem_choice
		semDetails := GetSemDetails(cookies, regNo)

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

		PrintSemDetails(regNo, cookies)

		var choice int
		fmt.Print("\nEnter the index of the semester to view marks: ")
		fmt.Scanln(&choice)

		// Validate the choice
		semDetails := GetSemDetails(cookies, regNo)
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

	GetMarks(regNo, cookies, selectedSemId)
}
