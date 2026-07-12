// features/attendance-calculator.go
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

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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

type AttendanceRecord struct {
	Subject         string
	Type            string
	FacultyName     string
	ClassesAttended string
	Percentage      string
	Alert           string
	CanMiss         int
	NeedsAttend     int
	IsLab           bool
}

func GetAttendance(regNo string, cookies types.Cookies, sem_choice int) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	semDetails, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			helpers.Printf("Error fetching semesters: %v\n", err)
		}
		helpers.Println("Please login using the cli-top login command.")
		return
	}

	if len(semDetails) == 0 {
		helpers.Println("No semesters found.")
		return
	}

	candidates, err := semesterCandidates(semDetails, sem_choice)
	if err != nil {
		helpers.Println("Invalid semester selection.")
		return
	}

	var attendanceRecords []AttendanceRecord
	for _, semester := range candidates {
		attendanceRecords, err = fetchAttendanceRecordsForSemester(regNo, cookies, semester.SemID)
		if err != nil {
			if debug.Debug {
				helpers.Printf("Error fetching attendance for Semester %s: %v\n", semester.SemName, err)
			}
			continue
		}

		if len(attendanceRecords) > 0 {
			if debug.Debug {
				helpers.Printf("Selected Semester: %s (%s)\n", semester.SemName, semester.SemID)
			}
			break
		}

		if debug.Debug {
			helpers.Printf("No attendance data found for Semester: %s (%s).\n", semester.SemName, semester.SemID)
		}
	}

	if len(attendanceRecords) == 0 {
		if sem_choice > 0 {
			helpers.Println("No attendance data available for the selected semester.")
		} else {
			helpers.Println("No attendance data available in any semester.")
		}
		return
	}

	helpers.Println()
	helpers.PrintTable(attendanceRecordsToTable(attendanceRecords), 1)
	helpers.Println()
}

func semesterCandidates(semesters []types.Semester, semChoice int) ([]types.Semester, error) {
	if semChoice < 0 || semChoice > len(semesters) {
		return nil, fmt.Errorf("invalid semester selection")
	}
	if semChoice > 0 {
		return []types.Semester{semesters[semChoice-1]}, nil
	}

	candidates := make([]types.Semester, len(semesters))
	for i := range semesters {
		candidates[i] = semesters[len(semesters)-1-i]
	}
	return candidates, nil
}

func attendanceRecordsToTable(records []AttendanceRecord) [][]string {
	attendanceList := [][]string{{"Subject", "Type", "Faculty Name", "Classes Attended", "Percentage", "75% Alert"}}
	for _, record := range records {
		attendanceList = append(attendanceList, []string{
			record.Subject,
			record.Type,
			record.FacultyName,
			record.ClassesAttended,
			record.Percentage,
			record.Alert,
		})
	}
	return attendanceList
}

func ExtractAttendanceRecords(doc *goquery.Document) []AttendanceRecord {
	var records []AttendanceRecord
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
					helpers.Printf("Error converting attendance numbers for subject %s: attended='%s', total='%s'\n", sub_name, attended, total)
				}
				return
			}

			isLab := sub_type == "Lab Only" || sub_type == "Embedded Lab"
			status := calculateAttendanceStatus(attendedInt, totalInt, isLab)

			records = append(records, AttendanceRecord{
				Subject:         sub_name,
				Type:            sub_type,
				FacultyName:     proff,
				ClassesAttended: classes_attended,
				Percentage:      percent,
				Alert:           status.Display,
				CanMiss:         status.CanMiss,
				NeedsAttend:     status.NeedsAttend,
				IsLab:           isLab,
			})
		})
	} else {
		if debug.Debug {
			helpers.Println("Table with ID 'AttendanceDetailDataTable' not found.")
		}
	}

	return records
}

type attendanceStatus struct {
	Display     string
	CanMiss     int
	NeedsAttend int
}

func calculateAttendanceStatus(attended, total int, isLab bool) attendanceStatus {
	if isLab {
		attended = attended / 2
		total = total / 2
	}

	// Calculate how many more classes need to be attended to meet 74.01% attendance
	targetAttendance := 0.7401
	neededAttendance := targetAttendance * float64(total)

	// If the current attendance is already below the target
	if float64(attended) < neededAttendance {
		// Calculate the exact number of additional classes required to meet 74.01%
		x := (neededAttendance - float64(attended)) / (1 - targetAttendance)
		x = math.Ceil(x) // Round up to ensure they meet the target after attending whole classes
		if isLab {
			return attendanceStatus{
				Display:     fmt.Sprintf("\033[31mAttend %d more lab(s)\033[0m", int(x)),
				NeedsAttend: int(x),
			}
		} else {
			return attendanceStatus{
				Display:     fmt.Sprintf("\033[31mAttend %d more class(es)\033[0m", int(x)),
				NeedsAttend: int(x),
			}
		}
	} else {
		// If already at or above the target, calculate how many can be missed
		canMiss := int(math.Floor((float64(attended) - neededAttendance) / targetAttendance))
		if isLab {
			return attendanceStatus{
				Display: fmt.Sprintf("\033[32mCan miss %d lab(s)\033[0m", canMiss),
				CanMiss: canMiss,
			}
		} else {
			return attendanceStatus{
				Display: fmt.Sprintf("\033[32mCan miss %d class(es)\033[0m", canMiss),
				CanMiss: canMiss,
			}
		}
	}
}

func fetchAttendanceRecordsForSemester(regNo string, cookies types.Cookies, semID string) ([]AttendanceRecord, error) {
	url := "https://vtop.vit.ac.in/vtop/processViewStudentAttendance"
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		return nil, err
	}

	return ExtractAttendanceRecords(doc), nil
}
