package helpers

import (
	"cli-top/debug"
	"cli-top/types"
	"fmt"
	"strings"
	"bufio"
	"os"

	"github.com/PuerkitoBio/goquery"
)

func FindAndSaveSemIds(doc *goquery.Document, targetClass string, result *[]string) {
	doc.Find("select."+targetClass+" option").Each(func(i int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists {
			*result = append(*result, value)
		}
	})
}

func GetSemDetails(cookies types.Cookies, regNo string) types.SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"

	bodyText, err := FetchReq(regNo, cookies, url, "", "", "POST", "application/x-www-form-urlencoded")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching semester details:", err)
			fmt.Printf("Response Body: %s\n", string(bodyText))
		}
		return types.SemesterDetails{}
	}

	if strings.Contains(string(bodyText), "HTTP Status 404") {
		fmt.Println("Received 404 Not Found. Please try logging in again.")
		return types.SemesterDetails{}
	}

	if debug.Debug {
		fmt.Println("---- Response Body Start ----")
		fmt.Println(string(bodyText))
		fmt.Println("---- Response Body End ----")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing the HTML document:", err)
		}
		return types.SemesterDetails{}
	}

	var SemNames []string
	var SemIds []string

	var tempIds []string
	FindAndSaveSemIds(doc, "form-select", &tempIds)
	SemIds = RemoveEmptyStrings(tempIds)

	for _, semId := range SemIds {
		optionText := FindOptionWithTagValue(doc, semId)
		if optionText != "" {
			SemNames = append(SemNames, optionText)
		} else {
			SemNames = append(SemNames, "Unknown")
		}
	}

	return types.SemesterDetails{
		SemNames: SemNames,
		SemIds:   SemIds,
	}
}

func SelectSemester(regNo string, cookies types.Cookies, sem_choice int) string {
	semDetails := GetSemDetails(cookies, regNo)
	if len(semDetails.SemIds) == 0 {
		fmt.Println("Error fetching semester details or no semesters available. Try logging out and logging back in.")
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
		semDetails.SemIds[i-1], semDetails.SemIds[j-1] = semDetails.SemIds[j-1], semDetails.SemIds[i-1]
		semDetails.SemNames[i-1], semDetails.SemNames[j-1] = semDetails.SemNames[j-1], semDetails.SemNames[i-1]
	}

	choice := TableSelector("semester", nested_sem_list, sem_choice)

	if choice == -1 {
		fmt.Println("Problem in selecting semester")
		clearInputBuffer()
		return ""
	}

	if choice < 1 || choice > len(semDetails.SemIds) {
		fmt.Println("Invalid semester selection.")
		clearInputBuffer()
		return ""
	}

	selectedSemId = semDetails.SemIds[choice-1]

	clearInputBuffer()

	return selectedSemId
}

func clearInputBuffer() {
	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadString('\n')
	if err != nil && debug.Debug {
		fmt.Println("Error clearing input buffer:", err)
	}
}
