// features/marks.go

package features

import (
	"fmt"
	"log"
	"strings"
	"vtop-cli/helpers"
	types "vtop-cli/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

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
	elements := helpers.FindElementsByClass(doc, class)

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
				helpers.PrintTable("Header", nil, &markdownTable) // Print header only once
				tableStarted = true
			}

			row := []string{}
			rowSelection.Find("td").Each(func(_ int, cellSelection *goquery.Selection) {
				// Extract and append cell text to the markdownTable string
				text := strings.TrimSpace(cellSelection.Text())
				row = append(row, text)
			})
			helpers.PrintFormattedRow(row, &markdownTable, 1) //broken
		}
	})

	return markdownTable.String(), nil
}

func Marks(regNo string, cookies types.Cookies, sem_choice int) {
	selectedSemId := ""
	selectedSemName := ""

	if sem_choice != 0 {
		// Validate the provided choice
		choice := sem_choice
		semDetails := helpers.GetSemDetails(cookies, regNo)

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

		helpers.PrintSemDetails(regNo, cookies)

		var choice int
		fmt.Print("\nEnter the index of the semester to view marks: ")
		fmt.Scanln(&choice)

		// Validate the choice
		semDetails := helpers.GetSemDetails(cookies, regNo)
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
