package helpers

import (
	"cli-top/ac"
	"cli-top/types"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

func GetSemDetails(cookies types.Cookies, regNo string) types.SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"

	//fmt.Println(regNo, cookies)
	bodyText, err := FetchReq(regNo, cookies, url, "")
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
	ac.FindAndSaveSemIds(doc, "form-select", &tempIds)
	//fmt.Printf("%q", tempIds)
	SemIds = ac.RemoveEmptyStrings(tempIds)
	//fmt.Printf("%q", SemIds)

	// fmt.Printf("%q\n", SemIds)

	for _, semId := range SemIds {
		optionText := ac.FindOptionWithTagValue(doc, semId)

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
	markdownTable := ac.GenerateSemDetailsMarkdownTable(semDetails)

	// Render Markdown using glamour
	rendered, err := glamour.Render(markdownTable, "dark")
	if err != nil {
		fmt.Println("Error rendering Markdown:", err)
		return
	}

	// Print the rendered Markdown
	fmt.Println(rendered)
}
