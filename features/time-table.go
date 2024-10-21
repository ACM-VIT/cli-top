package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func GetTimeTable(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/processViewTimeTable"

	schedule := "{(1, '08:00'): ['A1', 'L1'], (1, '09:00'): ['F1'], (1, '10:00'): ['D1'], (1, '11:00'): ['TB1'], (1, '12:00'): ['TG1'], (2, '08:00'): ['B1', 'L7'], (2, '09:00'): ['G1'], (2, '10:00'): ['E1'], (2, '11:00'): ['TC1'], (2, '12:00'): ['TAA1'], (3, '08:00'): ['C1', 'L13'], (3, '09:00'): ['A1'], (3, '10:00'): ['F1'], (3, '11:00'): ['TD1'], (3, '12:00'): ['TBB1'], (4, '08:00'): ['D1', 'L19'], (4, '09:00'): ['B1'], (4, '10:00'): ['G1'], (4, '11:00'): ['TE1'], (4, '12:00'): ['TCC1'], (5, '08:00'): ['E1', 'L25'], (5, '09:00'): ['C1'], (5, '10:00'): ['TA1'], (5, '11:00'): ['TF1'], (5, '12:00'): ['TD1'], (6, '08:00'): ['V8', 'L71'], (6, '09:00'): ['X11'], (6, '10:00'): ['X12'], (6, '11:00'): ['Y11'], (6, '12:00'): ['Y12'], (7, '08:00'): ['V10', 'L83'], (7, '09:00'): ['Y11'], (7, '10:00'): ['Y12'], (7, '11:00'): ['X11'], (7, '12:00'): ['X12'],(1, '14:00'): ['A2', 'L31'], (1, '15:00'): ['F2'], (1, '16:00'): ['D2'], (1, '17:00'): ['TB2'], (1, '18:00'): ['TG2'], (2, '14:00'): ['B2', 'L37'], (2, '15:00'): ['G2'], (2, '16:00'): ['E2'], (2, '17:00'): ['TC2'], (2, '18:00'): ['TAA2'], (3, '14:00'): ['C2', 'L43'], (3, '15:00'): ['A2'], (3, '16:00'): ['F2'], (3, '17:00'): ['TD2'], (3, '18:00'): ['TBB2'], (4, '14:00'): ['D2', 'L49'], (4, '15:00'): ['B2'], (4, '16:00'): ['G2'], (4, '17:00'): ['TE2'], (4, '18:00'): ['TCC2'], (5, '14:00'): ['E2', 'L55'], (5, '15:00'): ['C2'], (5, '16:00'): ['TA2'], (5, '17:00'): ['TF2'], (5, '18:00'): ['TD2'], (6, '14:00'): ['X21', 'L77'], (6, '15:00'): ['Z21'], (6, '16:00'): ['Y21'], (6, '17:00'): ['W21'], (6, '18:00'): ['W22'], (7, '14:00'): ['Y21', 'L89'], (7, '15:00'): ['Z21'], (7, '16:00'): ['X21'], (7, '17:00'): ['W21'], (7, '18:00'): ['W22'],(1, '09:50'): ['L3'], (1, '11:40'): ['L5'], (2, '09:50'): ['L9'], (2, '11:40'): ['L11'], (3, '09:50'): ['L15'], (3, '11:40'): ['L17'], (4, '09:50'): ['L21'], (4, '11:40'): ['L23'], (5, '09:50'): ['L27'], (5, '11:40'): ['L29'], (6, '09:50'): ['L73'], (6, '11:40'): ['L75'], (7, '09:50'): ['L85'], (7, '11:40'): ['L87'], (1, '15:50'): ['L33'], (1, '17:40'): ['L35'], (2, '15:50'): ['L39'], (2, '17:40'): ['L41'], (3, '15:50'): ['L45'], (3, '17:40'): ['L47'], (4, '15:50'): ['L51'], (4, '17:40'): ['L53'], (5, '15:50'): ['L57'], (5, '17:40'): ['L59'], (6, '15:50'): ['L79'], (6, '17:40'): ['L81'], (7, '15:50'): ['L91'], (7, '17:40'): ['L93']}"
	
	semesterID := helpers.SelectSemester(regNo, cookies, sem_choice)

	bodyText, err := helpers.FetchReq(regNo, cookies, url, semesterID, "UTC", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	findAndSaveTimeTable(doc, schedule)
}

func parsePythonDict(schedule string) map[types.KeyStruct][]string {
	pythonDict := make(map[types.KeyStruct][]string)

	// Removing unnecessary characters and splitting the schedule string
	schedule = strings.ReplaceAll(schedule, "{", "")
	schedule = strings.ReplaceAll(schedule, "}", "")
	entries := strings.Split(schedule, "],")

	// Iterating through entries to populate the pythonDict
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.Split(entry, ": [")
		if len(parts) != 2 {
			continue
		}

		keyPart := strings.TrimSpace(parts[0])
		valuePart := strings.TrimSpace(parts[1])

		// Extracting group and time from keyPart
		var group int
		var time string
		fmt.Sscanf(keyPart, "(%d, '%s')", &group, &time)

		// Creating KeyStruct
		key := types.KeyStruct{Group: group, Time: strings.TrimSuffix(time, "')")}

		// Extracting values and populating pythonDict
		values := strings.Split(strings.Trim(valuePart, "[]"), ", ")

		// Removing single quotes and any trailing ')'
		for i := range values {
			values[i] = strings.TrimSuffix(strings.Trim(values[i], "'"), "')")
		}

		pythonDict[key] = values
	}

	return pythonDict
}
func checkTime(goMap map[int][][]string, pythonDict map[types.KeyStruct][]string) {
	var sortedKeys []int
	for key := range goMap {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Ints(sortedKeys)

	for _, key := range sortedKeys {
		value := goMap[key]

		dayName := ""
		switch key {
		case 1:
			dayName = "Monday"
		case 2:
			dayName = "Tuesday"
		case 3:
			dayName = "Wednesday"
		case 4:
			dayName = "Thursday"
		case 5:
			dayName = "Friday"
		case 6:
			dayName = "Saturday"
		case 7:
			dayName = "Sunday"
		}

		fmt.Printf("\033[1m%s\033[0m\n\n", dayName)

		// Create a slice to hold the timetable entries for sorting
		var dayEntries []string

		for _, row := range value {
			for i := range row {
				if row[i] != "null" {
					slot := strings.Split(row[i], "-")[0]
					courseCode := strings.Split(row[i], "-")[1]
					venue := strings.Split(row[i], "-")[3]

					for keyPy, valuePy := range pythonDict {
						if key == keyPy.Group {
							for j := range valuePy {
								if valuePy[j] == slot {
									// Determine the end time for the slot
									hour, _ := strconv.Atoi(strings.Split(keyPy.Time, ":")[0])
									min, _ := strconv.Atoi(strings.Split(keyPy.Time, ":")[1])
									if slot[0] == 'L' {
										min += 100
										for min >= 60 {
											min -= 60
											hour++
										}
									} else {
										min += 50
										for min >= 60 {
											min -= 60
											hour++
										}
									}
									end := fmt.Sprintf("%02d:%02d", hour, min)

									// Create an entry string for sorting
									entry := fmt.Sprintf("%s to %s\t| %s\t\t| %s\t| %s\t\t", keyPy.Time, end, slot, courseCode, venue)
									dayEntries = append(dayEntries, entry)
								}
							}
						}
					}
				}
			}
		}

		// Sort the entries by start time
		sort.Slice(dayEntries, func(i, j int) bool {
			timeI := strings.Split(dayEntries[i], " to ")[0]
			timeJ := strings.Split(dayEntries[j], " to ")[0]
			return timeI < timeJ
		})

		// Print the sorted timetable entries for the current day
		for _, entry := range dayEntries {
			fmt.Println(entry)
		}
		fmt.Println() // Add a newline to separate days
	}
}



func findAndSaveTimeTable(doc *goquery.Document, schedule string) {

	//fmt.Println("printing inside func")
	//fmt.Println(schedule)
	pythonDict := parsePythonDict(schedule)
	//fmt.Println(pythonDict)

	targetID := "timeTableStyle"
	table := doc.Find("table#" + targetID)
	if table.Length() > 0 {
		rows := [][]string{}
		//timeL := [][]string{}
		//timeTh := [][]string{}
		var indices []int // Move the indices declaration outside the inner scope
		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {

			row := []string{} // Initialize a new slice for each row
			//timeLab := []string{}
			//timeTheory := []string{}
			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
				bgcolor, exists := cell.Attr("bgcolor")
				if exists && bgcolor == "#FC6C85" {
					text := strings.TrimSpace(cell.Text())
					row = append(row, text)

					indices = append(indices, j)
				} else {
					row = append(row, "null")
				}
			})
			if len(row) > 0 {
				rows = append(rows, row)
			}
			

		})

		// Print sub-rows after all rows have been processed
		subject := [][]string{}
		//var count int
		count := 0
		for i, subRow := range rows {
			if i > 3 && i<17 {
				subject = append(subject, subRow)
				count += 1
			}
			
			//fmt.Println()
		}
		fmt.Println(rows)

		var subjectDayWise map[int][][]string
		subjectDayWise = make(map[int][][]string)

		// Divide the subject array into groups of 2 subarrays
		groupSize := 2
		key := 1

		for i, subRow := range subject {
			subjectDayWise[key] = append(subjectDayWise[key], subRow)

			// Check if the current group is complete
			if (i+1)%groupSize == 0 {
				key++
			}
		}
		checkTime(subjectDayWise, pythonDict)

	} else {
		fmt.Println("Table with ID 'timeTableStyle' not found")
	}
}

// Utility function to get the index of a slot (e.g., F1 -> 1)
func getSlotIndex(slotName string) int {
	if len(slotName) > 1 {
		return int(slotName[1] - '0')
	}
	return 0
}

func separateArray(arr []int) [][]int {
	var result [][]int
	start := 0
	for i := 1; i < len(arr); i++ {
		if arr[i] < arr[i-1] {
			result = append(result, arr[start:i])
			start = i
		}
	}
	// Add the remaining elements if any
	if start < len(arr) {
		result = append(result, arr[start:])
	}
	return result
}

func printTableTimeTable(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-10s | %-50s | %-25s | %-21s | %-5s | %-6s |\n",
		"S.No.", "Course Code", "Course Title", "Course Type", "Credits", " Total", "Grade"))
	builder.WriteString("|       |             |                                                    |                           |-----------------------|        |        |\n")
	//builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")

	builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-7s| %-6s |\n",
		"", "", "", "", "L", "P", "J", "C", "", ""))
	builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")
}

func printFormattedRowTimeTable(row []string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s |\n", row[0]))
	//fmt.Println("hi")
}
