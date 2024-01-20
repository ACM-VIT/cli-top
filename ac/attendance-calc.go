package ac

import (
	"bytes"
	"cli-top/types"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	payload := fmt.Sprintf("authorizedID=%s&_csrf=%s&semesterSubId=%s&x=%s", regNo, cookies.CSRF, semID, time.Now().UTC().Format(time.RFC1123))
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

func GetAttendance(regNo string, cookies types.Cookies, semId string) {

	url := "https://vtop.vit.ac.in/vtop/processViewStudentAttendance"

	sel_id := Marks(regNo, cookies, 0)
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
	// 	Atten = RemoveEmptyStrings(temp)
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
		printTable("Header", nil, &markdownTable)
		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
			row := []string{} // Initialize a new slice for each row
			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				row = append(row, text)
			})
			//fmt.Println("hi table")
			//fmt.Println(row)
			printFormattedRow(row, &markdownTable)
		})
	} else {
		fmt.Println("Table with ID 'AttendanceDetailDataTable' not found")
	}
	fmt.Println(markdownTable.String())
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

func printTable(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-27s | %-16s | %-10s | %-18s |\n",
		"S.No.", "Course Code", "Slot No.", "Faculty Name", "Classes Attended", "Percentage", "75% Alert"))
	builder.WriteString("|-------|--------------|----------------------|-----------------------------|------------------|------------|--------------------|\n")

}

func printFormattedRow(row []string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-27s | %-16s | %-10s | %-17s |\n",
		row[0], strings.Split(row[2], "-")[0], strings.Split(row[3], "-")[1], strings.Split(row[4], "-")[0], row[5]+"/"+row[6], row[7], Cal75(strToInt(row[5]), strToInt(row[6]), strToInt(strings.Split(row[7], "%")[0]))))
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

	var choice int
	fmt.Print("\nEnter the index of the semester to view attendance: ")
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

	fmt.Print(output)

	fmt.Println()

	// Marks(regNo,cookies,sem_choice)
	// GetAttendance(regNo, cookies, selectedSemId)
	// fmt.Println("hii")

	return selectedSemId

}
