package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var schedule = map[string]map[string][]string{
	"A1": {
		"Monday":    []string{"08:00", "08:50"},
		"Wednesday": []string{"09:00", "09:50"},
	},
	"B1": {
		"Tuesday":  []string{"08:00", "08:50"},
		"Thursday": []string{"09:00", "09:50"},
	},
	"C1": {
		"Wednesday": []string{"08:00", "08:50"},
		"Friday":    []string{"09:00", "09:50"},
	},
	"D1": {
		"Monday":   []string{"10:00", "10:50"},
		"Thursday": []string{"08:00", "08:50"},
	},
	"E1": {
		"Tuesday": []string{"10:00", "10:50"},
		"Friday":  []string{"08:00", "08:50"},
	},
	"F1": {
		"Monday":    []string{"09:00", "09:50"},
		"Wednesday": []string{"10:00", "10:50"},
	},
	"G1": {
		"Tuesday":  []string{"09:00", "09:50"},
		"Thursday": []string{"10:00", "10:50"},
	},
	"TA1": {
		"Friday": []string{"10:00", "10:50"},
	},
	"TB1": {
		"Monday": []string{"11:00", "11:50"},
	},
	"TC1": {
		"Tuesday": []string{"11:00", "11:50"},
	},
	"TD1": {
		"Friday": []string{"12:00", "12:50"},
	},
	"TE1": {
		"Thursday": []string{"11:00", "11:50"},
	},
	"TF1": {
		"Friday": []string{"11:00", "11:50"},
	},
	"TG1": {
		"Monday": []string{"12:00", "12:50"},
	},
	"TAA1": {
		"Tuesday": []string{"12:00", "12:50"},
	},
	"TCC1": {
		"Thursday": []string{"12:00", "12:50"},
	},
	"A2": {
		"Monday":    []string{"14:00", "14:50"},
		"Wednesday": []string{"15:00", "15:50"},
	},
	"B2": {
		"Tuesday":  []string{"14:00", "14:50"},
		"Thursday": []string{"15:00", "15:50"},
	},
	"C2": {
		"Wednesday": []string{"14:00", "14:50"},
		"Friday":    []string{"15:00", "15:50"},
	},
	"D2": {
		"Monday":   []string{"16:00", "16:50"},
		"Thursday": []string{"14:00", "14:50"},
	},
	"E2": {
		"Tuesday": []string{"16:00", "16:50"},
		"Friday":  []string{"14:00", "14:50"},
	},
	"F2": {
		"Monday":    []string{"15:00", "15:50"},
		"Wednesday": []string{"16:00", "16:50"},
	},
	"G2": {
		"Tuesday":  []string{"15:00", "15:50"},
		"Thursday": []string{"16:00", "16:50"},
	},
	"TA2": {
		"Friday": []string{"16:00", "16:50"},
	},
	"TB2": {
		"Monday": []string{"17:00", "17:50"},
	},
	"TC2": {
		"Tuesday": []string{"17:00", "17:50"},
	},
	"TD2": {
		"Friday": []string{"18:00", "18:50"},
	},
	"TE2": {
		"Thursday": []string{"17:00", "17:50"},
	},
	"TF2": {
		"Friday": []string{"17:00", "17:50"},
	},
	"TG2": {
		"Monday": []string{"18:00", "18:50"},
	},
	"TAA2": {
		"Tuesday": []string{"18:00", "18:50"},
	},
	"TBB2": {
		"Wednesday": []string{"18:00", "18:50"},
	},
	"TCC2": {
		"Thursday": []string{"18:00", "18:50"},
	},
	"TDD2": {
		"Friday": []string{"18:00", "18:50"},
	},
	"L1+L2": {
		"Monday": []string{"08:00", "09:40"},
	},
	"L3+L4": {
		"Monday": []string{"09:50", "11:30"},
	},
	"L5+L6": {
		"Monday": []string{"11:40", "13:20"},
	},
	"L7+L8": {
		"Tuesday": []string{"08:00", "09:40"},
	},
	"L9+L10": {
		"Tuesday": []string{"09:50", "11:30"},
	},
	"L11+L12": {
		"Tuesday": []string{"11:40", "13:20"},
	},
	"L13+L14": {
		"Wednesday": []string{"08:00", "09:40"},
	},
	"L15+L16": {
		"Wednesday": []string{"09:50", "11:30"},
	},
	"L17+L18": {
		"Wednesday": []string{"11:40", "13:20"},
	},
	"L19+L20": {
		"Thursday": []string{"08:00", "09:40"},
	},
	"L21+L22": {
		"Thursday": []string{"09:50", "11:30"},
	},
	"L23+L24": {
		"Thursday": []string{"11:40", "13:20"},
	},
	"L25+L26": {
		"Friday": []string{"08:00", "09:40"},
	},
	"L27+L28": {
		"Friday": []string{"09:50", "11:30"},
	},
	"L29+L30": {
		"Friday": []string{"11:40", "13:20"},
	},
	"L31+L32": {
		"Monday": []string{"14:00", "15:40"},
	},
	"L33+L34": {
		"Monday": []string{"15:50", "17:30"},
	},
	"L35+L36": {
		"Monday": []string{"17:40", "19:20"},
	},
	"L37+L38": {
		"Tuesday": []string{"14:00", "15:40"},
	},
	"L39+L40": {
		"Tuesday": []string{"15:50", "17:30"},
	},
	"L41+L42": {
		"Tuesday": []string{"17:40", "19:20"},
	},
	"L43+L44": {
		"Wednesday": []string{"14:00", "15:40"},
	},
	"L45+L46": {
		"Wednesday": []string{"15:50", "17:30"},
	},
	"L47+L48": {
		"Wednesday": []string{"17:40", "19:20"},
	},
	"L49+L50": {
		"Thursday": []string{"14:00", "15:40"},
	},
	"L51+L52": {
		"Thursday": []string{"15:50", "17:30"},
	},
	"L53+L54": {
		"Thursday": []string{"17:40", "19:20"},
	},
	"L55+L56": {
		"Friday": []string{"14:00", "15:40"},
	},
	"L57+L58": {
		"Friday": []string{"15:50", "17:30"},
	},
	"L59+L60": {
		"Friday": []string{"17:40", "19:20"},
	},
	"V1": {
		"Wednesday": []string{"11:00", "11:50"},
	},
	"V2": {
		"Wednesday": []string{"12:00", "12:50"},
	},
	"V3": {
		"Monday": []string{"19:00", "19:50"},
	},
	"V4": {
		"Tuesday": []string{"19:00", "19:50"},
	},
	"V5": {
		"Wednesday": []string{"19:00", "19:50"},
	},
	"V6": {
		"Thursday": []string{"19:00", "19:50"},
	},
	"V7": {
		"Friday": []string{"19:00", "19:50"},
	},
	"V8": {
		"Saturday": []string{"08:00", "08:50"},
	},
	"X11": {
		"Saturday": []string{"09:00", "09:50"},
		"Sunday":   []string{"11:00", "11:50"},
	},
	"X12": {
		"Saturday": []string{"10:00", "10:50"},
		"Sunday":   []string{"12:00", "12:50"},
	},
	"Y11": {
		"Saturday": []string{"11:00", "11:50"},
		"Sunday":   []string{"09:00", "09:50"},
	},
	"Y12": {
		"Saturday": []string{"12:00", "12:50"},
		"Sunday":   []string{"10:00", "10:50"},
	},
	"X21": {
		"Saturday": []string{"14:00", "14:50"},
		"Sunday":   []string{"16:00", "16:50"},
	},
	"Z21": {
		"Saturday": []string{"15:00", "15:50"},
		"Sunday":   []string{"15:00", "15:50"},
	},
	"Y21": {
		"Saturday": []string{"16:00", "16:50"},
		"Sunday":   []string{"14:00", "14:50"},
	},
	"W21": {
		"Saturday": []string{"17:00", "17:50"},
		"Sunday":   []string{"17:00", "17:50"},
	},
	"W22": {
		"Saturday": []string{"18:00", "18:50"},
		"Sunday":   []string{"18:00", "18:50"},
	},
	"V9": {
		"Saturday": []string{"19:00", "19:50"},
	},
	"V10": {
		"Sunday": []string{"08:00", "08:50"},
	},
	"V11": {
		"Sunday": []string{"19:00", "19:50"},
	},
	"L71+L72": {
		"Saturday": []string{"08:00", "09:40"},
	},
	"L73+L74": {
		"Saturday": []string{"09:50", "11:30"},
	},
	"L75+L76": {
		"Saturday": []string{"11:40", "13:20"},
	},
	"L77+L78": {
		"Saturday": []string{"14:00", "15:40"},
	},
	"L79+L80": {
		"Saturday": []string{"15:50", "17:30"},
	},
	"L81+L82": {
		"Saturday": []string{"17:40", "19:20"},
	},
	"L83+L84": {
		"Sunday": []string{"08:00", "09:40"},
	},
	"L85+L86": {
		"Sunday": []string{"09:50", "11:30"},
	},
	"L87+L88": {
		"Sunday": []string{"11:40", "13:20"},
	},
	"L89+L90": {
		"Sunday": []string{"14:00", "15:40"},
	},
	"L91+L92": {
		"Sunday": []string{"15:50", "17:30"},
	},
	"L93+L94": {
		"Sunday": []string{"17:40", "19:20"},
	},
}

