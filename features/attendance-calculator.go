package features

import (
	//"cli-top/types"

	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
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
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
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

func printFormattedRowAttendance(row []string, builder *strings.Builder) {
	SlotNo := strings.Split(row[3], "-")[1]

	var calResult string
	//To check for Lab slots
	if strings.ContainsAny(SlotNo, "L") {
		calResult = Cal75Lab(strToInt(row[5]), strToInt(row[6]), strToInt(strings.Split(row[7], "%")[0]))
	} else {
		calResult = Cal75(strToInt(row[5]), strToInt(row[6]))
	}
	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-35s | %-16s | %-10s | %-17s |\n",
		row[0], strings.Split(row[2], "-")[0], SlotNo, strings.Split(row[4], "-")[0], row[5]+"/"+row[6], row[7], calResult))
}

func Cal75Lab(att int, tot int, perc int) string {
	var ret string
	if perc == 75 {
		ret = fmt.Sprintf("%-31s", "\033[32mCan skip 0 labs\033[0m")
	} else if perc < 75 {
		for i := 1; i < att; i++ {
			if math.Ceil((float64(att+i)/float64(tot+i))*100) <= 75 {
				ret = fmt.Sprintf("%-31s", fmt.Sprintf("\033[31mAttend %d lab(s)\033[0m", i/2))
			}
		}
	} else {
		ret = fmt.Sprintf("%-31s", "\033[32mCan skip 0 lab\033[0m")
		for i := 1; i < att; i++ {
			if math.Ceil((float64(att)/float64(tot+i))*100) >= 75 {
				ret = fmt.Sprintf("%-31s", fmt.Sprintf("\033[32mCan skip %d lab(s)\033[0m", i/2))
			}
		}
	}
	return ret
}

func strToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil && debug.Debug {
		fmt.Println("Error converting string to integer:", err)

	}

	return num
}

func Cal75(att int, tot int) string {
	var ret string
	perc := float64(att) / float64(tot) * 100
	if perc >= 74.01 && perc <= 75 {
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
		for i := 0; i <= tot; i++ { // Start from 0 to check if no classes can be skipped
			newPerc := math.Ceil((float64(att) / float64(tot+i)) * 100)
			if newPerc >= 75 {
				ret = fmt.Sprintf("%-31s", fmt.Sprintf("\033[32mCan skip %d class(es)\033[0m", i))
			} else {
				// When the attendance percentage is no longer >= 75, break out of the loop
				break
			}
		}
	}

	return ret
}
