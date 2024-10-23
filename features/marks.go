package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var scoredWeightageMarksSum float64       // Total sum of weightage marks
var maxMarksSum int // Total sum of weightage percentage

func GetMarks(regNo string, cookies types.Cookies, semID string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doStudentMarkView"
	semester,err := helpers.SelectSemester(regNo, cookies, sem_choice)
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}

	payload := fmt.Sprintf("------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"authorizedID\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"semesterSubId\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"_csrf\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J--\r\n", regNo, semester.SemID, cookies.CSRF)

	bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, payload, "POST", "marks")
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
		in := "No Data Found"
		// Use ANSI escape codes to format the message
		out := fmt.Sprintf("\033[1;31m%s\033[0m", in) // Bold red text
		fmt.Println(out)
		return
	}
	// Convert HTML elements to Markdown tables
	for i, element := range elements {
		// Convert each element to Markdown
		//markdownTable, err := convertHTMLElementToMarkdown(element)
		OneSubTable,weightageMark,maxMarkSum := ExtractMarks(element)
		if err != nil && debug.Debug {
			fmt.Println(OneSubTable)
			fmt.Println(err)
		}
		if ((len(OneSubTable) == 1) || (len(OneSubTable) == 0)) {
			fmt.Println("No Data Found for", subjectDetails[i])
			return
		}
		// for _,stuff :=range OneSubTable {

		// }
		course_detail := "\033[1;34m" + subjectDetails[i] + "\033[0m"
		fmt.Println(course_detail)
		helpers.PrintTable(OneSubTable,1)
		weightageMarkStr := "\033[32m" + fmt.Sprintf("%.2f", weightageMark) + "\033[0m"
		maxMarkSumStr := "\033[32m" + strconv.Itoa(maxMarkSum) + "\033[0m"
		fmt.Println(weightageMarkStr+"/"+maxMarkSumStr)
		fmt.Println()
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
		subject := td.Eq(2).Text()
		name := td.Eq(3).Text()
		ctype := td.Eq(4).Text()
		fac := td.Eq(6).Text()
		slot := td.Eq(7).Text()
		// Add more lines as needed for other columns

		// Print or use the extracted data
		detail := fmt.Sprintf("# CourseCode: %s, CourseTitle: %s,  CourseType: %s, Faculty: %s, Slot: %s\n", subject, name, ctype, fac, slot)
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
		}
	})

	return markdownTable.String(), nil
}

func printTableMarks(title string, data [][]string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-5s | %-45s | %-7s | %-10s | %-7s | %-10s | %-13s |\n",
		"Index", "Title", "MaxMark", "Weightage%", "Status", "ScoredMark", "WeightageMark"))
	builder.WriteString("|-------|---------------------------------------------|---------|------------|---------|------------|---------------|\n")
}

// func printFormattedRowMarks(row []string, builder *strings.Builder) {
// 	weightage, err := strconv.ParseFloat(row[6], 3)
// 	if err != nil && debug.Debug {
// 		fmt.Print("Error converting weightage to float:", err)
// 	}
// 	weightagePercentage, err := strconv.ParseInt(row[3], 10, 64)
// 	if err != nil && debug.Debug {
// 		fmt.Print("Error converting weightage Percentage to int")
// 	}

// 	weightageSum += weightage
// 	weightagePercentageSum += int(weightagePercentage)
// 	builder.WriteString(fmt.Sprintf("| %-5s | %-45s | %-7s | %-10s | %-7s | %-10s | %-13s |\n",
// 		row[0], row[1], row[2], row[3], row[4], row[5], row[6]))
// }

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

func ExtractMarks (element *goquery.Selection) ([][]string,float64,int) {
	// Find all elements with the specified class
	var SingleSubTable [][]string 
	weightageMarkSum := 0.0
	maxSubjectMarksSum := 0
	//SingleSubTable = append(SingleSubTable, []string{"Title", "MaxMark", "Weightage%", "Status", "ScoredMark", "WeightageMark"})
	element.Find("tbody tr").Each(func(_ int, rowSelection *goquery.Selection) {
		title := strings.TrimSpace(rowSelection.Find("td").Eq(1).Text())
		maxMark := strings.TrimSpace(rowSelection.Find("td").Eq(2).Text())
		weightage := strings.TrimSpace(rowSelection.Find("td").Eq(3).Text())
		status := strings.TrimSpace(rowSelection.Find("td").Eq(4).Text())
		scoredMark := strings.TrimSpace(rowSelection.Find("td").Eq(5).Text())
		weightageMark := strings.TrimSpace(rowSelection.Find("td").Eq(6).Text())
		SingleSubTable = append(SingleSubTable, []string{title, maxMark, weightage, status, scoredMark, weightageMark})
		maxMarkInt, err := strconv.Atoi(weightage)
        if err == nil {
            maxSubjectMarksSum = maxSubjectMarksSum + maxMarkInt
        } 
		weightageFloat, err := strconv.ParseFloat(weightageMark, 64)
        if err == nil {
            weightageMarkSum = weightageMarkSum + weightageFloat
        } 	
	})

	return SingleSubTable,weightageMarkSum,maxSubjectMarksSum
}