func GetTimeTable(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		fmt.Println("Please login first using the cli-top login command")
		return
	}
	url := "https://vtop.vit.ac.in/vtop/processViewTimeTable"

	semester, err := helpers.SelectSemester(regNo, cookies, sem_choice)
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}

	bodyText, err := helpers.FetchReq(regNo, cookies, url, semester.SemID, "UTC", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	grp_list := getClassGroups(regNo, cookies, semester)
	datelist := getDateList(regNo, cookies, semester, grp_list[1][1])
	semSec, month, year := processDates(regNo, cookies, semester, grp_list[1][1], datelist, 0)
	courseMap := getCourseName(doc)
	timetable := makeTT(schedule, courseMap)
	printTT(timetable)

	// Create the Other Downloads/ICS File directory
	icsDir, err := helpers.GetOrCreateDownloadDir(filepath.Join("Other Downloads", "ICS File"))
	if err != nil {
		fmt.Println("Error creating ICS file directory:", err)
		return
	}

	icsFilename := "VITtimetable.ics"
	icsFilepath := filepath.Join(icsDir, icsFilename)
	icsContent := makeISC(timetable, semSec, month, year)
	err = writetoFile(icsFilepath, icsContent)
	if err != nil {
		fmt.Println("Error generating ICS file:", err)
	} else {
		serverURL := "https://cli-calendar.acmvit.in"
		uploadedFileURL, err := helpers.UploadICSFile(icsFilepath, serverURL)
		if err != nil {
			fmt.Println("Error uploading ICS file:", err)
			fmt.Println("Please import the 'VITtimetable.ics' file manually from your Downloads folder.")
		} else {
			fmt.Println()
			fmt.Println("ICS file generated and saved successfully.")
			helpers.GenerateCalendarImportLinks(uploadedFileURL, "Timetable")
		}
	}
}

