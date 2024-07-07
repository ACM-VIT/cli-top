// features/marks.go

package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	"golang.org/x/net/html"
)

var weightageSum float64       // Total sum of weightage marks
var weightagePercentageSum int // Total sum of weightage percentage

func GetMarks(regNo string, cookies types.Cookies, semID string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doStudentMarkView"

	semesterID := helpers.SelectSemester(regNo, cookies, sem_choice)

	payload := fmt.Sprintf("------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"authorizedID\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"semesterSubId\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"_csrf\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J--\r\n", regNo, semesterID, cookies.CSRF)

	bodyText, err := helpers.FetchReq(regNo, cookies, url, semesterID, payload, "POST", "marks")
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
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
		weightageSum = 0
		weightagePercentageSum = 0
		markdownTable, err := convertHTMLElementToMarkdown(element)
		if err != nil && debug.Debug {
			fmt.Println(err)
		}

		renderer, e := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(150))
		if e != nil {
			fmt.Println("Error rendering markdown:", err)
			return
		}
		markdown, err := renderer.Render(markdownTable)
		if err != nil && debug.Debug {
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
		formattedWeightageSum := fmt.Sprintf("%.1f", weightageSum)
		// calculate  percentage
		percentage := float64(weightageSum) / float64(weightagePercentageSum) * 100
		fmt.Println(cal50(formattedWeightageSum, percentage))
	}
}

// Formats the color of the result
func cal50(formattedWeightageSum string, percentage float64) string {
	result := ""
	green := fmt.Sprintf("You scored: "+"\033[32m"+"%s/%d"+"\033[0m"+"\t", formattedWeightageSum, weightagePercentageSum)
	red := fmt.Sprintf("You scored: "+"\033[31m"+"%s/%d"+"\033[0m"+"\t", formattedWeightageSum, weightagePercentageSum)
	if percentage >= 50 {
		result = green
	} else {
		result = red
	}
	return result
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
				printTableMarks("Header", nil, &markdownTable) // Print header only once
				tableStarted = true
			}

			row := []string{}
			rowSelection.Find("td").Each(func(_ int, cellSelection *goquery.Selection) {
				// Extract and append cell text to the markdownTable string
				text := strings.TrimSpace(cellSelection.Text())
				row = append(row, text)
			})
			printFormattedRowMarks(row, &markdownTable)
		}
	})

	return markdownTable.String(), nil
}

func printTableMarks(title string, data [][]string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-5s | %-45s | %-7s | %-10s | %-7s | %-10s | %-13s |\n",
		"Index", "Title", "MaxMark", "Weightage%", "Status", "ScoredMark", "WeightageMark"))
	builder.WriteString("|-------|---------------------------------------------|---------|------------|---------|------------|---------------|\n")
}

func printFormattedRowMarks(row []string, builder *strings.Builder) {
	weightage, err := strconv.ParseFloat(row[6], 3)
	if err != nil && debug.Debug {
		fmt.Print("Error converting weightage to float:", err)
	}
	weightagePercentage, err := strconv.ParseInt(row[3], 10, 64)
	if err != nil && debug.Debug {
		fmt.Print("Error converting weightage Percentage to int")
	}

	weightageSum += weightage
	weightagePercentageSum += int(weightagePercentage)
	builder.WriteString(fmt.Sprintf("| %-5s | %-45s | %-7s | %-10s | %-7s | %-10s | %-13s |\n",
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
