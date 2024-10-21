package helpers

import (
	"cli-top/debug"
	"cli-top/types"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
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

func SelectSemester(regNo string, cookies types.Cookies, sem_choice int) string {
	semDetails := GetSemDetails(cookies, regNo)
	if len(semDetails.SemIds) == 0 {
		fmt.Println("Error fetching semester details or no semesters available.")
		return ""
	}
	var selectedSemId string
	var nested_sem_list [][]string
	nested_sem_list = append(nested_sem_list, []string{"SemId", "SemName"})
	for i := 0; i < len(semDetails.SemIds); i++ {
		nested_sem_list = append(nested_sem_list, []string{semDetails.SemIds[i], semDetails.SemNames[i]})
	}

	for i, j := 1, len(nested_sem_list)-1; i < j; i, j = i+1, j-1 {
		nested_sem_list[i], nested_sem_list[j] = nested_sem_list[j], nested_sem_list[i]
	}	

	choice := TableSelector("semester", nested_sem_list, sem_choice)

	if choice == -1 {
		return "Problem in selecting semester"
	}

	selectedSemId = semDetails.SemIds[len(semDetails.SemIds)-choice] // reversed here too
	return selectedSemId
}