func makeTT(schedule map[string]map[string][]string, courseMap map[string]types.SubjectTime) map[string][]types.Class {
	timetable := make(map[string][]types.Class)
	for key, value := range courseMap {
		for i := 0; i < len(value.Slot); i++ {
			for keyS, valueS := range schedule[value.Slot[i]] {
				timetable[keyS] = append(timetable[keyS], types.Class{
					Subject:   key,
					Slot:      value.Slot[i],
					Venue:     value.Venue,
					StartTime: valueS[0],
					EndTime:   valueS[1],
				})
			}
		}
	}
	return timetable
}

func makeISC(timetable map[string][]types.Class, semSection [][]int, startMonth int, startYear int) string {
	icsContent := "BEGIN:VCALENDAR\nVERSION:2.0\nCALSCALE:GREGORIAN\nX-WR-CALNAME:VIT timetable\n"
	startMonth++
	// Get the first day of the start month
	startDate := time.Date(startYear, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)

	for months := 0; months < len(semSection); months = months + 1 {
		for dayofMonth := 0; dayofMonth < len(semSection[months]); dayofMonth = dayofMonth + 1 {
			var day string
			if semSection[months][dayofMonth] == -1 {
				startDate = startDate.AddDate(0, 0, 1)
				continue
			} else if semSection[months][dayofMonth] == 0 {
				day = startDate.Format("Monday")
			} else {
				dayInt := semSection[months][dayofMonth]
				day = getDayName(time.Weekday(dayInt))
			}
			for _, class := range timetable[day] {
				startDateTime := fmt.Sprintf("%sT%s00", startDate.Format("20060102"),
					strings.ReplaceAll(class.StartTime, ":", ""))
				endDateTime := fmt.Sprintf("%sT%s00", startDate.Format("20060102"),
					strings.ReplaceAll(class.EndTime, ":", ""))
				icsContent += fmt.Sprintf("BEGIN:VEVENT\n"+
					"SUMMARY:%s\n"+
					"DTSTART;TZID=Asia/Kolkata:%s\n"+
					"DTEND;TZID=Asia/Kolkata:%s\n"+
					"LOCATION:%s\n"+
					"DESCRIPTION:Slot: %s\n"+
					"BEGIN:VALARM\n"+
					"TRIGGER:-PT5M\n"+
					"ACTION:DISPLAY\n"+
					"END:VALARM\n"+
					"END:VEVENT\n",
					class.Subject, startDateTime, endDateTime,
					class.Venue, class.Slot)
			}
			startDate = startDate.AddDate(0, 0, 1)
		}
	}
	icsContent += "END:VCALENDAR"
	return icsContent
}

