// features/marks.go

package features

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
	authID := "21BIT0151"
	csrf := "5ff29e3c-f47e-4dc9-a70e-4c78a1d0b9cf"
	jsessionID := "9A44D5A85C54D382FFF5C4ED5F6E9606"

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

	// Parse the HTML
	doc, err := html.Parse(strings.NewReader(string(bodyText)))
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

func findAndSaveSemIds(n *html.Node, targetClass string, result *[]string) {
	if n.Type == html.ElementNode && n.Data == "select" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, targetClass) {
				// Process the <select> element
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && c.Data == "option" {
						for _, optAttr := range c.Attr {
							if optAttr.Key == "value" {
								// Save the value attribute to the result slice
								*result = append(*result, optAttr.Val)
							}
						}
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		findAndSaveSemIds(c, targetClass, result)
	}
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

func findOptionWithTagValue(n *html.Node, targetValue string) string {
	if n.Type == html.ElementNode && n.Data == "option" {
		for _, attr := range n.Attr {
			if attr.Key == "value" && attr.Val == targetValue {
				// Found the <option> tag with the specified value, return its text content
				return getTextContent(n)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findOptionWithTagValue(c, targetValue); result != "" {
			return result
		}
	}

	return ""
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
	url := "https://vtop.vit.ac.in/vtop/examinations/doStudentMarkView"
	bodyText, err := fetchReq(url, semID)
	if err != nil {
		log.Fatal(err)
	}

	// Parse the HTML
	doc, err := html.Parse(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}

	// Find all elements with the specified class
	class := "customTable-level1"
	elements := findElementsByClass(doc, class)

	// Convert HTML elements to Markdown tables
	for _, element := range elements {
		// Convert each element to Markdown
		markdownTable, err := convertHTMLElementToMarkdown(element)
		if err != nil {
			log.Fatal(err)
		}

		// Print or process the Markdown table
		fmt.Println("Markdown Table:")
		fmt.Println(markdownTable)
	}

	// Print or process the found elements
	fmt.Printf("Number of elements found: %d\n", len(elements))

}

func convertHTMLElementToMarkdown(element *html.Node) (string, error) {
	var markdownTable string

	// Use goquery for easier HTML manipulation
	doc := goquery.NewDocumentFromNode(element)

	// Iterate over table rows
	doc.Find("tr").Each(func(_ int, rowSelection *goquery.Selection) {
		// Iterate over table cells in each row
		rowSelection.Find("td").Each(func(_ int, cellSelection *goquery.Selection) {
			// Extract and append cell text to the markdownTable string
			markdownTable += cellSelection.Text() + " | "
		})

		// Remove the trailing " | " and add a new line
		markdownTable = strings.TrimSuffix(markdownTable, " | ") + "\n"
	})

	return markdownTable, nil
}

func findElementsByClass(n *html.Node, class string) []*html.Node {
	var result []*html.Node

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && hasClass(n, class) {
			result = append(result, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)

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
		selectedSemId := semDetails.SemIds[selectedIndex]
		selectedSemName := semDetails.SemNames[selectedIndex]

		fmt.Printf("\nYou selected SemId: %s, SemName: %s\n", selectedSemId, selectedSemName)
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
		selectedSemId := semDetails.SemIds[selectedIndex]
		selectedSemName := semDetails.SemNames[selectedIndex]

		fmt.Printf("\nYou selected SemId: %s, SemName: %s\n", selectedSemId, selectedSemName)
	}
}
