package features

import (
	"bytes"
	"cli-top/types"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

func fetchReq0(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
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

func fetchReq20(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
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

func GetSemDetails0(cookies types.Cookies, regNo string) types.SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"

	//fmt.Println(regNo, cookies)
	bodyText, err := fetchReq0(regNo, cookies, url, "")
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

func PrintSemDetails0(regNo string, cookies types.Cookies) {
	semDetails := GetSemDetails0(cookies, regNo)
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

func GetExamSchedule(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/examinations/doSearchExamScheduleForStudent"
	selID := Marks0(regNo, cookies, sem_choice)

	bodyText, err := fetchReq20(regNo, cookies, url, selID)
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
	examSchedule, err := findAndSaveAtten0(doc)
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

func findAndSaveAtten0(doc *goquery.Document) ([][]string, error) {
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
			examSchedule = append(examSchedule, row)
		})
	} else {
		return nil, fmt.Errorf("Table with ID '%s' not found", targetID)
	}

	return examSchedule, nil
}

func generateExamScheduleMarkdownTable(examSchedule [][]string) string {
	var buf bytes.Buffer
	// Iterate through exam schedule and generate rows
	for i, row := range examSchedule {
		if i != 0 {
			if len(row) >= 13 {
				if row[0] == "1" {
					title := examSchedule[0]
					fmt.Printf("| %-5s | %-11s | %-45s | %-11s | %-11s | %-14s | %-19s | %-7s | %-7s | %-8s | %-15s |\n", title[0], title[1][7:], title[2], title[5], title[6], title[8], title[9], title[10], title[11][:4], title[12], "Days Remaining")
					fmt.Printf("|%-7s|%-13s|%-47s|%-13s|%-13s|%-16s|%-21s|%-9s|%-9s|%-10s|%-15s|\n", strings.Repeat("-", 7),
						strings.Repeat("-", 13), strings.Repeat("-", 47), strings.Repeat("-", 13), strings.Repeat("-", 13), strings.Repeat("-", 16), strings.Repeat("-", 21),
						strings.Repeat("-", 9), strings.Repeat("-", 9), strings.Repeat("-", 10), strings.Repeat("-", 17))
				}
				// Extract date from the exam date column
				datePart := row[6]

				// Calculate days remaining
				examDate, err := time.Parse("02-Jan-2006", datePart)
				if err != nil {
					fmt.Println("Error parsing exam date:", err)
					continue
				}
				daysRemaining := int(examDate.Sub(time.Now()).Hours() / 24)

				// Check if the exam is completed
				daysRemainingText := ""
				if daysRemaining >= 0 {
					daysRemainingText = fmt.Sprintf("%-3ddays", daysRemaining)
				} else {
					daysRemainingText = "Exam Completed"
				}

				// Table row with days remaining
				fmt.Printf("| %-5s | %-11s | %-45s | %-11s | %-11s | %-14s | %-19s | %-7s | %-7s | %-8s | %-15s |\n", row[0], row[1], row[2], row[5], row[6], row[8], row[9], row[10], row[11], row[12], daysRemainingText)
			} else {
				fmt.Println()
				fmt.Println(row)
				fmt.Println()
			}
		}
	}

	return buf.String()
}








func Marks0(regNo string, cookies types.Cookies, sem_choice int) string {

	selectedSemId := ""
	selectedSemName := ""

	var choice int
	if sem_choice == 0 {
		PrintSemDetails(regNo, cookies)
		fmt.Print("\nEnter the index of the semester to view exam schedule: ")
		fmt.Scanln(&choice)
	} else {
		choice = sem_choice
	}
	semDet := GetSemDetails0(cookies, regNo)

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
