package features

import (
	//"cli-top/types"

	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	//"golang.org/x/net/html"
)

func GetSemDetailsAtten(cookies types.Cookies, regNo string) SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"

	//fmt.Println(regNo, cookies)
	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", "", "POST")
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
	findAndSaveSemIds(doc, "form-select", &tempIds)
	//fmt.Printf("%q", tempIds)
	SemIds = removeEmptyStrings(tempIds)
	//fmt.Printf("%q", SemIds)

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

	//fmt.Println("\nget wroking")
	// Return the encapsulated struct
	return SemesterDetails{
		SemNames: SemNames,
		SemIds:   SemIds,
	}
}

func GetAttendance(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/processViewStudentAttendance"

	semesterID := Attendance(regNo, cookies, sem_choice)
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semesterID, "UTC", "POST")
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}
	findAndSaveAtten(doc)
}

func findAndSaveAtten(doc *goquery.Document) {
	var markdownTable strings.Builder
	targetID := "AttendanceDetailDataTable"
	table := doc.Find("table#" + targetID)
	if table.Length() > 0 {
		printTableAtten("Header", nil, &markdownTable)
		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
			row := []string{} // Initialize a new slice for each row
			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				row = append(row, text)
			})
			//fmt.Println("hi table")
			//fmt.Println(row)
			printFormattedRowAtten(row, &markdownTable)
		})
	} else {
		fmt.Println("Table with ID 'AttendanceDetailDataTable' not found")
	}
	fmt.Println(markdownTable.String())
}

func printTableAtten(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-35s | %-16s | %-10s | %-18s |\n",
		"S.No.", "Course Code", "Slot No.", "Faculty Name", "Classes Attended", "Percentage", "75% Alert"))
	builder.WriteString("|-------|--------------|----------------------|-------------------------------------|------------------|------------|--------------------|\n")

}

func printFormattedRowAtten(row []string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-35s | %-16s | %-10s | %-17s |\n",
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
		ret = fmt.Sprintf("\033[32m" + "Can skip 0 class" + "\033[0m" + "\t")
	} else if perc < 75 {
		for i := 1; i < att; i++ {
			if math.Ceil((float64(att+i)/float64(tot+i))*100) <= 75 {
				ret = fmt.Sprintf("\033[31m"+"Attend %d class\033[0m"+"\033[0m"+"\t", i)
			}
		}
	} else {
		ret = fmt.Sprintf("\033[32m" + "Can skip 0 class" + "\033[0m" + "\t")
		for i := 1; i < att; i++ {
			if math.Ceil((float64(att)/float64(tot+i))*100) >= 75 {

				ret = fmt.Sprintf("\033[32m"+"Can skip %d class"+"\033[0m"+"\t", i)

			}

		}

	}
	return ret
}

func Attendance(regNo string, cookies types.Cookies, sem_choice int) string {
	selectedSemId := ""
	selectedSemName := ""
	var choice int

	if sem_choice == 0 {
		PrintSemDetails(regNo, cookies)
		fmt.Print("\nEnter the index of the semester to view attendance: ")
		fmt.Scanln(&choice)
	} else {
		choice = sem_choice
	}

	semDet := GetSemDetailsAtten(cookies, regNo)
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
