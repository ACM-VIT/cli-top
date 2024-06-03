package helpers

import (
	"bytes"
	"cli-top/types"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

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

func GetSemDetails(cookies types.Cookies, regNo string) types.SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"

	//fmt.Println(regNo, cookies)
	bodyText, err := FetchReq(regNo, cookies, url, "", "", "POST", "")
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

func SelectSemester(regNo string, cookies types.Cookies, sem_choice int) string {

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

	return selectedSemId
}
