package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func GetAttendance(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/processViewStudentAttendance"
	semDetails, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			fmt.Printf("Error fetching semesters: %v\n", err)
		} else {
			fmt.Println(err)
			return
		}

	}
	if len(semDetails) == 0 {
		fmt.Println("No semesters found.")
		return
	}
	semID := semDetails[len(semDetails)-1].SemID
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	attendenceList := findAndSaveAttendance(doc)
	fmt.Println()
	helpers.PrintTable(attendenceList, 1)
	fmt.Println()
}

func findAndSaveAttendance(doc *goquery.Document) [][]string {
	var attendanceList [][]string
	attendanceList = append(attendanceList, []string{"Subject", "Type", "Faculty Name", "Classes Attended", "Percentage", "75% Alert"})
	// var markdownTable strings.Builder
	table := doc.Find("table#AttendanceDetailDataTable")
	if table.Length() > 0 {
		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
			sub_name_and_type := rowSelection.Find("td").Eq(2).Find("span").Text()
			var sub_name string
			var sub_type string
			proff := rowSelection.Find("td").Eq(4).Find("span").Text()
			attended := rowSelection.Find("td").Eq(5).Find("span").Text()
			total := rowSelection.Find("td").Eq(6).Find("span").Text()
			percent := rowSelection.Find("td").Eq(7).Find("span").Find("span").Text()
			reSub := regexp.MustCompile(`-\s*(.*?)\s*-`)
			match := reSub.FindStringSubmatch(sub_name_and_type)
			if len(match) > 1 {
				sub_name = strings.TrimSpace(match[1])
			}
			reSubType := regexp.MustCompile(`[^-]*$`)
			matchType := reSubType.FindString(sub_name_and_type)
			sub_type = strings.TrimSpace(matchType)

			reProf := regexp.MustCompile(`^(.*?)\s*-\s*`)
			matchProf := reProf.FindStringSubmatch(proff)
			if len(matchProf) > 1 {
				proff = strings.Title(strings.ToLower(matchProf[1]))
			}
			classes_attended := attended + "/" + total
			attendedInt, _ := strconv.Atoi(attended)
			totalInt, _ := strconv.Atoi(total)
			var missOrAttend string
			if sub_type == "Lab Only" || sub_type == "Embedded Lab" {
				attendedInt = attendedInt/2
				totalInt = totalInt/2
				missOrAttend = calculateAttendance(attendedInt, totalInt, 1)
			} else {
				missOrAttend = calculateAttendance(attendedInt, totalInt, 0)
			}
			attendanceList = append(attendanceList, []string{sub_name, sub_type, proff, classes_attended, percent, missOrAttend})
		})
	} else {
		fmt.Println("Table with ID 'AttendanceDetailDataTable' not found")
	}
	// fmt.Println(markdownTable.String())
	return attendanceList
}

func calculateAttendance(attended, total, classtype int) string {
	// Calculate how many more classes need to be attended to meet 74.01% attendance
	targetAttendance := 0.7401
	neededAttendance := targetAttendance * float64(total)
	// If the current attendance is already below the target
	if float64(attended) < neededAttendance {
		// Calculate the exact number of additional classes required to meet 74.01%
		x := (targetAttendance*float64(total) - float64(attended)) / (1 - targetAttendance)
		x = math.Ceil(x) // Round up to ensure they meet the target after attending whole classes
		if classtype == 1 {
			return fmt.Sprintf("\033[31mAttend %d more lab(s)\033[0m", int(x))
		} else {
			return fmt.Sprintf("\033[31mAttend %d more class(es)\033[0m", int(x))
		}
	} else {
		// If already at or above the target, calculate how many can be missed
		canMiss := int(math.Ceil(float64(attended - int(math.Ceil(neededAttendance)))/(targetAttendance)))
		if classtype == 1 {
			return fmt.Sprintf("\033[32mCan miss %d lab(s)\033[0m", canMiss)
		} else {
			return fmt.Sprintf("\033[32mCan miss %d class(es)\033[0m", canMiss)
		}
	}
}
