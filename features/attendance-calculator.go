// features/attendance-calculator.go
package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	AttendanceTableSelector = "table#AttendanceDetailDataTable"
	AttendanceRowsSelector  = "tbody tr"
	AttendanceCellSelector  = "td"
)

var (
	reSubjectName = regexp.MustCompile(`-\s*(.*?)\s*-`)
	reSubjectType = regexp.MustCompile(`[^-]*$`)
	reProfessor   = regexp.MustCompile(`^(.*?)\s*-\s*`)
)

func GetAttendance(regNo string, cookies types.Cookies, sem_choice int) {
	if !helpers.ValidateLogin(cookies) {
		return
	}
	url := "https://vtop.vit.ac.in/vtop/processViewStudentAttendance"

	semDetails, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			fmt.Printf("Error fetching semesters: %v\n", err)
		}
		fmt.Println("Please login using the cli-top login command.")
		return
	}

	if len(semDetails) == 0 {
		fmt.Println("No semesters found.")
		return
	}

	var semID string
	var attendanceList [][]string
	found := false

	// Iterate from the latest semester to the earliest
	for i := len(semDetails) - 1; i >= 0; i-- {
		semID = semDetails[i].SemID
		bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
		if err != nil {
			if debug.Debug {
				fmt.Printf("Error fetching attendance for Semester %s: %v\n", semDetails[i].SemName, err)
			}
			continue // Try the previous semester
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
		if err != nil {
			if debug.Debug {
				fmt.Printf("Error parsing HTML document for Semester %s: %v\n", semDetails[i].SemName, err)
			}
			continue // Try the previous semester
		}

		attendanceList = findAndSaveAttendance(doc)

		// Check if attendance data exists (more than header row)
		if len(attendanceList) > 1 {
			found = true
			if debug.Debug {
				fmt.Printf("Selected Semester: %s (%s)\n", semDetails[i].SemName, semID)
			}
			break
		} else {
			if debug.Debug {
				fmt.Printf("No attendance data found for Semester: %s (%s). Trying previous semester.\n", semDetails[i].SemName, semID)
			}
		}
	}

	// If no attendance data found in any semester
	if !found {
		fmt.Println("No attendance data available in any semester.")
		return
	}

	fmt.Println()
	helpers.PrintTable(attendanceList, 1)
	fmt.Println()
}

func findAndSaveAttendance(doc *goquery.Document) [][]string {
	var attendanceList [][]string
	attendanceList = append(attendanceList, []string{"Subject", "Type", "Faculty Name", "Classes Attended", "Percentage", "75% Alert"})

	table := doc.Find(AttendanceTableSelector)
	if table.Length() > 0 {
		table.Find(AttendanceRowsSelector).Each(func(i int, rowSelection *goquery.Selection) {
			sub_name_and_type := rowSelection.Find(AttendanceCellSelector).Eq(2).Find("span").Text()
			var sub_name string
			var sub_type string
			proff := rowSelection.Find(AttendanceCellSelector).Eq(4).Find("span").Text()
			attended := rowSelection.Find(AttendanceCellSelector).Eq(5).Find("span").Text()
			total := rowSelection.Find(AttendanceCellSelector).Eq(6).Find("span").Text()
			percent := rowSelection.Find(AttendanceCellSelector).Eq(7).Find("span").Find("span").Text()

			// Extract Subject Name
			match := reSubjectName.FindStringSubmatch(sub_name_and_type)
			if len(match) > 1 {
				sub_name = strings.TrimSpace(match[1])
			}

			// Extract Subject Type
			matchType := reSubjectType.FindString(sub_name_and_type)
			sub_type = strings.TrimSpace(matchType)

			// Normalize Faculty Name
			matchProf := reProfessor.FindStringSubmatch(proff)

			if len(matchProf) > 1 {
				caser := cases.Title(language.English)
				proff = caser.String(strings.ToLower(matchProf[1]))
			}

			// Classes Attended
			classes_attended := attended + "/" + total

			// Convert attended and total to integers
			attendedInt, err1 := strconv.Atoi(attended)
			totalInt, err2 := strconv.Atoi(total)
			if err1 != nil || err2 != nil {
				if debug.Debug {
					fmt.Printf("Error converting attendance numbers for subject %s: attended='%s', total='%s'\n", sub_name, attended, total)
				}
				return
			}

			// Calculate 75% Alert
			var missOrAttend string
			if sub_type == "Lab Only" || sub_type == "Embedded Lab" {
				attendedInt = attendedInt / 2
				totalInt = totalInt / 2
				missOrAttend = calculateAttendance(attendedInt, totalInt, 1)
			} else {
				missOrAttend = calculateAttendance(attendedInt, totalInt, 0)
			}

			attendanceList = append(attendanceList, []string{sub_name, sub_type, proff, classes_attended, percent, missOrAttend})
		})
	} else {
		if debug.Debug {
			fmt.Println("Table with ID 'AttendanceDetailDataTable' not found.")
		}
		fmt.Println("No attendance table found for the selected semester.")
	}

	return attendanceList
}

func calculateAttendance(attended, total, classtype int) string {
	// Calculate how many more classes need to be attended to meet 74.01% attendance
	targetAttendance := 0.7401
	neededAttendance := targetAttendance * float64(total)

	// If the current attendance is already below the target
	if float64(attended) < neededAttendance {
		// Calculate the exact number of additional classes required to meet 74.01%
		x := (neededAttendance - float64(attended)) / (1 - targetAttendance)
		x = math.Ceil(x) // Round up to ensure they meet the target after attending whole classes
		if classtype == 1 {
			return fmt.Sprintf("\033[31mAttend %d more lab(s)\033[0m", int(x))
		} else {
			return fmt.Sprintf("\033[31mAttend %d more class(es)\033[0m", int(x))
		}
	} else {
		// If already at or above the target, calculate how many can be missed
		canMiss := int(math.Floor((float64(attended) - neededAttendance) / targetAttendance))
		if classtype == 1 {
			return fmt.Sprintf("\033[32mCan miss %d lab(s)\033[0m", canMiss)
		} else {
			return fmt.Sprintf("\033[32mCan miss %d class(es)\033[0m", canMiss)
		}
	}
}