func getDayName(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "Monday"
	case time.Tuesday:
		return "Tuesday"
	case time.Wednesday:
		return "Wednesday"
	case time.Thursday:
		return "Thursday"
	case time.Friday:
		return "Friday"
	case time.Saturday:
		return "Saturday"
	case time.Sunday:
		return "Sunday"
	default:
		return ""
	}
}

func writetoFile(filepath string, content string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create ICS file: %v", err)
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write ICS content: %v", err)
	}
	return nil
}

func getCourseName(doc *goquery.Document) map[string]types.SubjectTime {
	courseMap := make(map[string]types.SubjectTime)
	table := doc.Find("table.table")

	if table.Length() > 0 {
		table.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
			courseCell := row.Find("td").Eq(2) // Third column (index 2)
			cellText := strings.TrimSpace(courseCell.Text())
			parts := strings.SplitN(cellText, " - ", 2)
			var courseName string
			if len(parts) == 2 {
				courseName = strings.TrimSpace(parts[1])
				// Check for text within parentheses
				if idxStart := strings.Index(courseName, "("); idxStart != -1 {
					idxEnd := strings.Index(courseName, ")")
					if idxEnd != -1 && idxEnd > idxStart {
						parenthetical := courseName[idxStart+1 : idxEnd]
						if strings.Contains(strings.ToLower(parenthetical), "embedded") {
							courseName = strings.TrimSpace(courseName[:idxStart]) + strings.TrimSpace(courseName[idxStart-1:])
						} else {
							courseName = strings.TrimSpace(courseName[:idxStart])
						}
					}
				}
			}
			slotCell := row.Find("td").Eq(7)
			slotText := strings.TrimSpace(slotCell.Text())
			parts = strings.SplitN(slotText, " - ", 2)
			newparts := strings.Split(parts[0], "+")
			if len(slotText) == 0 {
				return
			}
			if slotText[0] == 'L' {
				var joinedParts []string
				for i := 0; i < len(newparts); i += 2 {
					joinedParts = append(joinedParts, strings.Join([]string{newparts[i], newparts[i+1]}, "+"))
				}
				newparts = joinedParts
			}
			sub := types.SubjectTime{
				Slot:  newparts,
				Venue: strings.TrimSpace(parts[1]),
			}
			courseMap[courseName] = sub
		})
	} else {
		fmt.Println("Table with class 'table' not found")
	}
	return courseMap
}

func printTT(timetable map[string][]types.Class) {
	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	for _, day := range daysOfWeek {
		classes, exists := timetable[day]
		if exists {
			fmt.Printf("%s\n\n", day)
			sort.Slice(classes, func(i, j int) bool {
				return classes[i].StartTime < classes[j].StartTime
			})
			tableData := [][]string{
				{"Time", "Subject", "Slot", "Venue"},
			}
			for _, class := range classes {
				tableData = append(tableData, []string{
					fmt.Sprintf("%s-%s", class.StartTime, class.EndTime),
					class.Subject,
					class.Slot,
					class.Venue,
				})
			}
			helpers.PrintTable(tableData, 0)
			fmt.Println()
		}
	}
}
