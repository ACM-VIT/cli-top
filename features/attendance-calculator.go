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
	//"golang.org/x/net/html"
)

func GetAttendance(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/processViewStudentAttendance"

	semesterID := helpers.SelectSemester(regNo, cookies, sem_choice)
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semesterID, "UTC", "POST", "")
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}
	findAndSaveAttendance(doc)
}

func findAndSaveAttendance(doc *goquery.Document) {
	var markdownTable strings.Builder
	targetID := "AttendanceDetailDataTable"
	table := doc.Find("table#" + targetID)
	if table.Length() > 0 {
		printTableAttendance("Header", nil, &markdownTable)
		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
			row := []string{} // Initialize a new slice for each row
			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				row = append(row, text)
			})
			//fmt.Println("hi table")
			//fmt.Println(row)
			printFormattedRowAttendance(row, &markdownTable)
		})
	} else {
		fmt.Println("Table with ID 'AttendanceDetailDataTable' not found")
	}
	fmt.Println(markdownTable.String())
}

func printTableAttendance(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-35s | %-16s | %-10s | %-22s |\n",
		"S.No.", "Course Code", "Slot No.", "Faculty Name", "Classes Attended", "Percentage", "75% Alert"))
	builder.WriteString("|-------|--------------|----------------------|-------------------------------------|------------------|------------|------------------------|\n")

}

// <<<<<<< fix-fetchRequest
// func printFormattedRowAttendance(row []string, builder *strings.Builder) {
// 	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-35s | %-16s | %-10s | %-17s |\n",
// 		row[0], strings.Split(row[2], "-")[0], strings.Split(row[3], "-")[1], strings.Split(row[4], "-")[0], row[5]+"/"+row[6], row[7], Cal75(strToInt(row[5]), strToInt(row[6]), strToInt(strings.Split(row[7], "%")[0]))))
// =======
func printFormattedRowAtten(row []string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-35s | %-16s | %-10s | %-15s |\n",
		row[0], strings.Split(row[2], "-")[0], strings.Split(row[3], "-")[1], strings.Split(row[4], "-")[0], row[5]+"/"+row[6], row[7], Cal75(strToInt(row[5]), strToInt(row[6]))))

// >>>>>>> dev
}

func strToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		fmt.Println("Error converting string to integer:", err)

	}

	return num
}

func Cal75(att int, tot int) string {
	var ret string
	perc := float64(att) / float64(tot) * 100
	if perc >=74.01 && perc<=75 {
		ret = fmt.Sprintf("%-31s", "\033[32mCan skip 0 classes\033[0m")
	} else if perc < 74.01 {
		for i := 1; i <= (tot * 2); i++ {
			newPerc := float64(att+i) / float64(tot+i) * 100
			if math.Ceil(newPerc) >= 75 {
				ret = fmt.Sprintf("%-31s", fmt.Sprintf("\033[31mAttend %d class(es)\033[0m", i))
				break
			}
		}
	} else {
		for i := 1; i < tot; i++ {
			if math.Ceil((float64(att)/float64(tot+i))*100) >= 75 {
				ret = fmt.Sprintf("%-31s", fmt.Sprintf("\033[32mCan skip %d classes\033[0m", i))
			}
		}
	}

	return ret
}
// <<<<<<< fix-fetchRequest
// =======

// func Attendance(regNo string, cookies types.Cookies, sem_choice int) string {
// 	selectedSemId := ""
// 	selectedSemName := ""
// 	var choice int

// 	if sem_choice == 0 {
// 		PrintSemDetails(regNo, cookies)
// 		fmt.Print("\nEnter the index of the semester to view attendance: ")
// 		fmt.Scanln(&choice)
// 	} else {
// 		choice = sem_choice
// 	}

// 	semDet := GetSemDetailsAtten(cookies, regNo)
// 	if choice < 1 || choice > len(semDet.SemIds) {
// 		fmt.Println("Invalid choice.")
// 	} else {
// 		for i, id := range semDet.SemIds {
// 			if i+1 == choice {
// 				// fmt.Println("Selected sem id : ",id)
// 				// fmt.Println("Selected sem name : ",semDet.SemNames[i])
// 				selectedSemId = id
// 				selectedSemName = semDet.SemNames[i]
// 			}
// 		}
// 	}

// 	// Format the string with glamour
// 	formattedSelection := fmt.Sprintf("\n# You selected SemId: %s, SemName: %s\n", selectedSemId, selectedSemName)

// 	// Render and print the formatted string
// 	renderer, err := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(150))
// 	if err != nil {
// 		log.Fatal("Error creating glamour renderer:", err)
// 	}

// 	output, err := renderer.Render(formattedSelection)
// 	if err != nil {
// 		log.Fatal("Error rendering formatted string:", err)
// 	}

// 	fmt.Print(output)

// 	fmt.Println()

// 	return selectedSemId

// }
// >>>>>>> dev
